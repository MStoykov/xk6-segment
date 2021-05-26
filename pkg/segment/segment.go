/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2021 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package segment

import (
	"context"
	"errors"
	"sync"

	"go.k6.io/k6/lib"
)

// SegmentedIndex ...
type SegmentedIndex struct {
	start, lcd       int64
	offsets          []int64
	mx               sync.RWMutex
	scaled, unscaled int64 // for both the first element(vu) is 1 not 0
}

type Module struct {
	shared sharedSegmentedIndexes
}

type sharedSegmentedIndexes struct {
	data map[string]*SegmentedIndex
	mu   sync.RWMutex
}

func (s *sharedSegmentedIndexes) get(state *lib.State, name string) *SegmentedIndex {
	s.mu.RLock()
	array, ok := s.data[name]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		array, ok = s.data[name]
		if !ok {
			// cache those
			tuple, err := lib.NewExecutionTuple(state.Options.ExecutionSegment, state.Options.ExecutionSegmentSequence)
			if err != nil {
				panic(err)
			}
			start, offsets, lcd := tuple.GetStripedOffsets()

			array = NewSegmentedIndex(start, lcd, offsets)
			s.data[name] = array
		}
	}

	return array
}

func New() *Module {
	return &Module{
		shared: sharedSegmentedIndexes{
			data: make(map[string]*SegmentedIndex),
		},
	}
}

func (m *Module) XSegmentedIndex(ctx context.Context) *SegmentedIndex {
	state := lib.GetState(ctx)
	// TODO check state ;)

	// cache those
	tuple, err := lib.NewExecutionTuple(state.Options.ExecutionSegment, state.Options.ExecutionSegmentSequence)
	if err != nil {
		panic(err)
	}
	start, offsets, lcd := tuple.GetStripedOffsets()

	return NewSegmentedIndex(start, lcd, offsets)
}

func (m *Module) XSharedSegmentedIndex(ctx context.Context, name string) *SegmentedIndex {
	state := lib.GetState(ctx)
	// TODO check state ;)

	if len(name) == 0 {
		panic(errors.New("empty name provided to SharedArray's constructor"))
	}

	return m.shared.get(state, name)
}

// NewSegmentedIndex returns a pointer to a new SegmentedIndex instance,
// given a starting index, LCD and offsets as returned by GetStripedOffsets().
func NewSegmentedIndex(start, lcd int64, offsets []int64) *SegmentedIndex {
	return &SegmentedIndex{start: start, lcd: lcd, offsets: offsets}
}

// Next goes to the next scaled index and moves the unscaled one accordingly.
func (s *SegmentedIndex) Next() SegmentedIndexResult {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.scaled == 0 { // the 1 element(VU) is at the start
		s.unscaled += s.start + 1 // the first element of the start 0, but the here we need it to be 1 so we add 1
	} else { // if we are not at the first element we need to go through the offsets, looping over them
		s.unscaled += s.offsets[int(s.scaled-1)%len(s.offsets)] // slice's index start at 0 ours start at 1
	}
	s.scaled++
	return SegmentedIndexResult{Scaled: s.scaled, Unscaled: s.unscaled}
}

// Prev goes to the previous scaled value and sets the unscaled one accordingly.
// Calling Prev when s.scaled == 0 is undefined.
func (s *SegmentedIndex) Prev() SegmentedIndexResult {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.scaled == 1 { // we are the first need to go to the 0th element which means we need to remove the start
		s.unscaled -= s.start + 1 // this could've been just settign to 0
	} else { // not at the first element - need to get the previously added offset so
		s.unscaled -= s.offsets[int(s.scaled-2)%len(s.offsets)] // slice's index start 0 our start at 1
	}
	s.scaled--
	return SegmentedIndexResult{Scaled: s.scaled, Unscaled: s.unscaled}
}

type SegmentedIndexResult struct {
	Scaled, Unscaled int64
}

// GoTo sets the scaled index to its biggest value for which the corresponding
// unscaled index is is smaller or equal to value.
func (s *SegmentedIndex) GoTo(value int64) SegmentedIndexResult { // TODO optimize
	s.mx.Lock()
	defer s.mx.Unlock()
	var gi int64
	// Because of the cyclical nature of the striping algorithm (with a cycle
	// length of LCD, the least common denominator), when scaling large values
	// (i.e. many multiples of the LCD), we can quickly calculate how many times
	// the cycle repeats.
	wholeCycles := (value / s.lcd)
	// So we can set some approximate initial values quickly, since we also know
	// precisely how many scaled values there are per cycle length.
	s.scaled = wholeCycles * int64(len(s.offsets))
	s.unscaled = wholeCycles*s.lcd + s.start + 1 // our indexes are from 1 the start is from 0
	// Approach the final value using the slow algorithm with the step by step loop
	// TODO: this can be optimized by another array with size offsets that instead of the offsets
	// from the previous is the offset from either 0 or start
	i := s.start
	for ; i < value%s.lcd; gi, i = gi+1, i+s.offsets[gi] {
		s.scaled++
		s.unscaled += s.offsets[gi]
	}

	if gi > 0 { // there were more values after the wholecycles
		// the last offset actually shouldn't have been added
		s.unscaled -= s.offsets[gi-1]
	} else if s.scaled > 0 { // we didn't actually have more values after the wholecycles but we still had some
		// in this case the unscaled value needs to move back by the last offset as it would've been
		// the one to get it from the value it needs to be to it's current one
		s.unscaled -= s.offsets[len(s.offsets)-1]
	}

	if s.scaled == 0 {
		s.unscaled = 0 // we would've added the start and 1
	}

	return SegmentedIndexResult{Scaled: s.scaled, Unscaled: s.unscaled}
}
