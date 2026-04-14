# EdgeX 端设备与 edgeOS 发现协议开发测试方案

> 适用版本：edgeOS v1  
> 文档类型：EdgeX 端开发参考 + 可执行测试方案  
> 更新日期：2026-04-14

---

## 一、背景与目标

edgeOS 通过以下链路自动发现并管理局域网内的 EdgeX 网关节点：

```
EdgeX 节点 mDNS 广播
    → edgeOS discoverer 捕获候选
    → POST 握手认证
    → 节点入注册表
    → 心跳保活（每 10s）
    → 状态机驱动 Online / Unstable / Offline / Quarantine
```

**EdgeX 端只需实现两件事：**

1. 广播标准 mDNS 服务记录（`_gateway._tcp`）
2. 提供一个 HTTP 接口（握手 + 心跳复用同一路径）

本文档面向 EdgeX 端开发人员，给出完整的协议规格、可运行参考实现和逐步验证脚本。

---

## 二、当前 edgeOS 侧参数（固定值，EdgeX 端无需配置）

| 参数 | 当前值 | 来源 |
|------|--------|------|
| edgeOS NodeID | `node-001` | `config.yaml → node.node_id` |
| edgeOS Instance | `edgeos-queen.local` | `cmd/main.go` 硬编码 |
| Shared Secret | `edgeos-shared-secret` | `cmd/main.go` 硬编码 |
| mDNS 监听类型 | `_gateway._tcp` | `discoverer.go:browseMDNS()` |
| 握手超时 | 10s | `HandshakeClient.httpClient.Timeout` |
| 心跳间隔 | 10s | `HeartbeatNode.HeartbeatIntervalMs = 10000` |
| 心跳超时 | 3s | `Heartbeat()` context timeout |
| 去重窗口（防抖） | 3s | `discoverer.go:processCandidate()` |
| 重握手最短间隔 | 30s | `service.go:minReRegisterInterval` |

> **签名密钥** `edgeos-shared-secret` 用于 HMAC-SHA256 签名验证，EdgeX 端可以忽略校验（直接返回 200），但需要了解其格式以供后续生产强化。

---

## 三、EdgeX 端 mDNS 广播规格

### 3.1 必须广播的服务记录

| 字段 | 要求 | 示例值 |
|------|------|--------|
| 服务类型 | **固定** `_gateway._tcp` | `_gateway._tcp.local.` |
| 域 | **固定** `local.` | — |
| 实例名 | EdgeX 节点可读名称，建议使用主机名 | `LAPTOP-5E3D21EG` |
| 端口 | EdgeX HTTP 服务端口 | `8082` |
| 主机名（SRV 目标） | `<实例名>.local.` | `LAPTOP-5E3D21EG.local.` |
| IPv4 地址 | **必须有**，否则 edgeOS 丢弃该候选 | `192.168.1.100` |
| TXT 记录 | 至少包含 `model=` 和 `version=` | `model=edge-gateway`, `version=1.0` |

### 3.2 TXT 记录格式

```
model=edge-gateway
version=1.0.0
```

> TXT 内容变化（任意字段）会触发 edgeOS 重新握手，可利用此机制做版本更新通知。

### 3.3 注意事项

- **必须有 IPv4 地址**：`discoverer.go` 仅提取 `entry.AddrIPv4`，IPv6-only 节点不会被发现。
- 若多张网卡，建议在 TXT 里加 `iface=eth0` 辅助说明，但 edgeOS 目前不强制使用。
- mDNS 广播建议持续存活，不要在服务就绪前停止广播。

---

## 四、握手协议规格

edgeOS 在捕获 mDNS 候选后，向 EdgeX 节点发起 HTTP POST 握手。

### 4.1 握手端点

```
POST http://<EdgeX_IP>:<EdgeX_Port>/api/v1/edgeos/handshake
Content-Type: application/json
```

### 4.2 请求体结构（HandshakeRequest）

```json
{
  "edgeosNodeId":       "node-001",
  "edgeosInstance":     "edgeos-queen.local",
  "timestamp":          1713000000000,
  "nonce":              "1713000000000000000",
  "signature":          "Base64(HMAC-SHA256(canonicalPayload, sharedSecret))",
  "supportedFeatures":  ["shadow-sync", "ws-stream", "heartbeat"],
  "expectedApiVersion": "v1"
}
```

