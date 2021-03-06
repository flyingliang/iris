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

// File containing the inter-node communication methods. For every connection
// two separate go routines are started: a receiver that accepts inbound packets
// executing the routing on the same thread and a sender which moves messages
// from the application channel to the network socket. Network errors are
// detected by the receiver, which notifies the overlay.

package overlay

import (
	"encoding/gob"
	"github.com/karalabe/iris/proto"
	"math/big"
)

// Overlay connection operation code type.
type opcode uint8

// Overlay connection operation types.
const (
	opNop   opcode = iota // No-operation, placeholder for refactor
	opClose               // Signal peer connection termination
)

// Routing state exchange message (leaves, neighbors and common row).
type state struct {
	Addrs   map[string][]string
	Updated uint64
	Repair  bool
	Passive bool
}

// Extra headers for the overlay.
type header struct {
	Meta  interface{} // Additional upper layer headers
	Op    opcode      // The operation to execute
	Dest  *big.Int    // Destination id
	State *state      // Routing table state exchange
}

// Make sure the header struct is registered with gob.
func init() {
	gob.Register(&header{})
}

// Simple wrapper around the peer send method, to handle errors by dropping.
func (o *Overlay) send(msg *proto.Message, p *peer) {
	if err := p.send(msg); err != nil {
		go func() { o.dropSink <- p }()
	}
}

// Simple utility function to wrap the contents of a system message into the
// wire format.
func (o *Overlay) sendWrap(s *state, dest *big.Int, p *peer) {
	msg := &proto.Message{
		Head: proto.Header{
			Meta: &header{
				Dest:  dest,
				State: s,
			},
		},
	}
	o.send(msg, p)
}

// Sends an overlay join message to the remote peer, which is a simple state
// package having 0 as the update time and containing only the local addresses.
func (o *Overlay) sendJoin(p *peer) {
	s := new(state)
	s.Addrs = make(map[string][]string)

	// Ensure nodes can contact joining peer
	o.lock.RLock()
	s.Addrs[o.nodeId.String()] = o.addrs
	o.lock.RUnlock()

	o.sendWrap(s, o.nodeId, p)
}

// Sends an overlay state message to the remote peer and optionally may request a
// state update in response (route repair).
func (o *Overlay) sendState(p *peer, repair bool) {
	s := new(state)
	s.Addrs = make(map[string][]string)
	s.Repair = repair

	o.lock.RLock()
	s.Updated = o.time

	// Serialize the leaf set, common row and neighbor list into the address map.
	// Make sure all entries are checked for existence to avoid a race condition
	// with node dropping vs. table updates.
	s.Addrs[o.nodeId.String()] = o.addrs
	for _, id := range o.routes.leaves {
		if id.Cmp(o.nodeId) != 0 {
			sid := id.String()
			if node, ok := o.pool[sid]; ok {
				s.Addrs[sid] = node.addrs
			}
		}
	}
	idx, _ := prefix(o.nodeId, p.nodeId)
	for _, id := range o.routes.routes[idx] {
		if id != nil {
			sid := id.String()
			if node, ok := o.pool[sid]; ok {
				s.Addrs[sid] = node.addrs
			}
		}
	}
	o.lock.RUnlock()

	o.sendWrap(s, o.nodeId, p)
}

// Sends a heartbeat message, tagging whether the connection is an active route
// entry or not.
func (o *Overlay) sendBeat(p *peer, passive bool) {
	s := new(state)
	s.Passive = passive

	o.lock.RLock()
	s.Updated = o.time
	o.lock.RUnlock()

	o.sendWrap(s, p.nodeId, p)
}
