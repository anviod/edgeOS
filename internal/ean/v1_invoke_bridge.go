package ean

// V1InvokeBridge 已移除（OS-P3-01）
// V1 Fallback 机制不再需要：EdgeX 北向 EAN Runtime 已提供完整原生 Capability，
// 所有写操作通过 EAN Invoke 直接下发，无需降级到 V1 命令协议。