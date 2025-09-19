use anyhow::{anyhow, Result};
use async_nats::{Client, Message, Subscriber};
use futures::StreamExt;
use serde::de::DeserializeOwned;
use serde::Serialize;
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::{mpsc, Mutex};
use tokio::time::timeout;
use tracing::{debug, error, info, warn};
use uuid::Uuid;

use crate::fixtures::*;

#[derive(Debug, Clone)]
pub struct NatsClient {
    client: Client,
    event_listeners: Arc<Mutex<HashMap<String, mpsc::UnboundedSender<Message>>>>,
}

impl NatsClient {
    pub async fn new(nats_url: &str) -> Result<Self> {
        let client = async_nats::connect(nats_url).await
            .map_err(|e| anyhow!("Failed to connect to NATS: {}", e))?;

        info!("Connected to NATS at {}", nats_url);

        Ok(Self {
            client,
            event_listeners: Arc::new(Mutex::new(HashMap::new())),
        })
    }

    /// Publish event to NATS subject
    pub async fn publish_event<T: Serialize>(&self, subject: &str, event: &T) -> Result<()> {
        let payload = serde_json::to_vec(event)?;
        
        self.client
            .publish(subject, payload.into())
            .await
            .map_err(|e| anyhow!("Failed to publish to {}: {}", subject, e))?;

        debug!("Published event to subject: {}", subject);
        Ok(())
    }

    /// Subscribe to NATS subject and collect events
    pub async fn subscribe_to_events(&self, subject: &str) -> Result<EventCollector> {
        let subscriber = self.client
            .subscribe(subject)
            .await
            .map_err(|e| anyhow!("Failed to subscribe to {}: {}", subject, e))?;

        info!("Subscribed to NATS subject: {}", subject);

        let (tx, rx) = mpsc::unbounded_channel();
        
        // Store the sender for cleanup
        {
            let mut listeners = self.event_listeners.lock().await;
            listeners.insert(subject.to_string(), tx.clone());
        }

        // Start message handler
        let tx_clone = tx;
        tokio::spawn(async move {
            let mut subscriber = subscriber;
            while let Some(message) = subscriber.next().await {
                if tx_clone.send(message).is_err() {
                    break; // Receiver dropped
                }
            }
        });

        Ok(EventCollector::new(subject.to_string(), rx))
    }

    /// Wait for a specific event type with timeout
    pub async fn wait_for_event<T: DeserializeOwned>(
        &self,
        subject: &str,
        timeout_duration: Duration,
    ) -> Result<T> {
        let mut collector = self.subscribe_to_events(subject).await?;
        
        timeout(timeout_duration, async {
            collector.wait_for_event::<T>().await
        }).await
        .map_err(|_| anyhow!("Timeout waiting for event on subject: {}", subject))?
    }

    /// Simulate external service events
    pub async fn simulate_order_assigned(&self, driver_id: Uuid, order_id: Uuid) -> Result<()> {
        let event = OrderAssignedEvent {
            event_type: "order.assigned".to_string(),
            order_id: order_id.to_string(),
            driver_id: driver_id.to_string(),
            customer_id: Uuid::new_v4().to_string(),
            pickup_location: LocationData {
                latitude: 55.7558,
                longitude: 37.6176,
                address: Some("Красная площадь, 1".to_string()),
            },
            dropoff_location: LocationData {
                latitude: 55.7520,
                longitude: 37.6175,
                address: Some("Манежная площадь, 1".to_string()),
            },
            estimated_fare: 250.0,
            estimated_distance: 2.5,
            estimated_duration: 15,
            priority: 1,
            timestamp: chrono::Utc::now(),
        };

        self.publish_event("order.assigned", &event).await
    }

    /// Simulate payment processed event
    pub async fn simulate_payment_processed(&self, driver_id: Uuid, order_id: Uuid, amount: f64) -> Result<()> {
        let event = serde_json::json!({
            "event_type": "payment.processed",
            "payment_id": Uuid::new_v4().to_string(),
            "order_id": order_id.to_string(),
            "driver_id": driver_id.to_string(),
            "amount": amount,
            "commission": amount * 0.2,
            "net_amount": amount * 0.8,
            "payment_method": "card",
            "status": "completed",
            "timestamp": chrono::Utc::now()
        });

        self.publish_event("payment.processed", &event).await
    }

    /// Simulate vehicle assignment
    pub async fn simulate_vehicle_assigned(&self, driver_id: Uuid, vehicle_id: Uuid) -> Result<()> {
        let event = serde_json::json!({
            "event_type": "vehicle.assigned",
            "vehicle_id": vehicle_id.to_string(),
            "driver_id": driver_id.to_string(),
            "vehicle_type": "sedan",
            "license_plate": "М123КХ77",
            "assigned_by": "fleet.manager",
            "rental_rate": 1500.0,
            "timestamp": chrono::Utc::now()
        });

        self.publish_event("vehicle.assigned", &event).await
    }

    /// Simulate customer rating
    pub async fn simulate_customer_rating(&self, driver_id: Uuid, order_id: Uuid, rating: i32) -> Result<()> {
        let event = serde_json::json!({
            "event_type": "customer.rated.driver",
            "rating_id": Uuid::new_v4().to_string(),
            "order_id": order_id.to_string(),
            "driver_id": driver_id.to_string(),
            "customer_id": Uuid::new_v4().to_string(),
            "rating": rating,
            "comment": "Отличный водитель!",
            "criteria": {
                "cleanliness": 5,
                "driving": rating,
                "punctuality": rating
            },
            "anonymous": false,
            "timestamp": chrono::Utc::now()
        });

        self.publish_event("customer.rated.driver", &event).await
    }

