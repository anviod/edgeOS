# MQTT/NATS 实现完成及测试指南

## 功能

### 1. 消息中间件客户端
- ✅ MQTT Client (`internal/mqtt/client.go`)
- ✅ NATS Client (`internal/nats/client.go`)
- ✅ 统一的消息接口 (`internal/message/message.go`)

### 2. 消息路由和处理
- ✅ 消息路由器 (`internal/message/router.go`)
- ✅ 消息处理器接口 (`internal/message/handler.go`)
- ✅ 节点注册处理器 (`internal/handlers/node.go`)
- ✅ 设备上报处理器 (`internal/handlers/device.go`)
- ✅ 点位上报处理器 (`internal/handlers/point.go`)
- ✅ 数据采集处理器 (`internal/handlers/data.go`)
- ✅ 心跳处理器 (`internal/handlers/heartbeat.go`)
- ✅ 告警处理器 (`internal/handlers/alert.go`)

### 3. 服务层
- ✅ 注册服务 (`internal/services/registry_service.go`)
- ✅ 数据服务 (`internal/services/data_service.go`)
- ✅ 告警服务 (`internal/services/alert_service.go`)

### 4. 消息管理器
- ✅ 消息管理器 (`internal/messaging/manager.go`)
- ✅ 主程序集成 (`cmd/main.go`)
- ✅ 数据库桶初始化

### 5. 配置文件
- ✅ MQTT/NATS 配置 (`config/edgex_mqtt_nats.yaml`)
- ✅ 配置结构体 (`internal/config/edgex_mqtt_nats.go`)

### 6. 测试脚本
- ✅ Bash 测试脚本 (`test_mqtt_nats.sh`)
- ✅ PowerShell 测试脚本 (`test_mqtt_nats.ps1`)

## 测试步骤

### 前置条件

1. **安装 MQTT Broker (Mosquitto)**
   
   Windows:
   ```
   # 下载并安装 Mosquitto
   # https://mosquitto.org/download/
   
   # 或使用 Chocolatey
   choco install mosquitto
   ```

   Linux/Mac:
   ```bash
   # 使用 Docker
   docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto:2.0
   
   # 或直接安装
   sudo apt-get install mosquitto mosquitto-clients  # Ubuntu/Debian
   brew install mosquitto  # macOS
   ```

2. **启动 MQTT Broker**
   
   Windows:
   ```powershell
   mosquitto -v -c mosquitto.conf
   ```
   
   Docker:
   ```bash
   docker start mosquitto
   ```

3. **启动 EdgeOS**
   
   ```powershell
   cd d:\code\edgeOS
   .\edgeos.exe
   ```

### 运行测试

#### Windows PowerShell:

```powershell
cd d:\code\edgeOS
.\test_mqtt_nats.ps1
```

#### Linux/Mac Bash:

```bash
cd /path/to/edgeOS
chmod +x test_mqtt_nats.sh
./test_mqtt_nats.sh
```

### 测试场景

测试脚本包含以下场景:

1. **节点注册测试**
   - 发送节点注册消息到 `edgex/nodes/register`
   - 验证节点被正确注册到数据库

2. **设备上报测试**
   - 发送设备列表到 `edgex/devices/report`
   - 验证设备被同步到数据库

3. **点位上报测试**
   - 发送点位列表到 `edgex/points/report`
   - 验证点位被同步到数据库

4. **心跳测试**
   - 发送多条心跳消息到 `edgex/nodes/{node_id}/heartbeat`
   - 验证心跳时间戳被更新

5. **数据采集测试**
   - 发送实时数据到 `edgex/data/{node_id}/{device_id}`
   - 验证数据被存储到数据库

6. **告警测试**
   - 发送告警消息到 `edgex/events/alert`
   - 验证告警被存储到数据库

## 验证结果

### 检查数据库

使用 BoltDB 浏览器或编写 Go 程序检查数据库:

```go
package main

import (
    "fmt"
    "go.etcd.io/bbolt"
)

func main() {
    db, err := bbolt.Open("./data/edgeos.db", 0600, nil)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // 查看所有桶
    db.View(func(tx *bbolt.Tx) error {
        return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
            fmt.Printf("Bucket: %s\n", string(name))
            
            // 统计记录数
            count := 0
            b.ForEach(func(k, v []byte) error {
                count++
                return nil
            })
            fmt.Printf("Records: %d\n\n", count)
            return nil
        })
    })
}
```

### 预期结果

测试成功后,数据库中应该包含以下桶和记录:

- `edgex_nodes`: 1 条节点记录
- `edgex_devices`: 1 条设备记录
- `edgex_points`: 1 条点位记录
- `edgex_data`: 多条数据记录
- `edgex_alerts`: 1 条告警记录

## 当前可用设备

根据系统配置,当前可以进行以下测试:

1. **模拟 EdgeX 节点**
   - 使用测试脚本模拟 EdgeX 节点发送消息
   - 无需真实的 EdgeX 环境

2. **集成现有 EdgeX**
   - 修改 EdgeX 配置,使其连接到本 MQTT Broker
   - 配置 EdgeX 发送节点注册、设备上报等消息

3. **开发调试**
   - 查看 `edgeos_stdout.log` 和 `edgeos_stderr.log`
   - 使用 MQTT 客户端工具(如 MQTT Explorer)监听消息

## 故障排查

### 1. EdgeOS 无法启动

检查:
- 数据库目录权限: `d:\code\edgeOS\data\`
- 配置文件是否存在: `d:\code\edgeOS\config.yaml`
- 端口是否被占用: 8000

### 2. MQTT 连接失败

检查:
- MQTT Broker 是否运行: `mosquitto -v`
- 端口是否开放: `netstat -an | findstr 1883`
- 配置文件中的 broker 地址是否正确

### 3. 消息未被处理

检查:
- Topic 格式是否正确
- 消息格式是否符合规范
- 查看日志中的错误信息

## 下一步建议

1. **UI 集成**
   - 在前端添加节点、设备、点位列表页面
   - 实时显示数据采集结果
   - 告警通知功能

2. **命令发送**
   - 实现命令发送功能(设备发现、任务创建等)
   - 支持从 Web UI 发送控制命令

3. **性能优化**
   - 添加消息批量处理
   - 实现连接池
   - 数据库查询优化

4. **监控和告警**
   - 添加 Prometheus 指标
   - 实现健康检查接口
   - 集成告警通知系统

## 架构说明

### 消息流程

```
EdgeX 节点 → MQTT Broker → EdgeOS MQTT Client → 消息路由器 → 处理器 → 服务层 → 数据库
```

### 组件关系

```
main.go
  ├─→ MessagingManager
  │     ├─→ MQTTClient / NATSClient
  │     ├─→ MessageRouter
  │     │     └─→ Handlers (Node, Device, Point, Data, Heartbeat, Alert)
  │     │           └─→ Services (Registry, Data, Alert)
  │     │                 └─→ BoltDB
  │     └─→ Heartbeat Monitor
  └─→ HTTP Server (Fiber)
```

## 相关文档

- MQTT/NATS 实现指南: `docs/EdgeOS端MQTT-NATS实现指南.md`
- EdgeX 上报协议规范: `TODO/EdgeX上报到EdgeOS通信协议规范(MQTT-NATS).md`
- 配置文件: `config/edgex_mqtt_nats.yaml`