**签名算法（EdgeX 端如需验证）：**

```
canonicalPayload = edgeosNodeId + "|" + edgeosInstance + "|" + timestamp + "|" + nonce
signature        = Base64StdEncoding( HMAC-SHA256(canonicalPayload, "edgeos-shared-secret") )
```

> 开发/测试阶段：EdgeX 端可以**跳过签名验证**，直接返回握手响应。  
> 生产阶段：建议验证签名，并拒绝时间戳偏差超过 30 秒的请求。

### 4.3 响应体结构（HandshakeResponse）

EdgeX 端必须返回 **HTTP 200** + 以下 JSON：

```json
{
  "nodeId":            "edgex-node-laptop-5e3d21eg",
  "nodeName":          "LAPTOP-5E3D21EG",
  "instance":          "LAPTOP-5E3D21EG",
  "version":           "1.0.0",
  "model":             "edge-gateway",
  "apiBase":           "http://192.168.1.100:8082",
  "streamEndpoint":    "/api/v1/stream",
  "snapshotEndpoint":  "/api/v1/snapshot",
  "heartbeatEndpoint": "/api/v1/edgeos/handshake",
  "tokenType":         "Bearer",
  "accessToken":       "",
  "expiresIn":         3600,
  "capabilities":      ["shadow-sync", "heartbeat"],
  "serverTime":        1713000000000
}
```

### 4.4 响应字段说明

| 字段 | 是否必填 | 说明 |
|------|----------|------|
| `nodeId` | **必填** | 全局唯一且稳定，edgeOS 以此为持久化 key，重启不能变 |
| `nodeName` | **必填** | 显示用可读名称 |
| `instance` | **必填** | 建议与 mDNS 实例名保持一致 |
| `version` | 推荐 | 节点软件版本号 |
| `model` | 推荐 | 硬件/软件型号 |
| `apiBase` | 推荐 | 为空时 edgeOS 自动用 `http://<发现IP>:<发现Port>` 填充 |
| `heartbeatEndpoint` | 推荐 | 为空时默认 `/api/v1/edgeos/handshake` |
| `accessToken` | 可选 | 若非空，心跳请求携带 `Authorization: Bearer <token>` |
| `capabilities` | 可选 | 能力集，目前仅做展示记录 |
| `serverTime` | 可选 | Unix 毫秒时间戳，供时钟对齐参考 |

### 4.5 失败响应

| 情况 | 建议状态码 | edgeOS 行为 |
|------|-----------|------------|
| 签名验证失败 | 401 | 记录日志，等待下次 mDNS 触发重试 |
| 版本不兼容 | 400 | 不入注册表，记录日志 |
| 服务内部错误 | 500 | 同上 |
| 连接超时/拒绝 | — | 视为握手失败，日志输出 `Handshake failed for <IP>` |

---

## 五、心跳协议规格

握手成功后，edgeOS 每 **10 秒** GET 一次心跳端点。

### 5.1 心跳请求

```
GET http://<apiBase><heartbeatEndpoint>
Authorization: Bearer <accessToken>   // 仅当 accessToken 非空时携带
```

默认心跳端点为 `/api/v1/edgeos/handshake`（与握手复用同一路径，仅方法不同）。

### 5.2 心跳响应

```
HTTP 200 OK
（响应体内容不限，edgeOS 只检查状态码）
```

### 5.3 心跳失败状态机

| 连续失败次数 | 节点状态 | 退避时间 |
|------------|---------|---------|
| 1–2 次 | `Unstable`（不稳定） | 1s, 2s |
| 3–9 次 | `Offline`（离线） | 4s → 256s |
| ≥ 10 次 | `Quarantine`（隔离） | 最大 300s |

> 心跳成功一次即可从 `Unstable`/`Offline` 恢复为 `Online`。

---

## 六、参考实现（可直接运行）

### 6.1 Python 实现（推荐快速验证）

将以下内容保存为 `edgex_peer.py`，在 EdgeX 节点上运行：

