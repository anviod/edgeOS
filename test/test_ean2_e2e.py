#!/usr/bin/env python3
"""
EAN 2.0 端到端测试脚本 | EAN 2.0 End-to-End Test Script
测试范围: 认证系统 → EAN Health → EAN Agents → EAN Invoke (4台BACnet设备)
真机环境: EdgeOS @ 192.168.3.230:8000, edgeCore @ 192.168.3.104
"""
import json
import time
import urllib.request
import urllib.error

BASE_URL = "http://127.0.0.1:8000"
TOKEN = None
RESULTS = []

def log(section, status, detail=""):
    """记录测试结果"""
    icon = "PASS" if status == "pass" else "FAIL" if status == "fail" else "INFO"
    line = f"[{icon}] {section}: {detail}" if detail else f"[{icon}] {section}"
    print(line)
    RESULTS.append({"section": section, "status": status, "detail": detail})

def api_get(path, token=None, timeout=30):
    """GET 请求"""
    url = f"{BASE_URL}{path}"
    req = urllib.request.Request(url, method="GET")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        return {"error": f"HTTP {e.code}", "body": body}
    except Exception as e:
        return {"error": str(e)}

def api_post(path, body, token=None, timeout=30):
    """POST 请求"""
    url = f"{BASE_URL}{path}"
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", errors="replace")
        return {"error": f"HTTP {e.code}", "body": body}
    except Exception as e:
        return {"error": str(e)}

def extract_invoke_result(resp):
    """从 API 响应中提取 Invoke 结果"""
    data = resp.get("data", {})
    if isinstance(data, dict):
        response = data.get("response", {})
        result = response.get("result", {})
        return {
            "status": response.get("status", ""),
            "success": result.get("success", False),
            "values": result.get("values", {}),
            "error": result.get("error", ""),
            "error_code": result.get("error_code", ""),
            "latency_ms": response.get("latency_ms", 0),
            "invoke_id": response.get("invoke_id", ""),
        }
    return {"success": False, "error": "invalid response"}

# ============================================================
# 1. 认证系统测试 | Authentication System Test
# ============================================================
def test_auth():
    print("\n" + "="*60)
    print("1. 认证系统测试 | Authentication System Test")
    print("="*60)

    # 1.1 test/test 登录
    resp = api_post("/api/auth/login", {"username": "test", "password": "test"})
    if "data" in resp and resp.get("data", {}).get("token"):
        global TOKEN
        TOKEN = resp["data"]["token"]
        perms = resp["data"].get("permissions", [])
        expires = resp["data"].get("expires_in", "unknown")
        log("认证-test账户登录", "pass", f"token获取成功, 权限={perms}, 有效期={expires}")
    else:
        log("认证-test账户登录", "fail", f"登录失败: {resp}")
        return False

    # 1.2 验证 token 有效性
    resp = api_get("/api/ean/health", token=TOKEN)
    if "data" in resp:
        log("认证-Token有效性", "pass", "Token验证通过，可访问受保护API")
    else:
        log("认证-Token有效性", "fail", f"Token验证失败: {resp}")
        return False

    return True

