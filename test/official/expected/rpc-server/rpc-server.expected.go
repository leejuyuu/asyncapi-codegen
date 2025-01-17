// Package "rpcServer" provides primitives to interact with the AsyncAPI specification.
//
// Code generated by github.com/lerenn/asyncapi-codegen version (devel) DO NOT EDIT.
package rpcServer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// AppSubscriber represents all handlers that are expecting messages for App
type AppSubscriber interface {
	// RpcQueue
	RpcQueue(msg RpcQueueMessage, done bool)
}

// AppController is the structure that provides publishing capabilities to the
// developer and and connect the broker with the App
type AppController struct {
	brokerController BrokerController
	stopSubscribers  map[string]chan interface{}
	logger           Logger
}

// NewAppController links the App to the broker
func NewAppController(bs BrokerController) (*AppController, error) {
	if bs == nil {
		return nil, ErrNilBrokerController
	}

	return &AppController{
		brokerController: bs,
		stopSubscribers:  make(map[string]chan interface{}),
	}, nil
}

// AttachLogger attaches a logger that will log operations on controller
func (c *AppController) AttachLogger(logger Logger) {
	c.logger = logger
	c.brokerController.AttachLogger(logger)
}

// logError logs error if the logger has been set
func (c AppController) logError(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "App")
		c.logger.Error(msg, keyvals...)
	}
}

// logInfo logs information if the logger has been set
func (c AppController) logInfo(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "App")
		c.logger.Info(msg, keyvals...)
	}
}

// Close will clean up any existing resources on the controller
func (c *AppController) Close() {
	// Unsubscribing remaining channels
	c.logInfo("Closing App controller")
	c.UnsubscribeAll()
}

// SubscribeAll will subscribe to channels without parameters on which the app is expecting messages.
// For channels with parameters, they should be subscribed independently.
func (c *AppController) SubscribeAll(as AppSubscriber) error {
	if as == nil {
		return ErrNilAppSubscriber
	}

	if err := c.SubscribeRpcQueue(as.RpcQueue); err != nil {
		return err
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *AppController) UnsubscribeAll() {
	// Unsubscribe channels with no parameters (if any)
	c.UnsubscribeRpcQueue()

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribeRpcQueue will subscribe to new messages from 'rpc_queue' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *AppController) SubscribeRpcQueue(fn func(msg RpcQueueMessage, done bool)) error {
	// Get channel path
	path := "rpc_queue"

	// Check if there is already a subscription
	_, exists := c.stopSubscribers[path]
	if exists {
		err := fmt.Errorf("%w: %q channel is already subscribed", ErrAlreadySubscribedChannel, path)
		c.logError(err.Error(), "channel", path)
		return err
	}

	// Subscribe to broker channel
	c.logInfo("Subscribing to channel", "channel", path, "operation", "subscribe")
	msgs, stop, err := c.brokerController.Subscribe(path)
	if err != nil {
		c.logError(err.Error(), "channel", path, "operation", "subscribe")
		return err
	}

	// Asynchronously listen to new messages and pass them to app subscriber
	go func() {
		for {
			// Wait for next message
			um, open := <-msgs

			// Process message
			msg, err := newRpcQueueMessageFromUniversalMessage(um)
			if err != nil {
				c.logError(err.Error(), "channel", path, "operation", "subscribe", "message", msg)
			}

			// Send info if message is correct or susbcription is closed
			if err == nil || !open {
				c.logInfo("Received new message", "channel", path, "operation", "subscribe", "message", msg)
				fn(msg, !open)
			}

			// If subscription is closed, then exit the function
			if !open {
				return
			}
		}
	}()

	// Add the stop channel to the inside map
	c.stopSubscribers[path] = stop

	return nil
}

// UnsubscribeRpcQueue will unsubscribe messages from 'rpc_queue' channel
func (c *AppController) UnsubscribeRpcQueue() {
	// Get channel path
	path := "rpc_queue"

	// Get stop channel
	stopChan, exists := c.stopSubscribers[path]
	if !exists {
		return
	}

	// Stop the channel and remove the entry
	c.logInfo("Unsubscribing from channel", "channel", path, "operation", "unsubscribe")
	stopChan <- true
	delete(c.stopSubscribers, path)
}

// PublishQueue will publish messages to '{queue}' channel
func (c *AppController) PublishQueue(params QueueParameters, msg QueueMessage) error {
	// Convert to UniversalMessage
	um, err := msg.toUniversalMessage()
	if err != nil {
		return err
	}

	// Get channel path
	path := fmt.Sprintf("%v", params.Queue)

	// Publish on event broker
	c.logInfo("Publishing to channel", "channel", path, "operation", "publish", "message", msg)
	return c.brokerController.Publish(path, um)
}

// ClientSubscriber represents all handlers that are expecting messages for Client
type ClientSubscriber interface {
	// Queue
	Queue(msg QueueMessage, done bool)
}

// ClientController is the structure that provides publishing capabilities to the
// developer and and connect the broker with the Client
type ClientController struct {
	brokerController BrokerController
	stopSubscribers  map[string]chan interface{}
	logger           Logger
}

// NewClientController links the Client to the broker
func NewClientController(bs BrokerController) (*ClientController, error) {
	if bs == nil {
		return nil, ErrNilBrokerController
	}

	return &ClientController{
		brokerController: bs,
		stopSubscribers:  make(map[string]chan interface{}),
	}, nil
}

// AttachLogger attaches a logger that will log operations on controller
func (c *ClientController) AttachLogger(logger Logger) {
	c.logger = logger
	c.brokerController.AttachLogger(logger)
}

// logError logs error if the logger has been set
func (c ClientController) logError(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "Client")
		c.logger.Error(msg, keyvals...)
	}
}