```python
"""
edgex_peer.py — EdgeX 端 edgeOS 发现协议最小参考实现
依赖：pip install zeroconf
"""
import json
import socket
import time
from http.server import BaseHTTPRequestHandler, HTTPServer

# ========== 修改此处 ==========
NODE_ID   = "edgex-node-laptop-5e3d21eg"   # 必须唯一且稳定
NODE_NAME = "LAPTOP-5E3D21EG"              # 与 mDNS 实例名保持一致
PORT      = 8082
MODEL     = "edge-gateway"
VERSION   = "1.0.0"
# ==============================


def get_local_ip():
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        s.connect(("8.8.8.8", 80))
        return s.getsockname()[0]
    finally:
        s.close()


LOCAL_IP = get_local_ip()


class EdgeXHandler(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        print(f"[{self.log_date_time_string()}] [{self.address_string()}] {fmt % args}")

    def do_GET(self):
        if self.path == "/api/v1/edgeos/handshake":
            # 心跳探活 — 直接 200
            self.send_response(200)
            self.end_headers()
        else:
            self.send_response(404)
            self.end_headers()

    def do_POST(self):
        if self.path == "/api/v1/edgeos/handshake":
            # 读取请求体（不强制校验，仅打印）
            length = int(self.headers.get("Content-Length", 0))
            body = self.rfile.read(length) if length > 0 else b""
            try:
                req = json.loads(body)
                print(f"  [Handshake] edgeosNodeId={req.get('edgeosNodeId')} "
                      f"instance={req.get('edgeosInstance')}")
            except Exception:
                pass

            resp = {
                "nodeId":            NODE_ID,
                "nodeName":          NODE_NAME,
                "instance":          NODE_NAME,
                "version":           VERSION,
                "model":             MODEL,
                "apiBase":           f"http://{LOCAL_IP}:{PORT}",
                "streamEndpoint":    "/api/v1/stream",
                "snapshotEndpoint":  "/api/v1/snapshot",
                "heartbeatEndpoint": "/api/v1/edgeos/handshake",
                "tokenType":         "Bearer",
                "accessToken":       "",
                "expiresIn":         3600,
                "capabilities":      ["shadow-sync", "heartbeat"],
                "serverTime":        int(time.time() * 1000),
            }
            data = json.dumps(resp).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)
        else:
            self.send_response(404)
            self.end_headers()


def start_mdns(zc_holder):
    try:
        from zeroconf import Zeroconf, ServiceInfo
        zc = Zeroconf()
        info = ServiceInfo(
            "_gateway._tcp.local.",
            f"{NODE_NAME}._gateway._tcp.local.",
            addresses=[socket.inet_aton(LOCAL_IP)],
            port=PORT,
            properties={"model": MODEL, "version": VERSION},
            server=f"{NODE_NAME}.local.",
        )
        zc.register_service(info)
        zc_holder["zc"] = zc
        zc_holder["info"] = info
        print(f"[mDNS] Broadcasting {NODE_NAME}._gateway._tcp.local. -> {LOCAL_IP}:{PORT}")
    except ImportError:
        print("[mDNS] zeroconf not installed: pip install zeroconf  (跳过 mDNS 广播)")
    except Exception as e:
        print(f"[mDNS] Failed: {e}")


if __name__ == "__main__":
    zc_holder = {}
    start_mdns(zc_holder)

    server = HTTPServer(("0.0.0.0", PORT), EdgeXHandler)
    print(f"[HTTP] EdgeX peer node listening on 0.0.0.0:{PORT}")
    print(f"[INFO] Local IP: {LOCAL_IP}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[INFO] Shutting down...")
        if zc_holder.get("zc"):
            zc_holder["zc"].unregister_service(zc_holder["info"])
            zc_holder["zc"].close()
```

**运行：**

```bash
pip install zeroconf
python edgex_peer.py
```

---

### 6.2 Go 实现（与 edgeOS 同技术栈）

将以下内容保存为 `edgex_peer/main.go`：

