#!/usr/bin/env python3
"""
EdgeOS 与 EdgeX 通信测试脚本
用 Python 实现，跨平台支持

前提条件:
1. pip install paho-mqtt
2. MQTT Broker 运行在 127.0.0.1:1883
3. EdgeOS 服务运行并连接到 MQTT Broker

用法:
    python test_edgex_flows.py [flow_name]
    python test_edgex_flows.py all          # 运行所有测试
    python test_edgex_flows.py subscribe    # 显示订阅命令
"""

import json
import time
import sys
import argparse
from datetime import datetime

try:
    import paho.mqtt.client as mqtt
except ImportError:
    print("请安装 paho-mqtt: pip install paho-mqtt")
    sys.exit(1)

# 配置
MQTT_HOST = "127.0.0.1"
MQTT_PORT = 1883
NODE_ID = "edgex-node-001"
DEVICE_ID = "Room_FC_2014_19"

# 颜色输出
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
NC = '\033[0m'

def log_info(msg):
    print(f"{GREEN}[INFO]{NC} {msg}")

def log_warn(msg):
    print(f"{YELLOW}[WARN]{NC} {msg}")

def log_error(msg):
    print(f"{RED}[ERROR]{NC} {msg}")

class EdgeXTester:
    def __init__(self, host=MQTT_HOST, port=MQTT_PORT):
        self.host = host
        self.port = port
        self.client = mqtt.Client(client_id="edgex-test-client")
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.received_messages = []

    def _on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            log_info(f"已连接到 MQTT Broker {self.host}:{self.port}")
        else:
            log_error(f"连接失败, 返回码: {rc}")

    def _on_disconnect(self, client, userdata, rc):
        log_warn(f"断开连接, 返回码: {rc}")

    def _on_message(self, client, userdata, msg):
        self.received_messages.append({
            'topic': msg.topic,
            'payload': msg.payload.decode('utf-8', errors='ignore')
        })
        print(f"  收到消息 [{msg.topic}]: {msg.payload.decode('utf-8', errors='ignore')[:100]}...")

    def connect(self):
        try:
            self.client.connect(self.host, self.port, 60)
            self.client.loop_start()
            time.sleep(1)
            return True
        except Exception as e:
            log_error(f"连接失败: {e}")
            return False

    def disconnect(self):
        self.client.loop_stop()
        self.client.disconnect()

    def publish(self, topic, payload):
        result = self.client.publish(topic, json.dumps(payload), qos=1)
        if result.rc == mqtt.MQTT_ERR_SUCCESS:
            log_info(f"已发布消息到 {topic}")
            return True
        else:
            log_error(f"发布失败: {result.rc}")
            return False

    def subscribe(self, topic):
        self.client.subscribe(topic)
        log_info(f"已订阅主题: {topic}")

    def subscribe_and_wait(self, topic, timeout=5):
        self.received_messages = []
        self.client.message_callback_add(topic, self._on_message)
        self.subscribe(topic)
        time.sleep(timeout)
        self.client.message_callback_remove(topic)
        return self.received_messages

    def test_node_register(self):
        log_info("========== 测试节点注册流程 ==========")

        msg = {
            "header": {
                "message_id": "test-msg-001",
                "message_type": "register",
                "timestamp": int(datetime.now().timestamp() * 1000)
            },
            "body": {
                "node_id": NODE_ID,
                "node_name": "测试边缘节点",
                "protocol": "mqtt",
                "address": "192.168.1.100",
                "port": 1883,
                "description": "这是一个测试边缘节点"
            }
        }

        self.publish("edgex/nodes/register", msg)
        log_info("节点注册消息已发送")
        log_info("订阅响应: edgex/nodes/{}/response".format(NODE_ID))

    def test_node_heartbeat(self):
        log_info("========== 测试节点心跳流程 ==========")

        msg = {
            "header": {
                "message_id": "test-msg-002",
                "message_type": "heartbeat",
                "timestamp": int(datetime.now().timestamp() * 1000)
            },
            "body": {
                "node_id": NODE_ID,
                "status": "online"
            }
        }

        self.publish(f"edgex/nodes/{NODE_ID}/heartbeat", msg)
        log_info("心跳消息已发送")

    def test_device_sync(self):
        log_info("========== 测试设备列表同步流程 ==========")

        msg = {
            "header": {
                "message_id": "test-msg-003",
                "message_type": "device_report",
                "timestamp": int(datetime.now().timestamp() * 1000),
                "source": NODE_ID
            },
            "body": {
                "node_id": NODE_ID,
                "devices": [
                    {
                        "device_id": DEVICE_ID,
                        "device_name": "房间空调机组2014-19",
                        "protocol": "BACnet/IP",
                        "description": "19层2014房间的空调机组",
                        "manufacturer": "Trane",
                        "model": "Tracer SC",
                        "online": True
                    },
                    {
                        "device_id": "Lighting_2014",
                        "device_name": "2014房间照明",
                        "protocol": "BACnet/IP",
                        "description": "19层2014房间的照明控制",
                        "online": True
                    }
                ]
            }
        }

        self.publish("edgex/devices/report", msg)
        log_info("设备列表同步消息已发送")

    def test_point_sync(self):
        log_info("========== 测试点位列表同步流程 ==========")

        msg = {
            "header": {
                "message_id": "test-msg-004",
                "message_type": "point_report",
                "timestamp": int(datetime.now().timestamp() * 1000),
                "source": NODE_ID
            },
            "body": {
                "node_id": NODE_ID,
                "device_id": DEVICE_ID,
                "points": [
                    {
                        "point_id": "Temp_Setpoint",
                        "point_name": "温度设定",
                        "device_id": DEVICE_ID,
                        "service_name": "BACnet-Service",
                        "profile_name": "AHU-Profile",
                        "point_type": "Float",
                        "data_type": "Float",
                        "read_write": True,
                        "default_value": 24.0,
                        "units": "C",
                        "description": "房间温度设定值"
                    },
                    {
                        "point_id": "Temp_Readback",
                        "point_name": "温度回风",
                        "device_id": DEVICE_ID,
                        "service_name": "BACnet-Service",
                        "profile_name": "AHU-Profile",
                        "point_type": "Float",
                        "data_type": "Float",
                        "read_write": False,
                        "units": "C",
                        "description": "回风温度"
                    },
                    {
                        "point_id": "Fan_Speed",
                        "point_name": "风机转速",
                        "device_id": DEVICE_ID,
                        "service_name": "BACnet-Service",
                        "profile_name": "AHU-Profile",
                        "point_type": "Int",
                        "data_type": "Int64",
                        "read_write": True,
                        "default_value": 50,
                        "units": "%",
                        "description": "风机转速百分比"
                    }
                ]
            }
        }

        self.publish("edgex/points/report", msg)
        log_info("点位列表同步消息已发送")

    def test_realtime_data(self):
        log_info("========== 测试实时数据更新流程 ==========")

        # 全量数据快照
        msg_full = {
            "header": {
                "message_id": "test-msg-005",
                "message_type": "data_full",
                "timestamp": int(datetime.now().timestamp() * 1000),
                "source": NODE_ID
            },
            "body": {
                "node_id": NODE_ID,
                "device_id": DEVICE_ID,
                "points": {
                    "Temp_Setpoint": 25.5,
                    "Temp_Readback": 24.2,
                    "Fan_Speed": 65,
                    "Valve_Status": 1,
                    "Filter_Pressure": 250
                },
                "timestamp": int(datetime.now().timestamp() * 1000),
                "quality": "good",
                "is_full_snapshot": True
            }
        }

        self.publish(f"edgex/data/{NODE_ID}/{DEVICE_ID}", msg_full)
        log_info("全量数据快照已发送")

        time.sleep(2)

        # 差量数据更新
        msg_delta = {
            "header": {
                "message_id": "test-msg-006",
                "message_type": "data_delta",
                "timestamp": int(datetime.now().timestamp() * 1000),
                "source": NODE_ID
            },
            "body": {
                "node_id": NODE_ID,
                "device_id": DEVICE_ID,
                "points": {
                    "Temp_Readback": 24.5,
                    "Fan_Speed": 70
                },
                "timestamp": int(datetime.now().timestamp() * 1000),
                "quality": "good",
                "is_full_snapshot": False
            }
        }

        self.publish(f"edgex/data/{NODE_ID}/{DEVICE_ID}", msg_delta)
        log_info("差量数据更新已发送")

    def test_command(self):
        log_info("========== 测试命令下发流程 ==========")
        log_info("订阅命令响应: edgex/commands/response")

        msg = {
            "header": {
                "message_id": "test-msg-007",
                "message_type": "command_write",
                "timestamp": int(datetime.now().timestamp() * 1000),
                "request_id": "req-test-001"
            },
            "body": {
                "node_id": NODE_ID,
                "device_id": DEVICE_ID,
                "point_id": "Temp_Setpoint",
                "value": 26.0,
                "request_id": "req-test-001"
            }
        }

        self.publish(f"edgex/cmd/{NODE_ID}/{DEVICE_ID}/write", msg)
        log_info("命令消息已发送")

        # 等待并接收响应
        log_info("等待命令响应 (5秒)...")
        self.subscribe_and_wait("edgex/commands/response", timeout=5)

    def test_node_unregister(self):
        log_info("========== 测试节点注销流程 ==========")

        msg = {
            "header": {
                "message_id": "test-msg-008",
                "message_type": "unregister",
                "timestamp": int(datetime.now().timestamp() * 1000)
            },
            "body": {
                "node_id": NODE_ID,
                "reason": "测试注销"
            }
        }

        self.publish("edgex/nodes/unregister", msg)
        log_info("节点注销消息已发送")

    def test_all(self):
        log_info("========== 运行所有测试流程 ==========")

        self.test_node_register()
        print()
        time.sleep(1)

        self.test_node_heartbeat()
        print()
        time.sleep(1)

        self.test_device_sync()
        print()
        time.sleep(1)

        self.test_point_sync()
        print()
        time.sleep(1)

        self.test_realtime_data()
        print()
        time.sleep(1)

        self.test_command()
        print()

        log_info("========== 所有测试消息已发送 ==========")

