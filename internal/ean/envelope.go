package ean

import (
	"encoding/json"
)

// unwrapBody 从 EAN 信封中提取 body。
// 兼容三种形态：
//  1. 完整信封 {"header":..., "body": ...}
//  2. 仅 body 对象
//  3. 已是目标结构的裸 JSON（原样返回）
func unwrapBody(payload []byte) json.RawMessage {
	var env struct {
		Header *MessageHeader  `json:"header"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return payload
	}
	if env.Header != nil && len(env.Body) > 0 && string(env.Body) != "null" {
		return env.Body
	}
	return payload
}

// parseJSON 先尝试信封解包，再反序列化到目标类型。
func parseJSON(payload []byte, dest interface{}) error {
	body := unwrapBody(payload)
	return json.Unmarshal(body, dest)
}
