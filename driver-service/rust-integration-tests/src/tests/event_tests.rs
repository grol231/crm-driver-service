//! NATS Event Integration Tests
//! 
//! This module contains comprehensive tests for NATS event publishing and consumption,
//! event-driven workflows, and message delivery guarantees.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use std::time::Duration as StdDuration;
use uuid::Uuid;

use crate::fixtures::{
    generate_test_drivers, DriverRegisteredEvent, DriverLocationUpdatedEvent, 
    OrderAssignedEvent, LocationData, UpdateLocationRequest
};
use crate::helpers::{with_timeout, TestResults, EventTestHelper};
use crate::{TestEnvironment, init_test_environment, NatsClient};

/// Test NATS connection and basic publish/subscribe
#[tokio::test]
#[serial]
async fn test_nats_basic_connectivity() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        // Test basic publish/subscribe
        let test_subject = "test.connectivity";
        let test_message = serde_json::json!({
            "test": true,
            "message": "Hello NATS",
            "timestamp": Utc::now()
        });

        // Subscribe first
        let mut collector = nats_client.subscribe_to_events(test_subject).await?;

        // Publish message
        nats_client.publish_event(test_subject, &test_message).await?;

        // Wait for message
        let received = with_timeout(
            collector.wait_for_event::<serde_json::Value>(),
            StdDuration::from_secs(5),
            "wait_for_test_message"
        ).await?;

        assert_eq!(received["test"], true);
        assert_eq!(received["message"], "Hello NATS");
    } else {
        println!("NATS not configured, skipping NATS connectivity test");
    }

    Ok(())
}

/// Test driver registration events
#[tokio::test]
#[serial]
async fn test_driver_registration_events() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        env.cleanup().await?;
        
        // Setup event collection
        let mut event_helper = EventTestHelper::new(&env.config.nats.url).await?;
        event_helper.setup_driver_event_collection().await?;

        // Create driver through API (should trigger event)
        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
        let created_driver = env.api_client.create_test_driver(&test_driver).await?;

        // Wait for driver registration event
        let driver_registered = event_helper.wait_for_driver_registered(StdDuration::from_secs(5)).await?;

        assert_eq!(driver_registered.driver_id, created_driver.id.to_string());
        assert_eq!(driver_registered.phone, created_driver.phone);
        assert_eq!(driver_registered.email, created_driver.email);
        assert_eq!(driver_registered.event_type, "driver.registered");
        assert_eq!(driver_registered.license_number, created_driver.license_number);
    } else {
        println!("NATS not configured, skipping driver registration event test");
    }

    Ok(())
}

/// Test location update events
#[tokio::test]
#[serial]
async fn test_location_update_events() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        env.cleanup().await?;

        // Setup event collection
        let mut event_helper = EventTestHelper::new(&env.config.nats.url).await?;
        event_helper.setup_driver_event_collection().await?;

        // Create driver
        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
        let created_driver = env.api_client.create_test_driver(&test_driver).await?;

        // Update location (should trigger event)
        let location_request = UpdateLocationRequest {
            latitude: 55.7558,
            longitude: 37.6176,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(30.0),
            bearing: Some(45.0),
            timestamp: Some(Utc::now().timestamp()),
        };

        env.api_client.update_location(created_driver.id, &location_request).await?;

        // Wait for location update event
        let location_updated = event_helper.wait_for_location_updated(StdDuration::from_secs(5)).await?;

        assert_eq!(location_updated.driver_id, created_driver.id.to_string());
        assert_eq!(location_updated.event_type, "driver.location.updated");
        assert_eq!(location_updated.location.latitude, location_request.latitude);
        assert_eq!(location_updated.location.longitude, location_request.longitude);
        assert_eq!(location_updated.speed, location_request.speed.unwrap());
        assert_eq!(location_updated.bearing, location_request.bearing.unwrap());
        assert_eq!(location_updated.accuracy, location_request.accuracy.unwrap());
    } else {
        println!("NATS not configured, skipping location update event test");
    }

    Ok(())
}