def show_subscribe_help():
    print("")
    log_info("========== MQTT 订阅命令参考 ==========")
    print("")
    print("使用 mosquitto_sub:")
    print(f"  mosquitto_sub -h {MQTT_HOST} -p {MQTT_PORT} -t \"edgex/#\" -v")
    print("")
    print("使用 Python:")
    print("  tester = EdgeXTester()")
    print("  tester.connect()")
    print("  tester.subscribe_and_wait('edgex/#', timeout=10)")
    print("")
    print("测试步骤:")
    print("  1. 启动 EdgeOS 服务")
    print("  2. 启动 MQTT Broker (如 mosquitto)")
    print("  3. 运行测试: python test_edgex_flows.py all")
    print("  4. 在另一个终端运行订阅命令查看消息流")
    print("")

def main():
    global MQTT_HOST, MQTT_PORT, NODE_ID, DEVICE_ID

    parser = argparse.ArgumentParser(
        description="EdgeOS 与 EdgeX 通信测试脚本",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
可用测试流程:
  register   - 测试节点注册
  heartbeat  - 测试心跳
  devices    - 测试设备列表同步
  points     - 测试点位列表同步
  data       - 测试实时数据更新
  command    - 测试命令下发
  unregister - 测试节点注销
  subscribe  - 显示订阅命令参考
  all        - 运行所有测试 (默认)

示例:
  python test_edgex_flows.py register    # 只测试节点注册
  python test_edgex_flows.py subscribe  # 显示订阅命令
  python test_edgex_flows.py all        # 运行所有测试
        """
    )

    parser.add_argument('flow', nargs='?', default='all',
                        help='测试流程名称')
    parser.add_argument('--host', default=MQTT_HOST,
                        help=f'MQTT Broker 地址 (默认: {MQTT_HOST})')
    parser.add_argument('--port', type=int, default=MQTT_PORT,
                        help=f'MQTT Broker 端口 (默认: {MQTT_PORT})')
    parser.add_argument('--node-id', default=NODE_ID,
                        help=f'测试节点ID (默认: {NODE_ID})')
    parser.add_argument('--device-id', default=DEVICE_ID,
                        help=f'测试设备ID (默认: {DEVICE_ID})')

    args = parser.parse_args()

    # 更新全局配置
    MQTT_HOST = args.host
    MQTT_PORT = args.port
    NODE_ID = args.node_id
    DEVICE_ID = args.device_id

    print("")
    log_info("EdgeOS 与 EdgeX 通信测试脚本")
    log_info(f"MQTT Broker: {MQTT_HOST}:{MQTT_PORT}")
    log_info(f"测试节点: {NODE_ID}")
    log_info(f"测试设备: {DEVICE_ID}")
    print("")

    tester = EdgeXTester(MQTT_HOST, MQTT_PORT)

    if not tester.connect():
        sys.exit(1)

    try:
        if args.flow == 'register':
            tester.test_node_register()
        elif args.flow == 'heartbeat':
            tester.test_node_heartbeat()
        elif args.flow == 'devices':
            tester.test_device_sync()
        elif args.flow == 'points':
            tester.test_point_sync()
        elif args.flow == 'data':
            tester.test_realtime_data()
        elif args.flow == 'command':
            tester.test_command()
        elif args.flow == 'unregister':
            tester.test_node_unregister()
        elif args.flow == 'subscribe':
            show_subscribe_help()
        else:
            tester.test_all()

    except KeyboardInterrupt:
        log_warn("测试被用户中断")
    finally:
        tester.disconnect()
        log_info("已断开 MQTT 连接")

if __name__ == "__main__":
    main()
