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
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/sqmgr/sqmgr-api/pkg/model"
)

const (
	pgChannelSportsEventUpdated = "sports_event_updated"
	pgMinReconnect              = 10 * time.Second
	pgMaxReconnect              = 60 * time.Second
	pgPingInterval              = 90 * time.Second
)

// PGListener listens for PostgreSQL NOTIFY events and publishes them to the pool broker.
type PGListener struct {
	listener *pq.Listener
	model    *model.Model
	broker   *PoolBroker
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewPGListener creates a new PGListener that listens on the sports_event_updated channel.
func NewPGListener(dsn string, m *model.Model, broker *PoolBroker) (*PGListener, error) {
	listener := pq.NewListener(dsn, pgMinReconnect, pgMaxReconnect, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			logrus.WithError(err).Error("pg listener connection event")
		}
	})

	if err := listener.Listen(pgChannelSportsEventUpdated); err != nil {
		listener.Close()
		return nil, err
	}

	return &PGListener{
		listener: listener,
		model:    m,
		broker:   broker,
	}, nil
}

// Start launches the background goroutine that processes notifications.
func (l *PGListener) Start(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)
	l.wg.Add(1)
	go l.run(ctx)
}

func (l *PGListener) run(ctx context.Context) {
	defer l.wg.Done()

	pingTicker := time.NewTicker(pgPingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case n := <-l.listener.Notify:
			if n == nil {
				// Connection lost and re-established; the listener re-subscribes automatically
				continue
			}
			l.handleNotification(ctx, n)
		case <-pingTicker.C:
			if err := l.listener.Ping(); err != nil {
				logrus.WithError(err).Error("pg listener ping failed")
			}
		}
	}
}

func (l *PGListener) handleNotification(ctx context.Context, n *pq.Notification) {
	eventID, err := strconv.ParseInt(n.Extra, 10, 64)
	if err != nil {
		logrus.WithError(err).WithField("payload", n.Extra).Error("pg listener: invalid event ID in notification payload")
		return
	}

	tokens, err := l.model.PoolTokensByEventID(ctx, eventID)
	if err != nil {
		logrus.WithError(err).WithField("eventID", eventID).Error("pg listener: failed to look up pool tokens")
		return
	}

	for _, token := range tokens {
		l.broker.Publish(token, PoolEvent{Type: EventGridUpdated})
	}

	if len(tokens) > 0 {
		logrus.WithFields(logrus.Fields{
			"eventID": eventID,
			"pools":   len(tokens),
		}).Info("pg listener: published grid_updated events")
	}
}

// Close stops the listener and waits for the background goroutine to finish.
func (l *PGListener) Close() error {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
	return l.listener.Close()
}