    /// Clean up event listeners
    pub async fn cleanup(&self) -> Result<()> {
        let mut listeners = self.event_listeners.lock().await;
        listeners.clear();
        Ok(())
    }
}

/// Event collector that receives and stores NATS messages
#[derive(Debug)]
pub struct EventCollector {
    subject: String,
    receiver: mpsc::UnboundedReceiver<Message>,
    collected_events: Vec<Message>,
}

impl EventCollector {
    fn new(subject: String, receiver: mpsc::UnboundedReceiver<Message>) -> Self {
        Self {
            subject,
            receiver,
            collected_events: Vec::new(),
        }
    }

    /// Wait for the next event
    pub async fn wait_for_event<T: DeserializeOwned>(&mut self) -> Result<T> {
        match self.receiver.recv().await {
            Some(message) => {
                debug!("Received event on subject '{}': {} bytes", self.subject, message.payload.len());
                self.collected_events.push(message.clone());
                
                let event: T = serde_json::from_slice(&message.payload)
                    .map_err(|e| anyhow!("Failed to deserialize event: {}", e))?;
                
                Ok(event)
            }
            None => Err(anyhow!("Event stream closed for subject: {}", self.subject))
        }
    }

    /// Collect events for a specific duration
    pub async fn collect_events_for_duration(&mut self, duration: Duration) -> Result<Vec<Message>> {
        let mut events = Vec::new();
        let deadline = tokio::time::Instant::now() + duration;

        while tokio::time::Instant::now() < deadline {
            let remaining = deadline - tokio::time::Instant::now();
            
            match timeout(remaining, self.receiver.recv()).await {
                Ok(Some(message)) => {
                    debug!("Collected event on subject '{}'", self.subject);
                    events.push(message.clone());
                    self.collected_events.push(message);
                }
                Ok(None) => break, // Channel closed
                Err(_) => break,   // Timeout
            }
        }

        info!("Collected {} events from subject '{}' in {:?}", events.len(), self.subject, duration);
        Ok(events)
    }

    /// Get all collected events
    pub fn get_collected_events(&self) -> &[Message] {
        &self.collected_events
    }

    /// Count events of a specific type
    pub fn count_events_of_type(&self, event_type: &str) -> usize {
        self.collected_events
            .iter()
            .filter_map(|msg| {
                serde_json::from_slice::<serde_json::Value>(&msg.payload).ok()
            })
            .filter(|event| {
                event.get("event_type")
                    .and_then(|t| t.as_str())
                    .map(|t| t == event_type)
                    .unwrap_or(false)
            })
            .count()
    }
}

/// Test helper for NATS event testing
pub struct EventTestHelper {
    client: NatsClient,
    collectors: HashMap<String, EventCollector>,
}

impl EventTestHelper {
    pub async fn new(nats_url: &str) -> Result<Self> {
        let client = NatsClient::new(nats_url).await?;
        
        Ok(Self {
            client,
            collectors: HashMap::new(),
        })
    }

    /// Setup event collection for driver lifecycle tests
    pub async fn setup_driver_event_collection(&mut self) -> Result<()> {
        let subjects = vec![
            "driver.registered",
            "driver.verified",
            "driver.status.changed",
            "driver.location.updated",
            "driver.rating.updated",
        ];

        for subject in subjects {
            let collector = self.client.subscribe_to_events(subject).await?;
            self.collectors.insert(subject.to_string(), collector);
        }

        Ok(())
    }

    /// Wait for driver registration event
    pub async fn wait_for_driver_registered(&mut self, timeout_duration: Duration) -> Result<DriverRegisteredEvent> {
        if let Some(collector) = self.collectors.get_mut("driver.registered") {
            timeout(timeout_duration, collector.wait_for_event::<DriverRegisteredEvent>()).await
                .map_err(|_| anyhow!("Timeout waiting for driver.registered event"))?
        } else {
            Err(anyhow!("No collector for driver.registered events"))
        }
    }

    /// Wait for location update event
    pub async fn wait_for_location_updated(&mut self, timeout_duration: Duration) -> Result<DriverLocationUpdatedEvent> {
        if let Some(collector) = self.collectors.get_mut("driver.location.updated") {
            timeout(timeout_duration, collector.wait_for_event::<DriverLocationUpdatedEvent>()).await
                .map_err(|_| anyhow!("Timeout waiting for driver.location.updated event"))?
        } else {
            Err(anyhow!("No collector for driver.location.updated events"))
        }
    }

    /// Simulate full order lifecycle
    pub async fn simulate_order_lifecycle(&self, driver_id: Uuid) -> Result<Uuid> {
        let order_id = Uuid::new_v4();

        // 1. Assign order
        self.client.simulate_order_assigned(driver_id, order_id).await?;

        // 2. Wait a bit for processing
        tokio::time::sleep(Duration::from_millis(100)).await;

        // 3. Complete order
        let event = serde_json::json!({
            "event_type": "order.completed",
            "order_id": order_id.to_string(),
            "driver_id": driver_id.to_string(),
            "customer_id": Uuid::new_v4().to_string(),
            "actual_fare": 280.0,
            "actual_distance": 2.8,
            "duration": 18,
            "rating": 5,
            "tips": 50.0,
            "timestamp": chrono::Utc::now()
        });

        self.client.publish_event("order.completed", &event).await?;

        // 4. Process payment
        self.client.simulate_payment_processed(driver_id, order_id, 280.0).await?;

        // 5. Customer rating
        self.client.simulate_customer_rating(driver_id, order_id, 5).await?;

        Ok(order_id)
    }

    /// Get client for direct access
    pub fn get_client(&self) -> &NatsClient {
        &self.client
    }
}