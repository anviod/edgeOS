#!/usr/bin/env python3
import json
import time
import paho.mqtt.client as mqtt

MQTT_HOST = "127.0.0.1"
MQTT_PORT = 1883

def on_connect(client, userdata, flags, rc):
    print(f"Connected with result code {rc}")
    # 订阅响应主题
    client.subscribe("edgex/responses/#")
    print("Subscribed to edgex/responses/#")

def on_message(client, userdata, msg):
    print(f"\nReceived message on {msg.topic}")
    print(f"Payload: {msg.payload.decode()}")

def test_write_command():
    client = mqtt.Client(client_id="test-client")
    client.on_connect = on_connect
    client.on_message = on_message

    print(f"Connecting to MQTT broker at {MQTT_HOST}:{MQTT_PORT}...")
    client.connect(MQTT_HOST, MQTT_PORT, 60)

    # 启动网络循环
    client.loop_start()

    # 等待连接建立
    time.sleep(1)

    # 发送写入命令
    write_command = {
        "header": {
            "message_id": "test-write-004",
            "timestamp": int(time.time() * 1000),
            "source": "edgeOS",
            "message_type": "write_command",
            "version": "1.0"
        },
        "body": {
            "node_id": "edgex-node-001",
            "device_id": "slave-1",
            "point_id": "hr_40000",
            "value": 401
        }
    }

    write_topic = "edgex/cmd/edgex-node-001/slave-1/write"
    print(f"\nSending write command to {write_topic}")
    print(f"Payload: {json.dumps(write_command, indent=2)}")
    
    client.publish(write_topic, json.dumps(write_command), qos=1)
    print("Write command sent!")

    # 等待响应
    print("\nWaiting for response (10 seconds)...")
    time.sleep(10)

    # 发送模拟响应
    response = {
        "header": {
            "message_id": "test-write-004",
            "timestamp": int(time.time() * 1000),
            "source": "edgex-node-001",
            "message_type": "command_response",
            "version": "1.0"
        },
        "body": {
            "request_id": "test-write-004",
            "node_id": "edgex-node-001",
            "device_id": "slave-1",
            "point_id": "hr_40000",
            "success": True,
            "value": 401
        }
    }

    response_topic = "edgex/responses/edgex-node-001/test-write-004"
    print(f"\nSending simulated response to {response_topic}")
    print(f"Payload: {json.dumps(response, indent=2)}")
    
    client.publish(response_topic, json.dumps(response), qos=1)
    print("Response sent!")

    # 等待处理
    print("\nWaiting for processing (5 seconds)...")
    time.sleep(5)

    client.loop_stop()
    client.disconnect()
    print("\nTest completed!")

if __name__ == "__main__":
    test_write_command()