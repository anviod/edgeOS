package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Client represents a NATS client
type Client struct {
	nc      *nats.Conn
	js      nats.JetStreamContext
	config  *Config
	nodeID  string
	subs    map[string]*nats.Subscription
	subsLock sync.RWMutex
	logger  Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

// Config represents NATS client configuration
type Config struct {
	URL                 string
	ClientName          string
	Username            string
	Password            string
	Token               string
	ConnectTimeout      int
	ReconnectWait       int
	MaxReconnects       int
	PingInterval        int
	MaxPingsOutstanding int
	JetStreamEnabled    bool
}

// Logger defines the logging interface
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Message represents a standardized message structure
type Message struct {
	Header MessageHeader      `json:"header"`
	Body   interface{}        `json:"body"`
}

// MessageHeader contains message metadata
type MessageHeader struct {
	MessageID    string `json:"message_id"`
	Timestamp    int64  `json:"timestamp"`
	Source       string `json:"source"`
	Destination  string `json:"destination,omitempty"`
	MessageType  string `json:"message_type"`
	Version      string `json:"version"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// NewClient creates a new NATS client
func NewClient(config *Config, nodeID string, logger Logger) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := []nats.Option{
		nats.Name(config.ClientName),
		nats.UserInfo(config.Username, config.Password),
		nats.ReconnectWait(time.Duration(config.ReconnectWait) * time.Second),
		nats.MaxReconnects(config.MaxReconnects),
		nats.PingInterval(time.Duration(config.PingInterval) * time.Second),
		nats.MaxPingsOutstanding(config.MaxPingsOutstanding),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Errorf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infof("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warnf("NATS connection closed")
		}),
	}

	// If token is provided, use token authentication
	if config.Token != "" {
		opts = append(opts, nats.Token(config.Token))
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	client := &Client{
		nc:     nc,
		config: config,
		nodeID: nodeID,
		subs:   make(map[string]*nats.Subscription),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Enable JetStream
	if config.JetStreamEnabled {
		js, err := nc.JetStream()
		if err != nil {
			nc.Close()
			cancel()
			return nil, fmt.Errorf("failed to enable JetStream: %w", err)
		}
		client.js = js
	}

	logger.Infof("NATS client connected successfully")
	return client, nil
}

// Disconnect disconnects from the NATS server
func (c *Client) Disconnect() {
	c.cancel()
	c.subsLock.Lock()
	for subject, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Errorf("Failed to unsubscribe from %s: %v", subject, err)
		}
		delete(c.subs, subject)
	}
	c.subsLock.Unlock()

	if c.nc != nil {
		c.nc.Close()
	}
}

// Subscribe subscribes to a subject
func (c *Client) Subscribe(subject string, callback func(Message)) (*nats.Subscription, error) {
	handler := func(msg *nats.Msg) {
		var message Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			c.logger.Errorf("Failed to unmarshal message from subject %s: %v", msg.Subject, err)
			return
		}

		c.logger.Debugf("Received message from subject %s: %+v", msg.Subject, message.Header)
		callback(message)
	}

	sub, err := c.nc.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	c.subsLock.Lock()
	c.subs[subject] = sub
	c.subsLock.Unlock()

	c.logger.Infof("Subscribed to subject: %s", subject)
	return sub, nil
}

// SubscribeQueue subscribes to a subject with a queue group
func (c *Client) SubscribeQueue(subject, queue string, callback func(Message)) (*nats.Subscription, error) {
	handler := func(msg *nats.Msg) {
		var message Message
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			c.logger.Errorf("Failed to unmarshal message from subject %s: %v", msg.Subject, err)
			return
		}

		c.logger.Debugf("Received message from subject %s (queue: %s): %+v", msg.Subject, queue, message.Header)
		callback(message)
	}

	sub, err := c.nc.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to queue subscribe to subject %s: %w", subject, err)
	}

	c.subsLock.Lock()
	c.subs[subject] = sub
	c.subsLock.Unlock()

	c.logger.Infof("Queue subscribed to subject: %s (queue: %s)", subject, queue)
	return sub, nil
}

// Unsubscribe unsubscribes from a subject
func (c *Client) Unsubscribe(subject string) error {
	c.subsLock.RLock()
	sub, exists := c.subs[subject]
	c.subsLock.RUnlock()

	if !exists {
		return fmt.Errorf("no subscription found for subject: %s", subject)
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("failed to unsubscribe from %s: %w", subject, err)
	}

	c.subsLock.Lock()
	delete(c.subs, subject)
	c.subsLock.Unlock()

	c.logger.Infof("Unsubscribed from subject: %s", subject)
	return nil
}

// Publish publishes a message to a subject
func (c *Client) Publish(subject string, message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := c.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to subject %s: %w", subject, err)
	}

	c.logger.Debugf("Published message to subject %s: %+v", subject, message.Header)
	return nil
}

// PublishMsg publishes a NATS message
func (c *Client) PublishMsg(msg *nats.Msg) error {
	if err := c.nc.PublishMsg(msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Request sends a request and waits for a response
func (c *Client) Request(subject string, message Message, timeout time.Duration) (*Message, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := c.nc.Request(subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to request subject %s: %w", subject, err)
	}

	var response Message
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.Debugf("Received response from subject %s: %+v", subject, response.Header)
	return &response, nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.nc.IsConnected()
}

// GetNodeID returns the node ID
func (c *Client) GetNodeID() string {
	return c.nodeID
}