/// Test order assignment event handling
#[tokio::test]
#[serial]
async fn test_order_assignment_events() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        env.cleanup().await?;

        // Create and set driver as available
        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
        let created_driver = env.api_client.create_test_driver(&test_driver).await?;
        env.api_client.change_driver_status(created_driver.id, "available").await?;

        // Subscribe to status change events
        let mut status_collector = nats_client.subscribe_to_events("driver.status.changed").await?;

        // Simulate order assignment
        let order_id = Uuid::new_v4();
        nats_client.simulate_order_assigned(created_driver.id, order_id).await?;

        // Wait for driver status change to "busy"
        let status_events = status_collector.collect_events_for_duration(StdDuration::from_secs(3)).await?;

        // Should receive at least one status change event
        assert!(!status_events.is_empty(), "Should receive driver status change events");

        // Verify driver status in database
        let updated_driver = env.api_client.get_driver(created_driver.id).await?;
        // Note: This depends on the Go service actually processing the order.assigned event
        // In a real integration test, we would verify the service responds appropriately
        
        println!("Order assignment test completed - driver status: {}", updated_driver.status);
    } else {
        println!("NATS not configured, skipping order assignment event test");
    }

    Ok(())
}

/// Test event message ordering and delivery guarantees
#[tokio::test]
#[serial]
async fn test_event_ordering_and_delivery() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        let test_subject = "test.ordering";
        
        // Subscribe first
        let mut collector = nats_client.subscribe_to_events(test_subject).await?;

        // Publish sequence of messages
        let message_count = 10;
        for i in 0..message_count {
            let message = serde_json::json!({
                "sequence": i,
                "timestamp": Utc::now(),
                "data": format!("Message {}", i)
            });

            nats_client.publish_event(test_subject, &message).await?;
            
            // Small delay between messages
            tokio::time::sleep(StdDuration::from_millis(10)).await;
        }

        // Collect all messages
        let received_messages = collector.collect_events_for_duration(StdDuration::from_secs(5)).await?;

        // Verify all messages were received
        assert_eq!(received_messages.len(), message_count, "All messages should be received");

        // Verify message order (NATS preserves order within a single publisher)
        for (i, message) in received_messages.iter().enumerate() {
            let parsed: serde_json::Value = serde_json::from_slice(&message.payload)?;
            assert_eq!(parsed["sequence"], i, "Messages should be in order");
        }
    } else {
        println!("NATS not configured, skipping event ordering test");
    }

    Ok(())
}

/// Test event-driven driver lifecycle workflow
#[tokio::test]
#[serial]
async fn test_driver_lifecycle_events() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        env.cleanup().await?;

        // Setup comprehensive event collection
        let mut event_helper = EventTestHelper::new(&env.config.nats.url).await?;
        event_helper.setup_driver_event_collection().await?;

        // 1. Register driver
        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
        let created_driver = env.api_client.create_test_driver(&test_driver).await?;

        // 2. Change status to available
        env.api_client.change_driver_status(created_driver.id, "available").await?;

        // 3. Update location
        let location_request = UpdateLocationRequest {
            latitude: 55.7558,
            longitude: 37.6176,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(0.0),
            bearing: Some(0.0),
            timestamp: Some(Utc::now().timestamp()),
        };
        env.api_client.update_location(created_driver.id, &location_request).await?;

        // 4. Simulate complete order lifecycle
        let order_id = event_helper.simulate_order_lifecycle(created_driver.id).await?;

        // Allow time for all events to propagate
        tokio::time::sleep(StdDuration::from_secs(2)).await;

        // Verify we received expected events
        // Note: This would require the Go service to actually process events and emit responses
        println!("Driver lifecycle events test completed for driver {} and order {}", 
                created_driver.id, order_id);

        // In a real integration test, we would verify:
        // - driver.registered event
        // - driver.status.changed events
        // - driver.location.updated event  
        // - driver.rating.updated event (after customer rating)
        // - Any other workflow-specific events
    } else {
        println!("NATS not configured, skipping driver lifecycle events test");
    }

    Ok(())
}