```go
// edgex_peer/main.go — EdgeX 端 edgeOS 发现协议参考实现
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net"
    "net/http"
    "time"

    "github.com/grandcat/zeroconf"
)

// ========== 修改此处 ==========
const (
    NodeID   = "edgex-node-laptop-5e3d21eg" // 必须唯一且稳定
    NodeName = "LAPTOP-5E3D21EG"            // 与 mDNS 实例名保持一致
    Port     = 8082
    Model    = "edge-gateway"
    Version  = "1.0.0"
)
// ==============================

type HandshakeRequest struct {
    EdgeOSNodeID      string   `json:"edgeosNodeId"`
    EdgeOSInstance    string   `json:"edgeosInstance"`
    Timestamp         int64    `json:"timestamp"`
    Nonce             string   `json:"nonce"`
    Signature         string   `json:"signature"`
    SupportedFeatures []string `json:"supportedFeatures"`
    ExpectedApiVersion string  `json:"expectedApiVersion"`
}

type HandshakeResponse struct {
    NodeID            string   `json:"nodeId"`
    NodeName          string   `json:"nodeName"`
    Instance          string   `json:"instance"`
    Version           string   `json:"version"`
    Model             string   `json:"model"`
    APIBase           string   `json:"apiBase"`
    StreamEndpoint    string   `json:"streamEndpoint"`
    SnapshotEndpoint  string   `json:"snapshotEndpoint"`
    HeartbeatEndpoint string   `json:"heartbeatEndpoint"`
    TokenType         string   `json:"tokenType"`
    AccessToken       string   `json:"accessToken"`
    ExpiresIn         int      `json:"expiresIn"`
    Capabilities      []string `json:"capabilities"`
    ServerTime        int64    `json:"serverTime"`
}

func getLocalIP() string {
    conn, err := net.Dial("udp", "8.8.8.8:80")
    if err != nil {
        return "127.0.0.1"
    }
    defer conn.Close()
    return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func handshake(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/api/v1/edgeos/handshake" {
        http.NotFound(w, r)
        return
    }

    // GET → 心跳探活
    if r.Method == http.MethodGet {
        w.WriteHeader(http.StatusOK)
        return
    }

    // POST → 握手
    if r.Method == http.MethodPost {
        var req HandshakeRequest
        json.NewDecoder(r.Body).Decode(&req)
        log.Printf("[Handshake] edgeosNodeId=%s instance=%s from=%s",
            req.EdgeOSNodeID, req.EdgeOSInstance, r.RemoteAddr)

        localIP := getLocalIP()
        resp := HandshakeResponse{
            NodeID:            NodeID,
            NodeName:          NodeName,
            Instance:          NodeName,
            Version:           Version,
            Model:             Model,
            APIBase:           fmt.Sprintf("http://%s:%d", localIP, Port),
            StreamEndpoint:    "/api/v1/stream",
            SnapshotEndpoint:  "/api/v1/snapshot",
            HeartbeatEndpoint: "/api/v1/edgeos/handshake",
            TokenType:         "Bearer",
            AccessToken:       "",
            ExpiresIn:         3600,
            Capabilities:      []string{"shadow-sync", "heartbeat"},
            ServerTime:        time.Now().UnixMilli(),
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
        return
    }

    http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func startMDNS(localIP string) {
    host := NodeName + ".local."
    server, err := zeroconf.RegisterProxy(
        NodeName,
        "_gateway._tcp",
        "local.",
        Port,
        host,
        []string{localIP},
        []string{"model=" + Model, "version=" + Version},
        nil,
    )
    if err != nil {
        log.Fatalf("[mDNS] Register failed: %v", err)
    }
    log.Printf("[mDNS] Broadcasting %s._gateway._tcp.local. -> %s:%d", NodeName, localIP, Port)
    _ = server
    // server.Shutdown() 在进程退出时调用
}

func main() {
    localIP := getLocalIP()
    log.Printf("[INFO] Local IP: %s", localIP)

    go startMDNS(localIP)

    http.HandleFunc("/", handshake)
    log.Printf("[HTTP] EdgeX peer node listening on :%d", Port)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", Port), nil))
}
```

**运行：**

```bash
mkdir edgex_peer && cd edgex_peer
# 将上方 main.go 保存到此目录
go mod init edgex_peer
go get github.com/grandcat/zeroconf
go run main.go
```

