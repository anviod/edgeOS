#!/bin/bash
# EdgeOS 与 EdgeX 通信测试脚本
# 用法: ./test_edgex_flows.sh [flow_name]
# flow_name: register|heartbeat|devices|points|data|command|all
#
# 前提条件:
# 1. MQTT Broker 运行在 127.0.0.1:1883
# 2. EdgeOS 服务运行并连接到 MQTT Broker
# 3. mosquitto_pub 工具已安装

MQTT_HOST="${MQTT_HOST:-127.0.0.1}"
MQTT_PORT="${MQTT_PORT:-1883}"
NODE_ID="${NODE_ID:-edgex-node-001}"
DEVICE_ID="${DEVICE_ID:-Room_FC_2014_19}"
TOPIC_PREFIX="edgex"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 测试节点注册流程
test_node_register() {
    log_info "========== 测试节点注册流程 =========="
    log_info "订阅主题: edgex/nodes/register (接收注册请求)"
    log_info "发布主题: edgex/nodes/register (模拟 EdgeX 节点注册)"
    log_info "订阅主题: edgex/nodes/${NODE_ID}/response (接收注册响应)"

    # 模拟 EdgeX 节点发布注册消息
    REGISTER_MSG='{
        "header": {
            "message_id": "test-msg-001",
            "message_type": "register",
            "timestamp": 1713350400000
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "node_name": "测试边缘节点",
            "protocol": "mqtt",
            "address": "192.168.1.100",
            "port": 1883,
            "description": "这是一个测试边缘节点"
        }
    }'

    log_info "发布节点注册消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/nodes/register" -m "$REGISTER_MSG" -d
    log_info "节点注册消息已发送"
    log_info "请检查 EdgeOS 日志确认节点是否注册成功"
}

# 测试心跳流程
test_node_heartbeat() {
    log_info "========== 测试节点心跳流程 =========="
    log_info "发布主题: edgex/nodes/${NODE_ID}/heartbeat"

    HEARTBEAT_MSG='{
        "header": {
            "message_id": "test-msg-002",
            "message_type": "heartbeat",
            "timestamp": 1713350400000
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "status": "online"
        }
    }'

    log_info "发布心跳消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/nodes/${NODE_ID}/heartbeat" -m "$HEARTBEAT_MSG" -d
    log_info "心跳消息已发送"
}

# 测试设备列表同步流程
test_device_sync() {
    log_info "========== 测试设备列表同步流程 =========="
    log_info "发布主题: edgex/devices/report"

    DEVICE_MSG='{
        "header": {
            "message_id": "test-msg-003",
            "message_type": "device_report",
            "timestamp": 1713350400000,
            "source": "'${NODE_ID}'"
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "devices": [
                {
                    "device_id": "'${DEVICE_ID}'",
                    "device_name": "房间空调机组2014-19",
                    "protocol": "BACnet/IP",
                    "description": "19层2014房间的空调机组",
                    "manufacturer": "Trane",
                    "model": "Tracer SC",
                    "online": true
                },
                {
                    "device_id": "Lighting_2014",
                    "device_name": "2014房间照明",
                    "protocol": "BACnet/IP",
                    "description": "19层2014房间的照明控制",
                    "online": true
                }
            ]
        }
    }'

    log_info "发布设备列表同步消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/devices/report" -m "$DEVICE_MSG" -d
    log_info "设备列表同步消息已发送"
}

# 测试点位列表同步流程
test_point_sync() {
    log_info "========== 测试点位列表同步流程 =========="
    log_info "发布主题: edgex/points/report"

    POINT_MSG='{
        "header": {
            "message_id": "test-msg-004",
            "message_type": "point_report",
            "timestamp": 1713350400000,
            "source": "'${NODE_ID}'"
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "device_id": "'${DEVICE_ID}'",
            "points": [
                {
                    "point_id": "Temp_Setpoint",
                    "point_name": "温度设定",
                    "device_id": "'${DEVICE_ID}'",
                    "service_name": "BACnet-Service",
                    "profile_name": "AHU-Profile",
                    "point_type": "Float",
                    "data_type": "Float",
                    "read_write": true,
                    "default_value": 24.0,
                    "units": "C",
                    "description": "房间温度设定值"
                },
                {
                    "point_id": "Temp_Readback",
                    "point_name": "温度回风",
                    "device_id": "'${DEVICE_ID}'",
                    "service_name": "BACnet-Service",
                    "profile_name": "AHU-Profile",
                    "point_type": "Float",
                    "data_type": "Float",
                    "read_write": false,
                    "units": "C",
                    "description": "回风温度"
                },
                {
                    "point_id": "Fan_Speed",
                    "point_name": "风机转速",
                    "device_id": "'${DEVICE_ID}'",
                    "service_name": "BACnet-Service",
                    "profile_name": "AHU-Profile",
                    "point_type": "Int",
                    "data_type": "Int64",
                    "read_write": true,
                    "default_value": 50,
                    "units": "%",
                    "description": "风机转速百分比"
                }
            ]
        }
    }'

    log_info "发布点位列表同步消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/points/report" -m "$POINT_MSG" -d
    log_info "点位列表同步消息已发送"
}

