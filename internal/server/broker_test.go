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
	"testing"
	"time"
)

func TestPoolBroker_SubscribeAndPublish(t *testing.T) {
	broker := NewPoolBroker()

	ch := broker.Subscribe("pool-abc")
	defer broker.Unsubscribe("pool-abc", ch)

	broker.Publish("pool-abc", PoolEvent{Type: EventSquareUpdated})

	select {
	case event := <-ch:
		if event.Type != EventSquareUpdated {
			t.Errorf("expected event type %s, got %s", EventSquareUpdated, event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPoolBroker_MultipleSubscribers(t *testing.T) {
	broker := NewPoolBroker()

	ch1 := broker.Subscribe("pool-abc")
	ch2 := broker.Subscribe("pool-abc")
	defer broker.Unsubscribe("pool-abc", ch1)
	defer broker.Unsubscribe("pool-abc", ch2)

	if broker.SubscriberCount("pool-abc") != 2 {
		t.Errorf("expected 2 subscribers, got %d", broker.SubscriberCount("pool-abc"))
	}

	broker.Publish("pool-abc", PoolEvent{Type: EventGridUpdated})

	for i, ch := range []chan PoolEvent{ch1, ch2} {
		select {
		case event := <-ch:
			if event.Type != EventGridUpdated {
				t.Errorf("subscriber %d: expected event type %s, got %s", i, EventGridUpdated, event.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestPoolBroker_IsolatedPools(t *testing.T) {
	broker := NewPoolBroker()

	ch1 := broker.Subscribe("pool-abc")
	ch2 := broker.Subscribe("pool-xyz")
	defer broker.Unsubscribe("pool-abc", ch1)
	defer broker.Unsubscribe("pool-xyz", ch2)

	broker.Publish("pool-abc", PoolEvent{Type: EventSquareUpdated})

	select {
	case <-ch2:
		t.Fatal("pool-xyz subscriber should not receive events for pool-abc")
	case <-time.After(50 * time.Millisecond):
		// Expected: no event received
	}

	select {
	case event := <-ch1:
		if event.Type != EventSquareUpdated {
			t.Errorf("expected event type %s, got %s", EventSquareUpdated, event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPoolBroker_Unsubscribe(t *testing.T) {
	broker := NewPoolBroker()

	ch := broker.Subscribe("pool-abc")

	if broker.SubscriberCount("pool-abc") != 1 {
		t.Errorf("expected 1 subscriber, got %d", broker.SubscriberCount("pool-abc"))
	}

	broker.Unsubscribe("pool-abc", ch)

	if broker.SubscriberCount("pool-abc") != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", broker.SubscriberCount("pool-abc"))
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

func TestPoolBroker_UnsubscribeLastCleansUp(t *testing.T) {
	broker := NewPoolBroker()

	ch := broker.Subscribe("pool-abc")
	broker.Unsubscribe("pool-abc", ch)

	broker.mu.RLock()
	_, exists := broker.subscribers["pool-abc"]
	broker.mu.RUnlock()

	if exists {
		t.Error("expected pool entry to be removed when last subscriber leaves")
	}
}

func TestPoolBroker_NonBlockingPublish(t *testing.T) {
	broker := NewPoolBroker()

	ch := broker.Subscribe("pool-abc")
	defer broker.Unsubscribe("pool-abc", ch)

	// Fill the channel buffer (size 16)
	for i := 0; i < 16; i++ {
		broker.Publish("pool-abc", PoolEvent{Type: EventSquareUpdated})
	}

	// This should not block even though the channel is full
	done := make(chan struct{})
	go func() {
		broker.Publish("pool-abc", PoolEvent{Type: EventGridUpdated})
		close(done)
	}()

	select {
	case <-done:
		// Success: publish didn't block
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full channel")
	}
}

func TestPoolBroker_PublishToNonexistentPool(t *testing.T) {
	broker := NewPoolBroker()
	// Should not panic
	broker.Publish("nonexistent", PoolEvent{Type: EventSquareUpdated})
}

func TestPoolBroker_ConcurrentAccess(t *testing.T) {
	broker := NewPoolBroker()
	var wg sync.WaitGroup

	// Concurrent subscribes
	channels := make([]chan PoolEvent, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			channels[idx] = broker.Subscribe("pool-abc")
		}(i)
	}
	wg.Wait()

	// Concurrent publishes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			broker.Publish("pool-abc", PoolEvent{Type: EventSquareUpdated})
		}()
	}
	wg.Wait()

	// Concurrent unsubscribes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			broker.Unsubscribe("pool-abc", channels[idx])
		}(i)
	}
	wg.Wait()

	if broker.SubscriberCount("pool-abc") != 0 {
		t.Errorf("expected 0 subscribers, got %d", broker.SubscriberCount("pool-abc"))
	}
}