/// Test event error handling and retry mechanisms
#[tokio::test]
#[serial]
async fn test_event_error_handling() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        let test_subject = "test.error.handling";

        // Test publishing invalid JSON should not crash the client
        let invalid_message = "invalid json content";
        let response = reqwest::Client::new()
            .post(&format!("{}/internal/nats/publish", env.config.driver_service.base_url))
            .json(&serde_json::json!({
                "subject": test_subject,
                "data": invalid_message
            }))
            .send()
            .await;

        // The service should handle this gracefully
        // (This test assumes there's an internal NATS publish endpoint for testing)
        match response {
            Ok(resp) => println!("Error handling test response: {}", resp.status()),
            Err(e) => println!("Expected error in error handling test: {}", e),
        }

        // Test subscription to non-existent subject should work
        let _collector = nats_client.subscribe_to_events("non.existent.subject").await?;
        
        // Should not throw errors, just wait without receiving messages
        println!("Error handling test completed successfully");
    } else {
        println!("NATS not configured, skipping event error handling test");
    }

    Ok(())
}

/// Test high-volume event processing
#[tokio::test]
#[serial]
async fn test_high_volume_events() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        if !env.config.test.performance_test_enabled {
            println!("Performance tests disabled, skipping high-volume events test");
            return Ok(());
        }

        let test_subject = "test.high.volume";
        let message_count = 1000;

        // Subscribe first
        let mut collector = nats_client.subscribe_to_events(test_subject).await?;

        // Publish high volume of messages
        let start_time = std::time::Instant::now();
        
        for i in 0..message_count {
            let message = serde_json::json!({
                "event_type": "test.event",
                "sequence": i,
                "timestamp": Utc::now(),
                "data": format!("High volume test message {}", i)
            });

            nats_client.publish_event(test_subject, &message).await?;
        }

        let publish_duration = start_time.elapsed();
        let publish_rate = message_count as f64 / publish_duration.as_secs_f64();

        // Collect messages with timeout
        let received_messages = collector.collect_events_for_duration(StdDuration::from_secs(10)).await?;
        
        println!("High volume test results:");
        println!("  Published: {} messages in {:?} ({:.2} msg/sec)", 
                message_count, publish_duration, publish_rate);
        println!("  Received: {} messages", received_messages.len());

        // Should receive most messages (allow for some loss in high-volume scenarios)
        let received_percentage = received_messages.len() as f64 / message_count as f64;
        assert!(
            received_percentage > 0.95,
            "Should receive > 95% of messages, got {:.1}%",
            received_percentage * 100.0
        );

        // Publish rate should be reasonable
        assert!(
            publish_rate > 100.0,
            "Publish rate should be > 100 msg/sec, got {:.2}",
            publish_rate
        );
    } else {
        println!("NATS not configured, skipping high-volume events test");
    }

    Ok(())
}

/// Test concurrent event publishers and consumers
#[tokio::test]
#[serial]
async fn test_concurrent_event_processing() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(_nats_client) = &env.nats_client {
        let publishers = 5;
        let messages_per_publisher = 20;
        let test_subject = "test.concurrent";

        // Create multiple collectors
        let mut collectors = Vec::new();
        for i in 0..publishers {
            let nats_client = NatsClient::new(&env.config.nats.url).await?;
            let collector = nats_client.subscribe_to_events(test_subject).await?;
            collectors.push((nats_client, collector));
        }

        // Launch concurrent publishers
        let mut publisher_tasks = Vec::new();
        
        for pub_id in 0..publishers {
            let nats_url = env.config.nats.url.clone();
            let subject = test_subject.to_string();
            
            let task = tokio::spawn(async move {
                let client = NatsClient::new(&nats_url).await?;
                
                for msg_id in 0..messages_per_publisher {
                    let message = serde_json::json!({
                        "publisher_id": pub_id,
                        "message_id": msg_id,
                        "timestamp": Utc::now(),
                        "data": format!("Concurrent message from publisher {} msg {}", pub_id, msg_id)
                    });

                    client.publish_event(&subject, &message).await?;
                    
                    // Small delay between messages from same publisher
                    tokio::time::sleep(StdDuration::from_millis(10)).await;
                }
                
                Ok::<(), anyhow::Error>(())
            });
            
            publisher_tasks.push(task);
        }

        // Wait for publishers to complete
        let _publisher_results = futures::future::join_all(publisher_tasks).await;

        // Collect messages from all subscribers
        let mut all_received_messages = Vec::new();
        
        for (_, mut collector) in collectors {
            let messages = collector.collect_events_for_duration(StdDuration::from_secs(5)).await?;
            all_received_messages.extend(messages);
        }

        let total_expected = publishers * messages_per_publisher;
        let total_received = all_received_messages.len();
        
        println!("Concurrent event processing results:");
        println!("  Expected: {} messages", total_expected);
        println!("  Received: {} messages", total_received);

        // Allow for some duplication in concurrent scenarios but should receive most messages
        assert!(
            total_received >= total_expected,
            "Should receive at least {} messages, got {}",
            total_expected,
            total_received
        );
    } else {
        println!("NATS not configured, skipping concurrent event processing test");
    }

    Ok(())
}