// logInfo logs information if the logger has been set
func (c ClientController) logInfo(msg string, keyvals ...interface{}) {
	if c.logger != nil {
		keyvals = append(keyvals, "module", "asyncapi", "controller", "Client")
		c.logger.Info(msg, keyvals...)
	}
}

// Close will clean up any existing resources on the controller
func (c *ClientController) Close() {
	// Unsubscribing remaining channels
	c.logInfo("Closing Client controller")
	c.UnsubscribeAll()
}

// SubscribeAll will subscribe to channels without parameters on which the app is expecting messages.
// For channels with parameters, they should be subscribed independently.
func (c *ClientController) SubscribeAll(as ClientSubscriber) error {
	if as == nil {
		return ErrNilClientSubscriber
	}

	return nil
}

// UnsubscribeAll will unsubscribe all remaining subscribed channels
func (c *ClientController) UnsubscribeAll() {
	// Unsubscribe channels with no parameters (if any)

	// Unsubscribe remaining channels
	for n, stopChan := range c.stopSubscribers {
		stopChan <- true
		delete(c.stopSubscribers, n)
	}
}

// SubscribeQueue will subscribe to new messages from '{queue}' channel.
//
// Callback function 'fn' will be called each time a new message is received.
// The 'done' argument indicates when the subscription is canceled and can be
// used to clean up resources.
func (c *ClientController) SubscribeQueue(params QueueParameters, fn func(msg QueueMessage, done bool)) error {
	// Get channel path
	path := fmt.Sprintf("%v", params.Queue)

	// Check if there is already a subscription
	_, exists := c.stopSubscribers[path]
	if exists {
		err := fmt.Errorf("%w: %q channel is already subscribed", ErrAlreadySubscribedChannel, path)
		c.logError(err.Error(), "channel", path)
		return err
	}

	// Subscribe to broker channel
	c.logInfo("Subscribing to channel", "channel", path, "operation", "subscribe")
	msgs, stop, err := c.brokerController.Subscribe(path)
	if err != nil {
		c.logError(err.Error(), "channel", path, "operation", "subscribe")
		return err
	}

	// Asynchronously listen to new messages and pass them to app subscriber
	go func() {
		for {
			// Wait for next message
			um, open := <-msgs

			// Process message
			msg, err := newQueueMessageFromUniversalMessage(um)
			if err != nil {
				c.logError(err.Error(), "channel", path, "operation", "subscribe", "message", msg)
			}

			// Send info if message is correct or susbcription is closed
			if err == nil || !open {
				c.logInfo("Received new message", "channel", path, "operation", "subscribe", "message", msg)
				fn(msg, !open)
			}

			// If subscription is closed, then exit the function
			if !open {
				return
			}
		}
	}()

	// Add the stop channel to the inside map
	c.stopSubscribers[path] = stop

	return nil
}

// UnsubscribeQueue will unsubscribe messages from '{queue}' channel
func (c *ClientController) UnsubscribeQueue(params QueueParameters) {
	// Get channel path
	path := fmt.Sprintf("%v", params.Queue)

	// Get stop channel
	stopChan, exists := c.stopSubscribers[path]
	if !exists {
		return
	}

	// Stop the channel and remove the entry
	c.logInfo("Unsubscribing from channel", "channel", path, "operation", "unsubscribe")
	stopChan <- true
	delete(c.stopSubscribers, path)
}

// PublishRpcQueue will publish messages to 'rpc_queue' channel
func (c *ClientController) PublishRpcQueue(msg RpcQueueMessage) error {
	// Convert to UniversalMessage
	um, err := msg.toUniversalMessage()
	if err != nil {
		return err
	}

	// Get channel path
	path := "rpc_queue"

	// Publish on event broker
	c.logInfo("Publishing to channel", "channel", path, "operation", "publish", "message", msg)
	return c.brokerController.Publish(path, um)
}