---

### 6.3 现有 EdgeX Foundry 项目集成（推荐方式）

若对端已有 EdgeX Foundry 服务，在其 **application-service 或 device-service** 中添加一个路由：

```go
// 在 EdgeX 服务初始化时注册路由（示例基于 echo/gin/fiber，按实际框架调整）
router.POST("/api/v1/edgeos/handshake", edgeOSHandshakeHandler)
router.GET("/api/v1/edgeos/handshake", edgeOSHeartbeatHandler)
```

响应内容同 §4.3，`nodeId` 建议使用 EdgeX 的 `core-metadata` 中的设备服务 ID，确保重启后不变。

---

## 七、测试方案

### 7.1 阶段一：接口连通性验证（无需 edgeOS）

在 EdgeX 节点或同局域网机器上执行以下命令：

**Windows PowerShell：**

```powershell
# 测试握手接口
$body = '{"edgeosNodeId":"test","edgeosInstance":"test","timestamp":1713000000000,"nonce":"abc","signature":"x","supportedFeatures":[],"expectedApiVersion":"v1"}'
Invoke-WebRequest `
  -Uri "http://LAPTOP-5E3D21EG:8082/api/v1/edgeos/handshake" `
  -Method POST `
  -ContentType "application/json" `
  -Body $body

# 测试心跳接口
Invoke-WebRequest -Uri "http://LAPTOP-5E3D21EG:8082/api/v1/edgeos/handshake" -Method GET
```

**期望结果：**
- 握手：HTTP 200，响应体包含 `nodeId` 字段
- 心跳：HTTP 200

**Linux/macOS curl：**

```bash
# 握手
curl -s -X POST http://LAPTOP-5E3D21EG:8082/api/v1/edgeos/handshake \
  -H "Content-Type: application/json" \
  -d '{"edgeosNodeId":"test","edgeosInstance":"test","timestamp":1713000000000,"nonce":"abc","signature":"x","supportedFeatures":[],"expectedApiVersion":"v1"}' \
  | python3 -m json.tool

# 心跳
curl -o /dev/null -w "%{http_code}" http://LAPTOP-5E3D21EG:8082/api/v1/edgeos/handshake
```

---

### 7.2 阶段二：mDNS 广播验证（无需 edgeOS）

在同局域网另一台机器上执行：

**Python（需安装 zeroconf）：**

```python
from zeroconf import Zeroconf, ServiceBrowser
import time

class Listener:
    def add_service(self, zc, type_, name):
        info = zc.get_service_info(type_, name)
        print(f"[发现] {name}")
        if info:
            print(f"  地址: {[str(a) for a in info.parsed_addresses()]}")
            print(f"  端口: {info.port}")
            print(f"  TXT:  {info.properties}")

zc = Zeroconf()
browser = ServiceBrowser(zc, "_gateway._tcp.local.", Listener())
print("正在监听 _gateway._tcp.local. 广播，按 Ctrl+C 退出...")
try:
    input()
except KeyboardInterrupt:
    zc.close()
```

**期望结果：** 打印出 EdgeX 节点的实例名、IP 和 TXT 记录。

---

### 7.3 阶段三：edgeOS 完整端到端验证

1. 启动 EdgeX 参考实现（§6.1 Python 或 §6.2 Go）
2. 启动 edgeOS：

   ```bash
   cd d:/code/edgeOS
   go run ./cmd/main.go
   ```

3. 观察 edgeOS 控制台日志：

   ```
   # 期望出现以下日志（约 3–5 秒内）：
   EdgeX node handshake success: LAPTOP-5E3D21EG (192.168.x.x:8082)
   
   # 每 10 秒出现心跳日志（若有状态变化）：
   Node <nodeId> state changed: ...
   ```