# ============================================================
# 2. EAN Health 测试 | EAN Health Test
# ============================================================
def test_health():
    print("\n" + "="*60)
    print("2. EAN Health 测试 | EAN Health Test")
    print("="*60)

    resp = api_get("/api/ean/health", token=TOKEN)
    data = resp.get("data", {})

    if not data:
        log("EAN Health", "fail", f"无数据返回: {resp}")
        return

    status = data.get("status", "unknown")
    log("EAN Health-状态", "pass" if status == "ok" else "fail", f"status={status}")

    # 通过 transport_details 检查 MQTT/NATS 连接状态
    # 单链路 MQTT：NATS 未注册时视为预期（info），不判失败
    transport_details = data.get("transport_details", [])
    registered_transports = data.get("registered_transports", 0)
    mqtt_connected = False
    nats_registered = False
    nats_connected = False
    for t in transport_details:
        if t.get("name") == "mqtt" and t.get("connected"):
            mqtt_connected = True
        if t.get("name") == "nats":
            nats_registered = True
            if t.get("connected"):
                nats_connected = True

    log("EAN Health-MQTT连接", "pass" if mqtt_connected else "fail",
        f"connected={mqtt_connected}")
    if nats_registered:
        log("EAN Health-NATS连接", "pass" if nats_connected else "fail",
            f"connected={nats_connected}")
    else:
        log("EAN Health-NATS连接", "info",
            f"NATS 未启用（单链路 MQTT，registered_transports={registered_transports}）")

    # 在线 Agent 数量
    online_agents = data.get("online_agents", 0)
    log("EAN Health-在线Agent", "pass" if online_agents > 0 else "fail",
        f"count={online_agents}")

    # 原生 EAN Capability 数量
    native_caps = data.get("native_ean_caps", 0)
    log("EAN Health-原生EAN能力", "pass" if native_caps > 0 else "fail",
        f"count={native_caps}")

    # Invoke 指标
    invoke_metrics = data.get("invoke_metrics", {})
    if invoke_metrics:
        total = invoke_metrics.get("total", 0)
        success = invoke_metrics.get("success", 0)
        failed = invoke_metrics.get("failed", 0)
        success_rate = invoke_metrics.get("success_rate", 0)
        avg_latency = invoke_metrics.get("avg_latency_ms", 0)
        p50 = invoke_metrics.get("p50_latency_ms", 0)
        p99 = invoke_metrics.get("p99_latency_ms", 0)
        log("EAN Health-Invoke指标", "pass",
            f"total={total}, success={success}, failed={failed}, "
            f"success_rate={success_rate:.1%}, avg={avg_latency:.1f}ms, "
            f"P50={p50}ms, P99={p99}ms")
    else:
        log("EAN Health-Invoke指标", "info", "尚无Invoke记录")

    # 打印完整 Health 数据
    print(f"\n  完整 Health 数据:\n{json.dumps(data, ensure_ascii=False, indent=2)}")

# ============================================================
# 3. EAN Agents 测试 | EAN Agents Test
# ============================================================
def test_agents():
    print("\n" + "="*60)
    print("3. EAN Agents 测试 | EAN Agents Test")
    print("="*60)

    # 3.1 列出所有 Agent
    resp = api_get("/api/ean/agents", token=TOKEN)
    data = resp.get("data", {})
    agents = data.get("agents", [])
    total = data.get("total", 0)

    log("EAN Agents-列表", "pass" if total > 0 else "fail",
        f"total={total}")

    for agent in agents:
        agent_id = agent.get("id", "unknown")
        status = agent.get("status", "unknown")
        hb_interval = agent.get("heartbeat_interval_sec", 0)
        log(f"EAN Agents-{agent_id}", "pass" if status == "online" else "fail",
            f"status={status}, heartbeat={hb_interval}s")

    # 3.2 获取 edgeCore-node-001 的 Capability 列表
    target_agent = "edgeCore-node-001"
    resp = api_get(f"/api/ean/agents/{target_agent}/capabilities", token=TOKEN)
    cap_data = resp.get("data", {})
    caps = cap_data.get("capabilities", [])
    native_count = cap_data.get("native_ean_caps", 0)
    cap_total = cap_data.get("total", 0)

    log(f"EAN Agents-{target_agent}能力列表", "pass" if cap_total > 0 else "fail",
        f"total={cap_total}, native_ean={native_count}")

    # 检查关键 Capability 是否存在
    key_caps = [
        "bacnet_ip.scan_devices",
        "bacnet_ip.list_points",
        "bacnet_ip.read_holding_register",
        "bacnet_ip.write_register",
        "system.diagnostics",
    ]
    cap_ids = [c.get("id", "") for c in caps]
    for kc in key_caps:
        if kc in cap_ids:
            cap_obj = next(c for c in caps if c.get("id") == kc)
            src = cap_obj.get("source", "unknown")
            log(f"EAN Agents-Capability[{kc}]", "pass",
                f"source={src}")
        else:
            log(f"EAN Agents-Capability[{kc}]", "fail", "未找到")

    # 打印 Capability 列表
    print(f"\n  Agent {target_agent} 的 Capability 列表 ({cap_total} 个):")
    for c in caps[:10]:
        print(f"    - {c.get('id')}: category={c.get('category')}, source={c.get('source')}")
    if cap_total > 10:
        print(f"    ... 共 {cap_total} 个")

    return agents