# 测试实时数据更新流程
test_realtime_data() {
    log_info "========== 测试实时数据更新流程 =========="
    log_info "发布主题: edgex/data/${NODE_ID}/${DEVICE_ID}"

    # 全量数据快照
    DATA_MSG_FULL='{
        "header": {
            "message_id": "test-msg-005",
            "message_type": "data_full",
            "timestamp": '$(date +%s000)',
            "source": "'${NODE_ID}'"
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "device_id": "'${DEVICE_ID}'",
            "points": {
                "Temp_Setpoint": 25.5,
                "Temp_Readback": 24.2,
                "Fan_Speed": 65,
                "Valve_Status": 1,
                "Filter_Pressure": 250
            },
            "timestamp": '$(date +%s000)',
            "quality": "good",
            "is_full_snapshot": true
        }
    }'

    log_info "发布全量数据快照..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/data/${NODE_ID}/${DEVICE_ID}" -m "$DATA_MSG_FULL" -d

    # 差量数据更新
    sleep 2
    DATA_MSG_DELTA='{
        "header": {
            "message_id": "test-msg-006",
            "message_type": "data_delta",
            "timestamp": '$(date +%s000)',
            "source": "'${NODE_ID}'"
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "device_id": "'${DEVICE_ID}'",
            "points": {
                "Temp_Readback": 24.5,
                "Fan_Speed": 70
            },
            "timestamp": '$(date +%s000)',
            "quality": "good",
            "is_full_snapshot": false
        }
    }'

    log_info "发布差量数据更新..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/data/${NODE_ID}/${DEVICE_ID}" -m "$DATA_MSG_DELTA" -d
    log_info "实时数据更新消息已发送"
}

# 测试命令下发流程
test_command() {
    log_info "========== 测试命令下发流程 =========="
    log_info "订阅主题: edgex/commands/response (接收命令响应)"

    # 先订阅命令响应
    log_info "请在另一个终端执行: mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/commands/response\" -v"

    COMMAND_MSG='{
        "header": {
            "message_id": "test-msg-007",
            "message_type": "command_write",
            "timestamp": '$(date +%s000)',
            "request_id": "req-test-001"
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "device_id": "'${DEVICE_ID}'",
            "point_id": "Temp_Setpoint",
            "value": 26.0,
            "request_id": "req-test-001"
        }
    }'

    log_info "发布命令消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/cmd/${NODE_ID}/${DEVICE_ID}/write" -m "$COMMAND_MSG" -d
    log_info "命令消息已发送"
    log_info "请检查 EdgeX 是否收到命令并响应"
}

# 测试节点注销流程
test_node_unregister() {
    log_info "========== 测试节点注销流程 =========="
    log_info "发布主题: edgex/nodes/unregister"

    UNREGISTER_MSG='{
        "header": {
            "message_id": "test-msg-008",
            "message_type": "unregister",
            "timestamp": '$(date +%s000)'
        },
        "body": {
            "node_id": "'${NODE_ID}'",
            "reason": "测试注销"
        }
    }'

    log_info "发布节点注销消息..."
    mosquitto_pub -h $MQTT_HOST -p $MQTT_PORT -t "edgex/nodes/unregister" -m "$UNREGISTER_MSG" -d
    log_info "节点注销消息已发送"
}

# 运行所有测试
test_all() {
    log_info "========== 运行所有测试流程 =========="
    test_node_register
    echo ""
    sleep 2
    test_node_heartbeat
    echo ""
    sleep 2
    test_device_sync
    echo ""
    sleep 2
    test_point_sync
    echo ""
    sleep 2
    test_realtime_data
    echo ""
    sleep 2
    test_command
    echo ""
    log_info "========== 所有测试消息已发送 =========="
}

# 显示订阅命令帮助
show_subscribe_help() {
    log_info "========== MQTT 订阅命令参考 =========="
    echo ""
    echo "订阅所有 EdgeX 相关主题:"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/#\" -v"
    echo ""
    echo "分别订阅各个主题:"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/nodes/register\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/nodes/+/heartbeat\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/devices/report\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/points/report\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/data/#\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/cmd/+/+/write\" -v"
    echo "  mosquitto_sub -h $MQTT_HOST -p $MQTT_PORT -t \"edgex/commands/response\" -v"
    echo ""
}

# 主函数
main() {
    echo ""
    log_info "EdgeOS 与 EdgeX 通信测试脚本"
    log_info "MQTT Broker: $MQTT_HOST:$MQTT_PORT"
    log_info "测试节点: $NODE_ID"
    log_info "测试设备: $DEVICE_ID"
    echo ""

    case "${1:-all}" in
        register)
            test_node_register
            ;;
        heartbeat)
            test_node_heartbeat
            ;;
        devices)
            test_device_sync
            ;;
        points)
            test_point_sync
            ;;
        data)
            test_realtime_data
            ;;
        command)
            test_command
            ;;
        unregister)
            test_node_unregister
            ;;
        subscribe)
            show_subscribe_help
            ;;
        all)
            test_all
            ;;
        *)
            echo "用法: $0 [flow_name]"
            echo ""
            echo "可用测试流程:"
            echo "  register   - 测试节点注册"
            echo "  heartbeat  - 测试心跳"
            echo "  devices    - 测试设备列表同步"
            echo "  points     - 测试点位列表同步"
            echo "  data       - 测试实时数据更新"
            echo "  command    - 测试命令下发"
            echo "  unregister - 测试节点注销"
            echo "  subscribe  - 显示订阅命令参考"
            echo "  all        - 运行所有测试 (默认)"
            echo ""
            echo "示例:"
            echo "  $0 register      # 只测试节点注册"
            echo "  $0 subscribe     # 显示订阅命令"
            echo "  $0 all           # 运行所有测试"
            ;;
    esac
}

main "$@"