4. 通过 edgeOS API 确认节点已入库：

   ```powershell
   # 查询 EdgeX 节点列表（需 Bearer token，开发环境可先关闭鉴权测试）
   Invoke-WebRequest -Uri "http://localhost:8000/api/edgex/nodes" `
     -Headers @{ token = "test-token" }
   ```

   **期望响应：**
   ```json
   {
     "code": "0",
     "msg": "Success",
     "data": {
       "nodes": [
         {
           "node_id": "edgex-node-laptop-5e3d21eg",
           "node_name": "LAPTOP-5E3D21EG",
           "status": "Online",
           ...
         }
       ]
     }
   }
   ```

---

### 7.4 阶段四：心跳状态机验证

| 测试场景 | 操作 | 期望 edgeOS 状态 |
|---------|------|----------------|
| 正常运行 | 保持 EdgeX 服务在线 | `Online` |
| 短暂停止 | 停止 EdgeX HTTP 服务 1–2 个心跳周期（10–20s） | `Unstable` |
| 较长停止 | 停止 30–90s | `Offline` |
| 重新启动 | 重新启动 EdgeX HTTP 服务 | 下一次心跳成功后恢复 `Online` |
| 长期离线 | 停止 > 100s | `Quarantine`（≥10 次失败） |

可通过调用 edgeOS API 查询节点状态变化：

```powershell
# 轮询节点状态
while ($true) {
    $r = Invoke-WebRequest -Uri "http://localhost:8000/api/edgex/nodes" `
         -Headers @{ token = "test-token" } -UseBasicParsing
    Write-Host (Get-Date -Format "HH:mm:ss") ($r.Content | ConvertFrom-Json).data.nodes[0].status
    Start-Sleep 5
}
```

---

### 7.5 阶段五：TXT 变更重握手验证

修改 EdgeX 参考实现中的 `VERSION` 值（如从 `1.0.0` 改为 `1.0.1`）并重启 mDNS 广播：

**期望：** edgeOS 检测到 TXT 指纹变化，在 30 秒内重新握手，日志输出 `NodeUpdated` 事件。

---

## 八、常见问题排查

| 现象 | 可能原因 | 排查方式 |
|------|---------|---------|
| edgeOS 日志无任何发现输出 | mDNS 广播未正常工作 | 执行 §7.2 mDNS 验证 |
| 日志出现 `Handshake failed` | EdgeX 握手接口返回非 200 | 执行 §7.1 接口验证，查看 EdgeX 侧错误日志 |
| 节点入库但立即变 Offline | 心跳接口不可达（端口/路径不对） | 确认 `heartbeatEndpoint` 路径和端口，执行 §7.1 心跳验证 |
| 节点始终 Unstable | 网络有丢包或 EdgeX 响应慢（>3s） | 检查网络质量；适当延长 EdgeX 心跳响应时间 |
| 节点不断重复握手 | `nodeId` 每次不同 | **必须保证 `nodeId` 稳定**，不能使用随机数或时间戳生成 |
| mDNS 发现不到 IPv4 | IPv6-only 环境 | 在 EdgeX 侧明确绑定 IPv4 地址后再注册 mDNS |

---

## 九、协议速查表

```
┌──────────────────────────────────────────────────────────┐
│               EdgeX 端需要实现的接口                       │
├──────────────────────────────────────────────────────────┤
│  mDNS 广播                                                │
│    服务类型: _gateway._tcp.local.                         │
│    TXT:      model=edge-gateway  version=1.0.0            │
│    IPv4:     必须包含                                      │
│                                                          │
│  HTTP 接口（同一路径，两种方法）                            │
│    POST /api/v1/edgeos/handshake                         │
│         → 返回 200 + HandshakeResponse JSON              │
│    GET  /api/v1/edgeos/handshake                         │
│         → 返回 200（心跳探活，无需响应体）                  │
│                                                          │
│  HandshakeResponse 关键字段                               │
│    nodeId:            稳定唯一 ID（不能变）                │
│    apiBase:           http://<本机IP>:<Port>              │
│    heartbeatEndpoint: /api/v1/edgeos/handshake           │
└──────────────────────────────────────────────────────────┘

edgeOS 行为时序：
  mDNS 广播 → (≤3s) → 握手 POST → (成功) → 每 10s GET 心跳
  失败 1-2次 → Unstable
  失败 3-9次 → Offline   (指数退避 4s–256s)
  失败 ≥10次 → Quarantine (最大退避 300s)
  任意一次成功 → Online
```
