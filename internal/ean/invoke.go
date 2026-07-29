package ean

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ==================== 发布函数类型 ====================

// PublishFunc 抽象发布能力，便于单元测试 mock
type PublishFunc func(topic string, payload []byte) error

// ==================== InvokeOrchestrator ====================

// InvokeOrchestrator 调用编排器，负责发起 Invoke 并关联 Response
// 核心流程:
//  1. Invoke() 构建 InvokeRequest，通过 publishFn 发布到 $edgeos/invoke/{target}
//  2. 启动定时器等待 EdgeX 回复
//  3. HandleReply() 接收 $edgeos/reply/{source} 消息，按 invoke_id 关联并返回结果
//  4. 超时自动清理 pending 状态
type InvokeOrchestrator struct {
	sourceID  string // EdgeOS Planner 自身标识，用于 reply topic: $edgeos/reply/{sourceID}
	publishFn PublishFunc

	// pending 存储等待中的调用，key = invoke_id
	pending map[string]*pendingInvoke
	mu      sync.Mutex

	logger *zap.Logger
}

// pendingInvoke 等待中的 Invoke 调用状态
type pendingInvoke struct {
	invokeID string
	responseCh chan *InvokeResponse // 响应通道
	cancel     context.CancelFunc   // 用于取消定时器
}

// InvokeConfig 编排器配置
type InvokeConfig struct {
	SourceID  string     // EdgeOS Planner 标识，用于 reply topic 路由
	PublishFn PublishFunc // 发布函数
}

// InvokeCall 编排器 Invoke 返回值
type InvokeCall struct {
	Response *InvokeResponse // EdgeX 回复（成功时非 nil）
	Error    error           // 超时/发布失败等错误
}

// NewInvokeOrchestrator 创建调用编排器
func NewInvokeOrchestrator(cfg InvokeConfig, logger *zap.Logger) *InvokeOrchestrator {
	if cfg.SourceID == "" {
		cfg.SourceID = "edgeos-planner"
	}
	return &InvokeOrchestrator{
		sourceID:  cfg.SourceID,
		publishFn: cfg.PublishFn,
		pending:   make(map[string]*pendingInvoke),
		logger:    logger.Named("invoke"),
	}
}

// ==================== 发起调用 ====================

// Invoke 同步调用指定 Agent 的 Capability
// target: 目标 Agent ID
// capability: 要调用的能力 ID（如 "modbus-tcp.read_points"）
// arguments: 调用参数
// timeout: 客户端超时（覆盖 Capability 默认超时），0 则使用 cap.TimeoutSec
//
// 流程:
//  1. 生成 invoke_id（UUID）
//  2. 构造 Message 信封（message_type=invoke_capability，correlation_id=invoke_id）
//  3. 通过 publishFn 发布到 $edgeos/invoke/{target}
//  4. 等待 HandleReply 通过 invoke_id 关联回复
//  5. 超时返回错误
func (io *InvokeOrchestrator) Invoke(ctx context.Context, target, capability string, arguments map[string]interface{}, timeout time.Duration) *InvokeCall {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	invokeID := uuid.New().String()

	// 构造 InvokeRequest
	req := InvokeRequest{
		InvokeID:   invokeID,
		Target:     target,
		Capability: capability,
		Arguments:  arguments,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return &InvokeCall{Error: fmt.Errorf("marshal invoke request failed: %w", err)}
	}

	// 构造 EAN Message 信封
	msg := Message{
		Header: MessageHeader{
			MessageID:     uuid.New().String(),
			Timestamp:     time.Now().UnixMilli(),
			Source:        io.sourceID,
			MessageType:   "invoke_capability",
			Version:       "2.0",
			CorrelationID: invokeID,
		},
		Body: reqBody,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return &InvokeCall{Error: fmt.Errorf("marshal message failed: %w", err)}
	}

	// 注册 pending 等待
	respCh := make(chan *InvokeResponse, 1)
	invCtx, cancel := context.WithTimeout(ctx, timeout)
	p := &pendingInvoke{
		invokeID:   invokeID,
		responseCh: respCh,
		cancel:     cancel,
	}

	io.mu.Lock()
	io.pending[invokeID] = p
	io.mu.Unlock()

	// 发布请求
	topic := InvokeTopic(target)
	if err := io.publishFn(topic, payload); err != nil {
		io.cleanup(invokeID)
		return &InvokeCall{Error: fmt.Errorf("publish invoke failed: %w", err)}
	}

	io.logger.Info("invoke published",
		zap.String("invoke_id", invokeID),
		zap.String("target", target),
		zap.String("capability", capability),
		zap.Duration("timeout", timeout))

	// 同步等待回复或超时
	select {
	case resp := <-respCh:
		io.cleanup(invokeID)
		return &InvokeCall{Response: resp}

	case <-invCtx.Done():
		io.cleanup(invokeID)
		return &InvokeCall{Error: fmt.Errorf("invoke %s timed out after %v", invokeID, timeout)}
	}
}

