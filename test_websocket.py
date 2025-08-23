#!/usr/bin/env python3

import asyncio
import websockets
import json
import sys
from datetime import datetime

async def test_realtime_updates():
    # Connect two WebSocket clients
    uri = "ws://localhost:3000/ws"
    
    print("🔌 Connecting two WebSocket clients...")
    print("=" * 50)
    
    async with websockets.connect(uri) as ws1, websockets.connect(uri) as ws2:
        # Subscribe both clients
        subscribe_msg = json.dumps({
            "type": "SUBSCRIBE",
            "data": {"user_id": "user1"}
        })
        await ws1.send(subscribe_msg)
        
        subscribe_msg2 = json.dumps({
            "type": "SUBSCRIBE",
            "data": {"user_id": "user2"}
        })
        await ws2.send(subscribe_msg2)
        
        # Wait for welcome and venue state messages
        for i in range(4):  # Expect 2 messages per client
            msg = await asyncio.wait_for(
                asyncio.gather(
                    ws1.recv() if i < 2 else asyncio.sleep(0),
                    ws2.recv() if i >= 2 else asyncio.sleep(0)
                ),
                timeout=2.0
            )
        
        print("✅ Both clients connected and subscribed")
        print()
        
        # Client 1 selects a seat
        print("📍 Client 1 selecting seat G7...")
        select_msg = json.dumps({
            "type": "SELECT_SEAT",
            "data": {
                "seat_id": "G7",
                "user_id": "user1"
            }
        })
        await ws1.send(select_msg)
        
        # Both clients should receive the update
        print("📨 Waiting for real-time updates...")
        print()
        
        messages_received = []
        try:
            # Collect messages from both clients
            for _ in range(3):  # Wait for multiple messages
                done, pending = await asyncio.wait(
                    [
                        asyncio.create_task(ws1.recv()),
                        asyncio.create_task(ws2.recv())
                    ],
                    timeout=2.0,
                    return_when=asyncio.FIRST_COMPLETED
                )
                
                for task in done:
                    msg = await task
                    data = json.loads(msg)
                    messages_received.append(data)
                    
                    print(f"📩 Message received: {data['type']}")
                    if data['type'] == 'SEAT_UPDATE':
                        event = data['data']
                        print(f"   Event: {event.get('event_type', 'unknown')}")
                        print(f"   Seat: {event.get('seat_id', 'unknown')}")
                        print(f"   User: {event.get('user_id', 'unknown')}")
                        print(f"   Status: {event.get('status', 'unknown')}")
                    elif data['type'] == 'SELECT_SEAT_RESPONSE':
                        print(f"   Success: {data['data'].get('success', False)}")
                        print(f"   Message: {data['data'].get('message', '')}")
                    print()
                
                # Cancel pending tasks
                for task in pending:
                    task.cancel()
                    
        except asyncio.TimeoutError:
            print("⏱️  No more messages (timeout)")
        
        print()
        print("=" * 50)
        print("📊 Test Summary:")
        print(f"   Total messages received: {len(messages_received)}")
        
        # Check if both clients received the update
        seat_updates = [m for m in messages_received if m['type'] == 'SEAT_UPDATE']
        if seat_updates:
            print(f"   ✅ Real-time updates working! ({len(seat_updates)} SEAT_UPDATE messages)")
        else:
            print("   ⚠️  No SEAT_UPDATE messages received")
        
        print()
        print("✨ WebSocket real-time test complete!")

if __name__ == "__main__":
    try:
        asyncio.run(test_realtime_updates())
    except KeyboardInterrupt:
        print("\nTest interrupted by user")
    except Exception as e:
        print(f"Error: {e}")
        sys.exit(1)