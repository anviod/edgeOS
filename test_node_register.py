import json
import time
import paho.mqtt.client as mqtt

# MQTT 配置
broker = "localhost"
port = 1883
client_id = "test-publisher"

# 节点注册消息
register_message = {
    "header": {
        "message_id": "msg-node-reg-001",
        "timestamp": int(time.time() * 1000),
        "source": "edgex-node-001",
        "destination": "edgeos-queen",
        "message_type": "node_register",
        "version": "1.0"
    },
    "body": {
        "node_id": "edgex-node-001",
        "node_name": "EdgeX Gateway Node",
        "model": "edge-gateway",
        "version": "1.0.0",
        "api_version": "v1",
        "capabilities": [
            "shadow-sync",
            "heartbeat",
            "device-control",
            "task-execution"
        ],
        "protocol": "edgeOS(MQTT)",
        "endpoint": {
            "host": "127.0.0.1",
            "port": 8082
        },
        "metadata": {
            "os": "linux",
            "arch": "amd64",
            "hostname": "edgex-node-001.local"
        }
    }
}

# 连接回调
def on_connect(client, userdata, flags, rc):
    print(f"Connected with result code {rc}")
    # 发布节点注册消息
    client.publish("edgex/nodes/register", json.dumps(register_message), qos=1)
    print("Node register message published")
    # 等待 2 秒后断开连接
    time.sleep(2)
    client.disconnect()

# 初始化 MQTT 客户端
client = mqtt.Client(client_id=client_id)
client.on_connect = on_connect

# 连接到 MQTT 代理
print("Connecting to MQTT broker...")
client.connect(broker, port, 60)

# 开始循环
client.loop_forever()