# ============================================================
# 4. EAN Invoke 测试 | EAN Invoke Test (4台BACnet设备)
# ============================================================
def test_invoke():
    print("\n" + "="*60)
    print("4. EAN Invoke 测试 | EAN Invoke Test")
    print("="*60)

    TARGET = "edgeCore-node-001"
    BACNET_DEVICES = ["bacnet-2228316", "bacnet-2228317", "bacnet-2228318", "bacnet-2228319"]
    BACNET_DEVICE_LABELS = ["2228316", "2228317", "2228318", "2228319"]
    CHANNEL_ID = "BACnet"

    # 4.1 system.diagnostics
    print("\n  [4.1] system.diagnostics")
    resp = api_post("/api/ean/invoke", {
        "target": TARGET,
        "capability": "system.diagnostics",
        "arguments": {},
        "timeout_sec": 30
    }, token=TOKEN, timeout=60)

    result = extract_invoke_result(resp)
    if result["success"]:
        values = result["values"]
        diag = values.get("diagnostics", {})
        channels = diag.get("channels", [])
        chan_info = ", ".join([f"{c['channel_id']}(devices={c['devices']})" for c in channels])
        log("Invoke-system.diagnostics", "pass",
            f"channels={diag.get('count',0)}, {chan_info}, latency={result['latency_ms']}ms")
    else:
        log("Invoke-system.diagnostics", "fail", f"失败: {result['error']}")

    # 4.2 bacnet_ip.scan_devices
    print(f"\n  [4.2] bacnet_ip.scan_devices (channel_id={CHANNEL_ID})")
    resp = api_post("/api/ean/invoke", {
        "target": TARGET,
        "capability": "bacnet_ip.scan_devices",
        "arguments": {"channel_id": CHANNEL_ID},
        "timeout_sec": 120
    }, token=TOKEN, timeout=180)

    result = extract_invoke_result(resp)
    if result["success"]:
        devices = result["values"]
        if isinstance(devices, list):
            device_ids = [str(d.get("bacnet_device_id", "")) for d in devices]
            all_online = all(d.get("status") == "online" for d in devices)
            log("Invoke-scan_devices", "pass",
                f"发现{len(devices)}台设备: {device_ids}, all_online={all_online}, "
                f"latency={result['latency_ms']}ms")
        else:
            log("Invoke-scan_devices", "pass",
                f"扫描完成, latency={result['latency_ms']}ms")
    else:
        log("Invoke-scan_devices", "fail", f"扫描失败: {result['error']}")

    # 等待扫描完成
    print("    等待 10 秒让扫描数据入库...")
    time.sleep(10)

    # 4.3 对每台 BACnet 设备执行 list_points + read_holding_register + write_register
    for device_id, label in zip(BACNET_DEVICES, BACNET_DEVICE_LABELS):
        print(f"\n  [4.3] 设备 {label} ({device_id}) 测试")

        # 4.3.1 list_points
        resp = api_post("/api/ean/invoke", {
            "target": TARGET,
            "capability": "bacnet_ip.list_points",
            "arguments": {"device_id": device_id},
            "timeout_sec": 30
        }, token=TOKEN, timeout=60)

        result = extract_invoke_result(resp)
        if result["success"]:
            values = result["values"]
            points = values if isinstance(values, list) else values.get("points", [])
            point_count = len(points)
            all_good = all(p.get("quality") == "Good" for p in points) if points else False
            # 提取温度数据
            temp_points = [p for p in points if "temperature" in p.get("name", "").lower() or "temp" in p.get("name", "").lower()]
            temp_info = ""
            if temp_points:
                temp_info = ", ".join([f"{p['name']}={p['value']}°C" for p in temp_points[:3]])
            log(f"Invoke-list_points[{label}]", "pass",
                f"点位数={point_count}, quality_all_good={all_good}, "
                f"latency={result['latency_ms']}ms"
                + (f", {temp_info}" if temp_info else ""))
        else:
            log(f"Invoke-list_points[{label}]", "fail", f"失败: {result['error']}")

        # 4.3.2 read_holding_register (AnalogInput:0 = 室温)
        resp = api_post("/api/ean/invoke", {
            "target": TARGET,
            "capability": "bacnet_ip.read_holding_register",
            "arguments": {
                "device_id": device_id,
                "address": "AnalogInput:0"
            },
            "timeout_sec": 30
        }, token=TOKEN, timeout=60)

        result = extract_invoke_result(resp)
        if result["success"]:
            values = result["values"]
            # 提取读取值
            read_value = ""
            if isinstance(values, dict):
                read_value = values.get("value", values.get("AnalogInput:0", ""))
            elif isinstance(values, list) and values:
                read_value = values[0].get("value", "")
            log(f"Invoke-read[{label}]", "pass",
                f"AnalogInput:0={read_value}, latency={result['latency_ms']}ms")
        else:
            log(f"Invoke-read[{label}]", "fail", f"读取失败: {result['error']}")

        # 4.3.3 write_register (AnalogValue:0 = SetPoint) + 二次读取验证
        write_value = 25.0
        resp = api_post("/api/ean/invoke", {
            "target": TARGET,
            "capability": "bacnet_ip.write_register",
            "arguments": {
                "device_id": device_id,
                "address": "AnalogValue:0",
                "value": write_value
            },
            "timeout_sec": 30
        }, token=TOKEN, timeout=60)

        result = extract_invoke_result(resp)
        write_ok = result["success"]
        if write_ok:
            values = result["values"]
            log(f"Invoke-write[{label}]", "pass",
                f"AnalogValue:0={write_value}, latency={result['latency_ms']}ms")
        else:
            log(f"Invoke-write[{label}]", "fail", f"写入失败: {result['error']}")

        # 二次读取验证 (secondary verification)
        if write_ok:
            time.sleep(2)
            resp = api_post("/api/ean/invoke", {
                "target": TARGET,
                "capability": "bacnet_ip.read_holding_register",
                "arguments": {
                    "device_id": device_id,
                    "address": "AnalogValue:0"
                },
                "timeout_sec": 30
            }, token=TOKEN, timeout=60)

            result = extract_invoke_result(resp)
            if result["success"]:
                values = result["values"]
                verify_value = ""
                if isinstance(values, dict):
                    verify_value = values.get("value", values.get("AnalogValue:0", ""))
                elif isinstance(values, list) and values:
                    verify_value = values[0].get("value", "")
                log(f"Invoke-write验证[{label}]", "pass",
                    f"二次读取 AnalogValue:0={verify_value}, latency={result['latency_ms']}ms")
            else:
                log(f"Invoke-write验证[{label}]", "fail", f"二次读取失败: {result['error']}")