// WaitForQueue will wait for a specific message by its correlation ID
//
// The pub function is the publication function that should be used to send the message
// It will be called after subscribing to the channel to avoid race condition, and potentially loose the message
func (cc *ClientController) WaitForQueue(ctx context.Context, params QueueParameters, publishMsg MessageWithCorrelationID, pub func() error) (QueueMessage, error) {
	// Get channel path
	path := fmt.Sprintf("%v", params.Queue)

	// Subscribe to broker channel
	cc.logInfo("Wait for response", "channel", path, "operation", "wait-for", "correlation-id", publishMsg.CorrelationID())
	msgs, stop, err := cc.brokerController.Subscribe(path)
	if err != nil {
		cc.logError(err.Error(), "channel", path, "operation", "wait-for")
		return QueueMessage{}, err
	}

	// Close subscriber on leave
	defer func() { stop <- true }()

	// Execute publication
	cc.logInfo("Sending request", "channel", path, "operation", "wait-for", "message", publishMsg, "correlation-id", publishMsg.CorrelationID())
	if err := pub(); err != nil {
		return QueueMessage{}, err
	}

	// Wait for corresponding response
	for {
		select {
		case um, open := <-msgs:
			// Get new message
			msg, err := newQueueMessageFromUniversalMessage(um)
			if err != nil {
				cc.logError(err.Error(), "channel", path, "operation", "wait-for")
			}

			// If valid message with corresponding correlation ID, return message
			if err == nil && publishMsg.CorrelationID() == msg.CorrelationID() {
				cc.logInfo("Received expected message", "channel", path, "operation", "wait-for", "message", msg, "correlation-id", msg.CorrelationID())
				return msg, nil
			} else if !open { // If message is invalid or not corresponding and the subscription is closed, then return error
				cc.logError("Channel closed before getting message", "channel", path, "operation", "wait-for", "correlation-id", publishMsg.CorrelationID())
				return QueueMessage{}, ErrSubscriptionCanceled
			}
		case <-ctx.Done(): // Return error if context is done
			cc.logError("Context done before getting message", "channel", path, "operation", "wait-for", "correlation-id", publishMsg.CorrelationID())
			return QueueMessage{}, ErrContextCanceled
		}
	}
}

const (
	// CorrelationIDField is the name of the field that will contain the correlation ID
	CorrelationIDField = "correlation_id"
)

// UniversalMessage is a wrapper that will contain all information regarding a message
type UniversalMessage struct {
	CorrelationID *string
	Payload       []byte
}

// BrokerController represents the functions that should be implemented to connect
// the broker to the application or the client
type BrokerController interface {
	// AttachLogger attaches a logger that will log operations on broker controller
	AttachLogger(logger Logger)

	// Publish a message to the broker
	Publish(channel string, mw UniversalMessage) error

	// Subscribe to messages from the broker
	Subscribe(channel string) (msgs chan UniversalMessage, stop chan interface{}, err error)
}

var (
	// Generic error for AsyncAPI generated code
	ErrAsyncAPI = errors.New("error when using AsyncAPI")

	// ErrContextCanceled is given when a given context is canceled
	ErrContextCanceled = fmt.Errorf("%w: context canceled", ErrAsyncAPI)

	// ErrNilBrokerController is raised when a nil broker controller is user
	ErrNilBrokerController = fmt.Errorf("%w: nil broker controller has been used", ErrAsyncAPI)

	// ErrNilAppSubscriber is raised when a nil app subscriber is user
	ErrNilAppSubscriber = fmt.Errorf("%w: nil app subscriber has been used", ErrAsyncAPI)

	// ErrNilClientSubscriber is raised when a nil client subscriber is user
	ErrNilClientSubscriber = fmt.Errorf("%w: nil client subscriber has been used", ErrAsyncAPI)

	// ErrAlreadySubscribedChannel is raised when a subscription is done twice
	// or more without unsubscribing
	ErrAlreadySubscribedChannel = fmt.Errorf("%w: the channel has already been subscribed", ErrAsyncAPI)

	// ErrSubscriptionCanceled is raised when expecting something and the subscription has been canceled before it happens
	ErrSubscriptionCanceled = fmt.Errorf("%w: the subscription has been canceled", ErrAsyncAPI)
)

type Logger interface {
	// Info logs information based on a message and key-value elements
	Info(msg string, keyvals ...interface{})

	// Error logs error based on a message and key-value elements
	Error(msg string, keyvals ...interface{})
}

