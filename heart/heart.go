// Iris - Decentralized Messaging Framework
// Copyright 2013 Peter Szilagyi. All rights reserved.
//
// Iris is dual licensed: you can redistribute it and/or modify it under the
// terms of the GNU General Public License as published by the Free Software
// Foundation, either version 3 of the License, or (at your option) any later
// version.
//
// The framework is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for
// more details.
//
// Alternatively, the Iris framework may be used in accordance with the terms
// and conditions contained in a signed written agreement between you and the
// author(s).
//
// Author: peterke@gmail.com (Peter Szilagyi)

// Package heart provides a simple and generic heartbeat mechanism that oversees
// the pinging of some entities and reports various events through callbacks.
package heart

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
)

// Heartbeat callback interface to get notified of events.
type Callback interface {
	Beat()
	Dead(id *big.Int)
}

// Heartbeat mechanism to monitor the liveliness of some entities.
type Heart struct {
	mems entitySlice   // List of entities monitored
	tick int           // Current monitoring cycle tick
	beat time.Duration // Time duration of a beat cycle
	kill int           // Number of missed ticks before and entity is reported dead

	call Callback // Application callback to notify of events

	quit chan struct{}
	lock sync.Mutex
}

// Creates and returns a new heartbeat mechanism beating once every beat,
// reporting entities as dead if not seen in kill beats.
func New(beat time.Duration, kill int, handler Callback) *Heart {
	return &Heart{
		mems: []*entity{},
		beat: beat,
		kill: kill,
		call: handler,
		quit: make(chan struct{}),
	}
}

// Starts the beater and event notifier.
func (h *Heart) Start() {
	go h.beater()
}

// Terminates the heartbeat mechanism.
func (h *Heart) Terminate() {
	close(h.quit)
}

// Registers a new entity for the beater to monitor.
func (h *Heart) Monitor(id *big.Int) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	// Make sure no duplicate entries are specified
	idx := h.mems.Search(id)
	if idx < len(h.mems) && h.mems[idx].id.Cmp(id) == 0 {
		return fmt.Errorf("duplicate entry")
	}

	h.mems = append(h.mems, &entity{id: id, tick: h.tick})
	sort.Sort(h.mems)
	return nil
}

// Unregisters an entity from the possible balancing destinations.
func (h *Heart) Unmonitor(id *big.Int) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	idx := h.mems.Search(id)
	if idx < len(h.mems) && h.mems[idx].id.Cmp(id) == 0 {
		// Swap with last element
		last := len(h.mems) - 1
		h.mems[idx] = h.mems[last]
		h.mems = h.mems[:last]

		// Get back to sorted order
		sort.Sort(h.mems)
		return nil
	}
	return fmt.Errorf("non-monitored entity")
}

// Updates the life tick of an entity.
func (h *Heart) Ping(id *big.Int) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	idx := h.mems.Search(id)
	if idx < len(h.mems) && h.mems[idx].id.Cmp(id) == 0 {
		h.mems[idx].tick = h.tick
		return nil
	}
	return fmt.Errorf("non-monitored entity")
}

// Beater function meant to run as a separate go routine to keep pinging each
// monitored entity and report when some fail to respond within alloted time.
func (h *Heart) beater() {
	beat := time.NewTicker(h.beat)
	defer beat.Stop()

	dead := []*big.Int{}
	for {
		select {
		case <-h.quit:
			return
		case <-beat.C:
			// Beat cycle: update tick and collect dead entries
			h.lock.Lock()
			h.tick++
			dead = dead[:0]
			for _, m := range h.mems {
				if h.tick-m.tick >= h.kill {
					dead = append(dead, m.id)
				}
			}
			h.lock.Unlock()

			// Signal beat and dead entities after releasing the lock
			h.call.Beat()
			for _, id := range dead {
				h.call.Dead(id)
			}
		}
	}
}