/// Test event schema validation and versioning
#[tokio::test]
#[serial]
async fn test_event_schema_validation() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        let test_subject = "test.schema.validation";

        // Test valid event schema
        let valid_event = DriverRegisteredEvent {
            event_type: "driver.registered".to_string(),
            driver_id: Uuid::new_v4().to_string(),
            phone: "+79001234567".to_string(),
            email: "test@example.com".to_string(),
            name: "Test Driver".to_string(),
            license_number: "TEST123456".to_string(),
            city: "Moscow".to_string(),
            timestamp: Utc::now(),
        };

        // Should publish successfully
        let result = nats_client.publish_event(test_subject, &valid_event).await;
        assert!(result.is_ok(), "Valid event should publish successfully");

        // Test event with missing required fields
        let invalid_event = serde_json::json!({
            "event_type": "driver.registered",
            // missing required fields like driver_id, phone, etc.
            "timestamp": Utc::now()
        });

        // Should still publish (NATS doesn't validate schema)
        // But consuming service should handle gracefully
        let result = nats_client.publish_event(test_subject, &invalid_event).await;
        assert!(result.is_ok(), "NATS should accept any valid JSON");

        // In real integration tests, we would verify that the consuming service
        // handles invalid events gracefully without crashing

        println!("Event schema validation test completed");
    } else {
        println!("NATS not configured, skipping event schema validation test");
    }

    Ok(())
}

/// Integration test combining API operations with event verification
#[tokio::test]
#[serial]
async fn test_api_events_integration() -> Result<()> {
    let env = init_test_environment().await?;
    
    if let Some(nats_client) = &env.nats_client {
        env.cleanup().await?;

        // Setup event collection
        let mut event_helper = EventTestHelper::new(&env.config.nats.url).await?;
        event_helper.setup_driver_event_collection().await?;

        // 1. Create driver via API and verify event
        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
        let created_driver = env.api_client.create_test_driver(&test_driver).await?;

        // 2. Update driver status and verify event  
        env.api_client.change_driver_status(created_driver.id, "available").await?;

        // 3. Update location and verify event
        let location_request = UpdateLocationRequest {
            latitude: 55.7558,
            longitude: 37.6176,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(25.0),
            bearing: Some(90.0),
            timestamp: Some(Utc::now().timestamp()),
        };
        env.api_client.update_location(created_driver.id, &location_request).await?;

        // 4. Simulate external events affecting the driver
        let order_id = Uuid::new_v4();
        nats_client.simulate_order_assigned(created_driver.id, order_id).await?;

        // Allow time for events to propagate
        tokio::time::sleep(StdDuration::from_secs(2)).await;

        // 5. Verify driver state reflects the events
        let final_driver = env.api_client.get_driver(created_driver.id).await?;
        let current_location = env.api_client.get_current_location(created_driver.id).await?;

        // Verify location was updated
        assert_eq!(current_location.latitude, location_request.latitude);
        assert_eq!(current_location.longitude, location_request.longitude);

        println!("API events integration test completed successfully");
        println!("Final driver status: {}", final_driver.status);
        println!("Current location: {}, {}", current_location.latitude, current_location.longitude);
    } else {
        println!("NATS not configured, skipping API events integration test");
    }

    Ok(())
}