type MessageWithCorrelationID interface {
	CorrelationID() string
}

type Error struct {
	Channel string
	Err     error
}

func (e *Error) Error() string {
	return fmt.Sprintf("channel %q: err %v", e.Channel, e.Err)
}

// RpcQueueMessage is the message expected for 'RpcQueue' channel
type RpcQueueMessage struct {
	// Headers will be used to fill the message headers
	Headers struct {
		CorrelationID *string `json:"correlation_id"`
	}

	// Payload will be inserted in the message payload
	Payload struct {
		Numbers []float64 `json:"numbers"`
	}
}

func NewRpcQueueMessage() RpcQueueMessage {
	var msg RpcQueueMessage

	// Set correlation ID
	u := uuid.New().String()
	msg.Headers.CorrelationID = &u

	return msg
}

// newRpcQueueMessageFromUniversalMessage will fill a new RpcQueueMessage with data from UniversalMessage
func newRpcQueueMessageFromUniversalMessage(um UniversalMessage) (RpcQueueMessage, error) {
	var msg RpcQueueMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// Get correlation ID
	msg.Headers.CorrelationID = um.CorrelationID

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from RpcQueueMessage data
func (msg RpcQueueMessage) toUniversalMessage() (UniversalMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return UniversalMessage{}, err
	}

	// Set correlation ID if it does not exist
	var correlationID *string
	if msg.Headers.CorrelationID != nil {
		correlationID = msg.Headers.CorrelationID
	} else {
		u := uuid.New().String()
		correlationID = &u
	}

	return UniversalMessage{
		Payload:       payload,
		CorrelationID: correlationID,
	}, nil
}

// CorrelationID will give the correlation ID of the message, based on AsyncAPI spec
func (msg RpcQueueMessage) CorrelationID() string {
	if msg.Headers.CorrelationID != nil {
		return *msg.Headers.CorrelationID
	}

	return ""
}

// SetAsResponseFrom will correlate the message with the one passed in parameter.
// It will assign the 'req' message correlation ID to the message correlation ID,
// both specified in AsyncAPI spec.
func (msg *RpcQueueMessage) SetAsResponseFrom(req MessageWithCorrelationID) {
	id := req.CorrelationID()
	msg.Headers.CorrelationID = &id
}

// QueueParameters represents Queue channel parameters
type QueueParameters struct {
	Queue string
}

// QueueMessage is the message expected for 'Queue' channel
type QueueMessage struct {
	// Headers will be used to fill the message headers
	Headers struct {
		CorrelationID *string `json:"correlation_id"`
	}

	// Payload will be inserted in the message payload
	Payload struct {
		Result *float64 `json:"result"`
	}
}

func NewQueueMessage() QueueMessage {
	var msg QueueMessage

	// Set correlation ID
	u := uuid.New().String()
	msg.Headers.CorrelationID = &u

	return msg
}

// newQueueMessageFromUniversalMessage will fill a new QueueMessage with data from UniversalMessage
func newQueueMessageFromUniversalMessage(um UniversalMessage) (QueueMessage, error) {
	var msg QueueMessage

	// Unmarshal payload to expected message payload format
	err := json.Unmarshal(um.Payload, &msg.Payload)
	if err != nil {
		return msg, err
	}

	// Get correlation ID
	msg.Headers.CorrelationID = um.CorrelationID

	// TODO: run checks on msg type

	return msg, nil
}

// toUniversalMessage will generate an UniversalMessage from QueueMessage data
func (msg QueueMessage) toUniversalMessage() (UniversalMessage, error) {
	// TODO: implement checks on message

	// Marshal payload to JSON
	payload, err := json.Marshal(msg.Payload)
	if err != nil {
		return UniversalMessage{}, err
	}

	// Set correlation ID if it does not exist
	var correlationID *string
	if msg.Headers.CorrelationID != nil {
		correlationID = msg.Headers.CorrelationID
	} else {
		u := uuid.New().String()
		correlationID = &u
	}

	return UniversalMessage{
		Payload:       payload,
		CorrelationID: correlationID,
	}, nil
}

// CorrelationID will give the correlation ID of the message, based on AsyncAPI spec
func (msg QueueMessage) CorrelationID() string {
	if msg.Headers.CorrelationID != nil {
		return *msg.Headers.CorrelationID
	}

	return ""
}

// SetAsResponseFrom will correlate the message with the one passed in parameter.
// It will assign the 'req' message correlation ID to the message correlation ID,
// both specified in AsyncAPI spec.
func (msg *QueueMessage) SetAsResponseFrom(req MessageWithCorrelationID) {
	id := req.CorrelationID()
	msg.Headers.CorrelationID = &id
}