// cleanup 清理 pending 状态
func (io *InvokeOrchestrator) cleanup(invokeID string) {
	io.mu.Lock()
	defer io.mu.Unlock()
	if p, ok := io.pending[invokeID]; ok {
		if p.cancel != nil {
			p.cancel()
		}
		delete(io.pending, invokeID)
	}
}

// ==================== 接收回复 ====================

// HandleReply 处理来自 EdgeX 的 Invoke 回复
// 用作 Subscribe(ReplyTopic(sourceID), orchestrator.HandleReply) 的回调
// 通过 header.correlation_id / body.invoke_id 关联原始请求
func (io *InvokeOrchestrator) HandleReply(topic string, payload []byte, transport string) {
	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		io.logger.Warn("failed to unmarshal reply message",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	var resp InvokeResponse
	if err := json.Unmarshal(msg.Body, &resp); err != nil {
		io.logger.Warn("failed to unmarshal invoke response body",
			zap.String("topic", topic), zap.Error(err))
		return
	}

	// 关联: 优先使用 body.invoke_id，回退到 header.correlation_id
	invokeID := resp.InvokeID
	if invokeID == "" {
		invokeID = msg.Header.CorrelationID
	}
	if invokeID == "" {
		io.logger.Warn("reply missing invoke_id and correlation_id",
			zap.String("topic", topic))
		return
	}

	io.mu.Lock()
	p, ok := io.pending[invokeID]
	io.mu.Unlock()

	if !ok {
		io.logger.Debug("reply for unknown/expired invoke",
			zap.String("invoke_id", invokeID),
			zap.String("status", resp.Status))
		return
	}

	// 通过 channel 返回响应（非阻塞，channel 已预留 1 缓冲）
	select {
	case p.responseCh <- &resp:
		io.logger.Info("invoke reply correlated",
			zap.String("invoke_id", invokeID),
			zap.String("status", resp.Status),
			zap.String("transport", transport))
	default:
		io.logger.Warn("response channel full, dropping reply",
			zap.String("invoke_id", invokeID))
	}
}

// ==================== 辅助方法 ====================

// PendingCount 当前等待中的调用数量
func (io *InvokeOrchestrator) PendingCount() int {
	io.mu.Lock()
	defer io.mu.Unlock()
	return len(io.pending)
}

// SourceID 返回编排器标识
func (io *InvokeOrchestrator) SourceID() string {
	return io.sourceID
}

// ReplyTopic 返回编排器应订阅的 reply topic
func (io *InvokeOrchestrator) ReplyTopic() string {
	return ReplyTopic(io.sourceID)
}

// RegisterReplySubscription 在 bus 上注册 reply 订阅
func (io *InvokeOrchestrator) RegisterReplySubscription(bus interface{ Subscribe(string, MessageHandler) error }) error {
	topic := io.ReplyTopic()
	if err := bus.Subscribe(topic, io.HandleReply); err != nil {
		return fmt.Errorf("subscribe reply topic %s failed: %w", topic, err)
	}
	io.logger.Info("reply subscription registered", zap.String("topic", topic))
	return nil
}
