/*
Copyright (C) 2019 Tom Peters

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package server

import (
	"sync"
)

// PoolEventType represents the type of pool event
type PoolEventType string

// Pool event type constants
const (
	EventSquareUpdated PoolEventType = "square_updated"
	EventGridUpdated   PoolEventType = "grid_updated"
	EventPoolUpdated   PoolEventType = "pool_updated"
)

// PoolEvent represents an event that occurred in a pool
type PoolEvent struct {
	Type PoolEventType `json:"type"`
}

// PoolBroker manages per-pool SSE subscriptions
type PoolBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan PoolEvent]struct{}
}

// NewPoolBroker creates a new broker for managing pool event subscriptions
func NewPoolBroker() *PoolBroker {
	return &PoolBroker{
		subscribers: make(map[string]map[chan PoolEvent]struct{}),
	}
}

// Subscribe registers a new subscriber for a pool and returns a channel to receive events
func (b *PoolBroker) Subscribe(poolToken string) chan PoolEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan PoolEvent, 16)
	if b.subscribers[poolToken] == nil {
		b.subscribers[poolToken] = make(map[chan PoolEvent]struct{})
	}
	b.subscribers[poolToken][ch] = struct{}{}
	return ch
}

// Unsubscribe removes a subscriber from a pool and closes the channel
func (b *PoolBroker) Unsubscribe(poolToken string, ch chan PoolEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[poolToken]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(b.subscribers, poolToken)
		}
	}
}

// Publish sends an event to all subscribers of a pool (non-blocking)
func (b *PoolBroker) Publish(poolToken string, event PoolEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers[poolToken] {
		select {
		case ch <- event:
		default:
			// Drop event if subscriber's channel is full
		}
	}
}

// SubscriberCount returns the number of active subscribers for a pool
func (b *PoolBroker) SubscriberCount(poolToken string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[poolToken])
}