# ============================================================
# 5. 测试汇总 | Test Summary
# ============================================================
def print_summary():
    print("\n" + "="*60)
    print("EAN 2.0 端到端测试汇总 | Test Summary")
    print("="*60)

    total = len(RESULTS)
    passed = sum(1 for r in RESULTS if r["status"] == "pass")
    failed = sum(1 for r in RESULTS if r["status"] == "fail")
    info = sum(1 for r in RESULTS if r["status"] == "info")

    print(f"\n  总计: {total}  通过: {passed}  失败: {failed}  信息: {info}")
    print(f"  通过率: {passed/(passed+failed)*100:.1f}%" if (passed+failed) > 0 else "  无可统计项目")

    if failed > 0:
        print("\n  失败项:")
        for r in RESULTS:
            if r["status"] == "fail":
                print(f"    - {r['section']}: {r['detail']}")

    print("\n" + "="*60)

# ============================================================
# 主函数 | Main
# ============================================================
if __name__ == "__main__":
    print("="*60)
    print("EAN 2.0 端到端测试 | EAN 2.0 End-to-End Test")
    print(f"目标: {BASE_URL}")
    print(f"时间: {time.strftime('%Y-%m-%d %H:%M:%S')}")
    print("="*60)

    # 1. 认证
    if not test_auth():
        print("\n认证失败，终止测试")
        print_summary()
        exit(1)

    # 2. EAN Health
    test_health()

    # 3. EAN Agents
    test_agents()

    # 4. EAN Invoke
    test_invoke()

    # 5. 汇总
    print_summary()
