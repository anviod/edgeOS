package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client represents an MQTT client
type Client struct {
	client      mqtt.Client
	config      *Config
	nodeID      string
	handlers    map[string]mqtt.MessageHandler
	handlerLock sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      Logger
}

// Config represents MQTT client configuration
type Config struct {
	Broker               string
	ClientID             string
	Username             string
	Password             string
	QoS                  byte
	Retain               bool
	CleanSession         bool
	KeepAlive            int
	ConnectTimeout       int
	AutoReconnect        bool
	MaxReconnectInterval int
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

// NewClient creates a new MQTT client
func NewClient(config *Config, nodeID string, logger Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	opts := mqtt.NewClientOptions()
	opts.AddBroker(config.Broker)
	opts.SetClientID(config.ClientID)
	opts.SetUsername(config.Username)
	opts.SetPassword(config.Password)
	opts.SetAutoReconnect(config.AutoReconnect)
	opts.SetKeepAlive(time.Duration(config.KeepAlive) * time.Second)
	opts.SetCleanSession(config.CleanSession)
	opts.SetConnectRetry(true)

	// Connection lost callback
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.Errorf("MQTT connection lost: %v", err)
	})

	// Connection success callback
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Infof("MQTT connected successfully")
	})

	return &Client{
		client:   mqtt.NewClient(opts),
		config:   config,
		nodeID:   nodeID,
		handlers: make(map[string]mqtt.MessageHandler),
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
	}
}

// Connect connects to the MQTT broker
func (c *Client) Connect() error {
	token := c.client.Connect()
	if token.WaitTimeout(time.Duration(c.config.ConnectTimeout) * time.Second) {
		if token.Error() != nil {
			return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
		}
		return nil
	}
	return fmt.Errorf("connection timeout")
}

// Disconnect disconnects from the MQTT broker
func (c *Client) Disconnect() {
	c.cancel()
	if c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}

// Subscribe subscribes to a topic
func (c *Client) Subscribe(topic string, qos byte, callback func(Message)) error {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()

	handler := func(client mqtt.Client, msg mqtt.Message) {
		var message Message
		if err := json.Unmarshal(msg.Payload(), &message); err != nil {
			c.logger.Errorf("Failed to unmarshal message from topic %s: %v", msg.Topic(), err)
			return
		}

		c.logger.Debugf("Received message from topic %s: %+v", msg.Topic(), message.Header)
		callback(message)
	}

	token := c.client.Subscribe(topic, qos, handler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}

	c.handlers[topic] = handler
	c.logger.Infof("Subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe unsubscribes from topics
func (c *Client) Unsubscribe(topics ...string) error {
	token := c.client.Unsubscribe(topics...)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe: %w", token.Error())
	}

	c.handlerLock.Lock()
	for _, topic := range topics {
		delete(c.handlers, topic)
	}
	c.handlerLock.Unlock()

	c.logger.Infof("Unsubscribed from topics: %v", topics)
	return nil
}

// Publish publishes a message to a topic
func (c *Client) Publish(topic string, qos byte, retained bool, message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	token := c.client.Publish(topic, qos, retained, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, token.Error())
	}

	c.logger.Debugf("Published message to topic %s: %+v", topic, message.Header)
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// GetNodeID returns the node ID
func (c *Client) GetNodeID() string {
	return c.nodeID
}
