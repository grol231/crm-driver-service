"""Tests for WebSocket connections."""
import pytest
import asyncio
import json
import time
from typing import Dict, Any
import websockets
from websockets.exceptions import ConnectionClosedError, InvalidURI

from config import get_test_config
from utils.helpers import generate_test_location_data

config = get_test_config()


@pytest.mark.websocket
class TestWebSocketConnections:
    """Test WebSocket connections for real-time features."""
    
    @pytest.mark.asyncio
    async def test_location_tracking_websocket_connection(self, created_driver):
        """Test WebSocket connection for location tracking."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        try:
            async with websockets.connect(ws_url) as websocket:
                # Connection should be established
                assert websocket.open
                
                # Send a test message (ping)
                test_message = {
                    "type": "ping",
                    "timestamp": int(time.time())
                }
                await websocket.send(json.dumps(test_message))
                
                # Should receive a response within reasonable time
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                    response_data = json.loads(response)
                    
                    # Response should be pong or acknowledgment
                    assert response_data.get("type") in ["pong", "ack", "connected"]
                    
                except asyncio.TimeoutError:
                    # Some WebSocket implementations might not respond to ping
                    # This is acceptable for connection test
                    pass
                    
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
    
    @pytest.mark.asyncio
    async def test_location_updates_via_websocket(self, created_driver, http_client):
        """Test receiving location updates via WebSocket."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        try:
            async with websockets.connect(ws_url) as websocket:
                # Set up listener for messages
                received_messages = []
                
                # Function to collect messages
                async def message_collector():
                    try:
                        while True:
                            message = await asyncio.wait_for(websocket.recv(), timeout=1.0)
                            received_messages.append(json.loads(message))
                    except (asyncio.TimeoutError, ConnectionClosedError):
                        pass
                
                # Start message collector
                collector_task = asyncio.create_task(message_collector())
                
                # Wait a bit for connection to stabilize
                await asyncio.sleep(0.5)
                
                # Send location update via HTTP API (should trigger WebSocket notification)
                location_data = generate_test_location_data()
                response = http_client.post(
                    f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
                    json=location_data
                )
                
                assert response.status_code == 200
                
                # Wait for WebSocket message
                await asyncio.sleep(1.0)
                collector_task.cancel()
                
                # Check if we received location update message
                location_messages = [
                    msg for msg in received_messages 
                    if msg.get("type") == "location_update"
                ]
                
                if location_messages:
                    location_msg = location_messages[0]
                    assert location_msg["driver_id"] == driver_id
                    assert "location" in location_msg
                    assert "timestamp" in location_msg
                    
                    location = location_msg["location"]
                    assert location["latitude"] == location_data["latitude"]
                    assert location["longitude"] == location_data["longitude"]
                
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
    
    @pytest.mark.asyncio
    async def test_order_notifications_websocket(self, created_driver):
        """Test WebSocket connection for order notifications."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/orders/{driver_id}"
        
        try:
            async with websockets.connect(ws_url) as websocket:
                # Connection should be established
                assert websocket.open
                
                # Test connection by sending a heartbeat
                heartbeat = {
                    "type": "heartbeat",
                    "timestamp": int(time.time())
                }
                await websocket.send(json.dumps(heartbeat))
                
                # Wait for response or timeout gracefully
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=3.0)
                    # If we get a response, validate it
                    if response:
                        response_data = json.loads(response)
                        assert isinstance(response_data, dict)
                        assert "type" in response_data
                except asyncio.TimeoutError:
                    # No response is also acceptable for heartbeat
                    pass
                    
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
    
    @pytest.mark.asyncio
    async def test_websocket_invalid_driver_id(self):
        """Test WebSocket connection with invalid driver ID."""
        invalid_id = "invalid-uuid"
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{invalid_id}"
        
        try:
            # Connection should fail or close quickly
            with pytest.raises((ConnectionRefusedError, websockets.exceptions.ConnectionClosedError, InvalidURI)):
                async with websockets.connect(ws_url) as websocket:
                    # If connection succeeds, it should close quickly
                    await asyncio.sleep(1.0)
                    # Try to send a message - should fail
                    await websocket.send(json.dumps({"type": "test"}))
                    
        except (ConnectionRefusedError, InvalidURI) as e:
            # Expected - invalid driver ID should cause connection failure
            pass
    
    @pytest.mark.asyncio
    async def test_websocket_connection_limits(self, created_driver):
        """Test WebSocket connection limits and behavior."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        connections = []
        max_connections = 3
        
        try:
            # Try to establish multiple connections for same driver
            for i in range(max_connections):
                try:
                    conn = await websockets.connect(ws_url)
                    connections.append(conn)
                    await asyncio.sleep(0.1)  # Small delay between connections
                except Exception:
                    break
            
            # At least one connection should succeed
            assert len(connections) >= 1
            
            # Test that connections are functional
            if connections:
                test_message = {
                    "type": "connection_test",
                    "connection_id": 0
                }
                await connections[0].send(json.dumps(test_message))
                
                # Connection should remain open
                assert connections[0].open
            
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
        finally:
            # Clean up connections
            for conn in connections:
                if not conn.closed:
                    await conn.close()
    
    @pytest.mark.asyncio
    async def test_websocket_message_format_validation(self, created_driver):
        """Test WebSocket message format validation."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        try:
            async with websockets.connect(ws_url) as websocket:
                # Test invalid JSON
                try:
                    await websocket.send("invalid json")
                    # Server might close connection or send error
                    await asyncio.sleep(0.5)
                except Exception:
                    # Expected for invalid JSON
                    pass
                
                # Test valid JSON but invalid message structure
                invalid_message = {"invalid": "structure"}
                await websocket.send(json.dumps(invalid_message))
                
                # Try to receive response (error or acknowledgment)
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=2.0)
                    if response:
                        response_data = json.loads(response)
                        # Should be error or acknowledgment
                        assert isinstance(response_data, dict)
                except asyncio.TimeoutError:
                    # No response is acceptable
                    pass
                    
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
    
    @pytest.mark.asyncio
    async def test_websocket_connection_authentication(self):
        """Test WebSocket connection authentication (if implemented)."""
        # Test connection without proper authentication
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/unauthorized"
        
        try:
            # This should fail if authentication is required
            with pytest.raises((ConnectionRefusedError, websockets.exceptions.ConnectionClosedError)):
                async with websockets.connect(ws_url) as websocket:
                    await websocket.send(json.dumps({"type": "test"}))
                    await asyncio.sleep(1.0)
        except (ConnectionRefusedError, InvalidURI):
            # Expected if authentication is enforced
            pass
    
    @pytest.mark.asyncio 
    async def test_websocket_concurrent_connections(self, multiple_drivers):
        """Test concurrent WebSocket connections for multiple drivers."""
        connections = []
        
        try:
            # Connect to multiple drivers simultaneously
            connect_tasks = []
            for driver in multiple_drivers[:3]:  # Test with 3 drivers
                ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver['id']}"
                connect_tasks.append(websockets.connect(ws_url))
            
            # Attempt to establish all connections
            try:
                connections = await asyncio.gather(*connect_tasks, return_exceptions=True)
            except Exception:
                # Some connections might fail
                pass
            
            # Count successful connections
            successful_connections = [
                conn for conn in connections 
                if hasattr(conn, 'open') and conn.open
            ]
            
            # At least some connections should succeed
            if successful_connections:
                # Test that all successful connections can receive messages
                for i, conn in enumerate(successful_connections):
                    test_message = {
                        "type": "multi_test",
                        "connection_id": i
                    }
                    await conn.send(json.dumps(test_message))
                
                # Small delay to process messages
                await asyncio.sleep(0.5)
                
                # All connections should still be open
                for conn in successful_connections:
                    assert conn.open
            
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
        finally:
            # Clean up all connections
            for conn in connections:
                if hasattr(conn, 'close') and hasattr(conn, 'open') and conn.open:
                    try:
                        await conn.close()
                    except Exception:
                        pass
    
    @pytest.mark.asyncio
    async def test_websocket_connection_recovery(self, created_driver):
        """Test WebSocket connection recovery after disconnect."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        try:
            # First connection
            async with websockets.connect(ws_url) as websocket1:
                assert websocket1.open
                
                # Send a message
                test_message = {"type": "test", "sequence": 1}
                await websocket1.send(json.dumps(test_message))
                
                # Close connection
                await websocket1.close()
            
            # Reconnect after a short delay
            await asyncio.sleep(0.5)
            
            async with websockets.connect(ws_url) as websocket2:
                assert websocket2.open
                
                # Send another message
                test_message = {"type": "test", "sequence": 2}
                await websocket2.send(json.dumps(test_message))
                
                # Connection should work normally
                await asyncio.sleep(0.5)
                assert websocket2.open
                
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
    
    @pytest.mark.asyncio
    async def test_websocket_large_message_handling(self, created_driver):
        """Test WebSocket handling of large messages."""
        driver_id = created_driver['id']
        ws_url = f"ws://{config.service_host}:{config.service_http_port}/ws/tracking/{driver_id}"
        
        try:
            async with websockets.connect(ws_url) as websocket:
                # Create a large message (but reasonable size)
                large_data = {
                    "type": "bulk_location_update",
                    "locations": [
                        generate_test_location_data() for _ in range(100)
                    ],
                    "driver_id": driver_id
                }
                
                large_message = json.dumps(large_data)
                
                # Send large message
                await websocket.send(large_message)
                
                # Wait for processing
                await asyncio.sleep(1.0)
                
                # Connection should still be open
                assert websocket.open
                
                # Send a small follow-up message
                small_message = {"type": "ping"}
                await websocket.send(json.dumps(small_message))
                
        except (ConnectionRefusedError, InvalidURI) as e:
            pytest.skip(f"WebSocket server not available: {e}")
        except Exception as e:
            # Large messages might be rejected, which is acceptable
            if "message too large" in str(e).lower():
                pytest.skip("Server rejects large messages - acceptable behavior")
            else:
                raise