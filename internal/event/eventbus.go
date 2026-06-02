package event

import (
	"sync"
)

// Event is a string identifier for event types.
type Event string

const (
	ISOAdded      Event = "iso:added"
	ISORemoved    Event = "iso:removed"
	ISOChanged    Event = "iso:changed"
	ConfigReloaded Event = "config:reloaded"
	LogEntry      Event = "log:entry"
)

// Subscriber is a channel that receives event payloads.
type Subscriber chan any

// EventBus is an in-process publish/subscribe bus.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[Event][]Subscriber
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[Event][]Subscriber),
	}
}

// Subscribe registers a channel for a specific event type.
// Returns the subscriber channel. Callers should read from this channel.
func (eb *EventBus) Subscribe(event Event) Subscriber {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(Subscriber, 100)
	eb.subscribers[event] = append(eb.subscribers[event], ch)
	return ch
}

// Unsubscribe removes a subscriber channel for a specific event.
func (eb *EventBus) Unsubscribe(event Event, sub Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subs := eb.subscribers[event]
	for i, s := range subs {
		if s == sub {
			eb.subscribers[event] = append(subs[:i], subs[i+1:]...)
			close(s)
			return
		}
	}
}

// Publish sends data to all subscribers of the given event.
// This is non-blocking: if a subscriber's buffer is full, the message is dropped.
func (eb *EventBus) Publish(event Event, data any) {
	eb.mu.RLock()
	subs := make([]Subscriber, len(eb.subscribers[event]))
	copy(subs, eb.subscribers[event])
	eb.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- data:
		default:
			// Drop if subscriber buffer is full
		}
	}
}

// Close shuts down the event bus, closing all subscriber channels.
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, subs := range eb.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	eb.subscribers = make(map[Event][]Subscriber)
}
