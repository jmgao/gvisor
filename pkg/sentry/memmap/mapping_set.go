// Copyright 2018 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memmap

import (
	"fmt"
	"math"

	"gvisor.dev/gvisor/pkg/sentry/usermem"
)

// MappingSet maps offsets into a Mappable to mappings of those offsets. It is
// used to implement Mappable.AddMapping and RemoveMapping for Mappables that
// may need to call MappingSpace.Invalidate.
//
// type MappingSet <generated by go_generics>

// MappingsOfRange is the value type of MappingSet, and represents the set of
// all mappings of the corresponding MappableRange.
//
// Using a map offers O(1) lookups in RemoveMapping and
// mappingSetFunctions.Merge.
type MappingsOfRange map[MappingOfRange]struct{}

// MappingOfRange represents a mapping of a MappableRange.
//
// +stateify savable
type MappingOfRange struct {
	MappingSpace MappingSpace
	AddrRange    usermem.AddrRange
	Writable     bool
}

func (r MappingOfRange) invalidate(opts InvalidateOpts) {
	r.MappingSpace.Invalidate(r.AddrRange, opts)
}

// String implements fmt.Stringer.String.
func (r MappingOfRange) String() string {
	return fmt.Sprintf("%#v", r.AddrRange)
}

// mappingSetFunctions implements segment.Functions for MappingSet.
type mappingSetFunctions struct{}

// MinKey implements segment.Functions.MinKey.
func (mappingSetFunctions) MinKey() uint64 {
	return 0
}

// MaxKey implements segment.Functions.MaxKey.
func (mappingSetFunctions) MaxKey() uint64 {
	return math.MaxUint64
}

// ClearValue implements segment.Functions.ClearValue.
func (mappingSetFunctions) ClearValue(v *MappingsOfRange) {
	*v = MappingsOfRange{}
}

// Merge implements segment.Functions.Merge.
//
// Since each value is a map of MappingOfRanges, values can only be merged if
// all MappingOfRanges in each map have an exact pair in the other map, forming
// one contiguous region.
func (mappingSetFunctions) Merge(r1 MappableRange, val1 MappingsOfRange, r2 MappableRange, val2 MappingsOfRange) (MappingsOfRange, bool) {
	if len(val1) != len(val2) {
		return nil, false
	}

	merged := make(MappingsOfRange, len(val1))

	// Each MappingOfRange in val1 must have a matching region in val2, forming
	// one contiguous region.
	for k1 := range val1 {
		// We expect val2 to to contain a key that forms a contiguous
		// region with k1.
		k2 := MappingOfRange{
			MappingSpace: k1.MappingSpace,
			AddrRange: usermem.AddrRange{
				Start: k1.AddrRange.End,
				End:   k1.AddrRange.End + usermem.Addr(r2.Length()),
			},
			Writable: k1.Writable,
		}
		if _, ok := val2[k2]; !ok {
			return nil, false
		}

		// OK. Add it to the merged map.
		merged[MappingOfRange{
			MappingSpace: k1.MappingSpace,
			AddrRange: usermem.AddrRange{
				Start: k1.AddrRange.Start,
				End:   k2.AddrRange.End,
			},
			Writable: k1.Writable,
		}] = struct{}{}
	}

	return merged, true
}

// Split implements segment.Functions.Split.
func (mappingSetFunctions) Split(r MappableRange, val MappingsOfRange, split uint64) (MappingsOfRange, MappingsOfRange) {
	if split <= r.Start || split >= r.End {
		panic(fmt.Sprintf("split is not within range %v", r))
	}

	m1 := make(MappingsOfRange, len(val))
	m2 := make(MappingsOfRange, len(val))

	// split is a value in MappableRange, we need the offset into the
	// corresponding MappingsOfRange.
	offset := usermem.Addr(split - r.Start)
	for k := range val {
		k1 := MappingOfRange{
			MappingSpace: k.MappingSpace,
			AddrRange: usermem.AddrRange{
				Start: k.AddrRange.Start,
				End:   k.AddrRange.Start + offset,
			},
			Writable: k.Writable,
		}
		m1[k1] = struct{}{}

		k2 := MappingOfRange{
			MappingSpace: k.MappingSpace,
			AddrRange: usermem.AddrRange{
				Start: k.AddrRange.Start + offset,
				End:   k.AddrRange.End,
			},
			Writable: k.Writable,
		}
		m2[k2] = struct{}{}
	}

	return m1, m2
}

// subsetMapping returns the MappingOfRange that maps subsetRange, given that
// ms maps wholeRange beginning at addr.
//
// For instance, suppose wholeRange = [0x0, 0x2000) and addr = 0x4000,
// indicating that ms maps addresses [0x4000, 0x6000) to MappableRange [0x0,
// 0x2000). Then for subsetRange = [0x1000, 0x2000), subsetMapping returns a
// MappingOfRange for which AddrRange = [0x5000, 0x6000).
func subsetMapping(wholeRange, subsetRange MappableRange, ms MappingSpace, addr usermem.Addr, writable bool) MappingOfRange {
	if !wholeRange.IsSupersetOf(subsetRange) {
		panic(fmt.Sprintf("%v is not a superset of %v", wholeRange, subsetRange))
	}

	offset := subsetRange.Start - wholeRange.Start
	start := addr + usermem.Addr(offset)
	return MappingOfRange{
		MappingSpace: ms,
		AddrRange: usermem.AddrRange{
			Start: start,
			End:   start + usermem.Addr(subsetRange.Length()),
		},
		Writable: writable,
	}
}

// AddMapping adds the given mapping and returns the set of MappableRanges that
// previously had no mappings.
//
// Preconditions: As for Mappable.AddMapping.
func (s *MappingSet) AddMapping(ms MappingSpace, ar usermem.AddrRange, offset uint64, writable bool) []MappableRange {
	mr := MappableRange{offset, offset + uint64(ar.Length())}
	var mapped []MappableRange
	seg, gap := s.Find(mr.Start)
	for {
		switch {
		case seg.Ok() && seg.Start() < mr.End:
			seg = s.Isolate(seg, mr)
			seg.Value()[subsetMapping(mr, seg.Range(), ms, ar.Start, writable)] = struct{}{}
			seg, gap = seg.NextNonEmpty()

		case gap.Ok() && gap.Start() < mr.End:
			gapMR := gap.Range().Intersect(mr)
			mapped = append(mapped, gapMR)
			// Insert a set and continue from the above case.
			seg, gap = s.Insert(gap, gapMR, make(MappingsOfRange)), MappingGapIterator{}

		default:
			return mapped
		}
	}
}

// RemoveMapping removes the given mapping and returns the set of
// MappableRanges that now have no mappings.
//
// Preconditions: As for Mappable.RemoveMapping.
func (s *MappingSet) RemoveMapping(ms MappingSpace, ar usermem.AddrRange, offset uint64, writable bool) []MappableRange {
	mr := MappableRange{offset, offset + uint64(ar.Length())}
	var unmapped []MappableRange

	seg := s.FindSegment(mr.Start)
	if !seg.Ok() {
		panic(fmt.Sprintf("MappingSet.RemoveMapping(%v): no segment containing %#x: %v", mr, mr.Start, s))
	}
	for seg.Ok() && seg.Start() < mr.End {
		// Ensure this segment is limited to our range.
		seg = s.Isolate(seg, mr)

		// Remove this part of the mapping.
		mappings := seg.Value()
		delete(mappings, subsetMapping(mr, seg.Range(), ms, ar.Start, writable))

		if len(mappings) == 0 {
			unmapped = append(unmapped, seg.Range())
			seg = s.Remove(seg).NextSegment()
		} else {
			seg = seg.NextSegment()
		}
	}
	s.MergeAdjacent(mr)
	return unmapped
}

// Invalidate calls MappingSpace.Invalidate for all mappings of offsets in mr.
func (s *MappingSet) Invalidate(mr MappableRange, opts InvalidateOpts) {
	for seg := s.LowerBoundSegment(mr.Start); seg.Ok() && seg.Start() < mr.End; seg = seg.NextSegment() {
		segMR := seg.Range()
		for m := range seg.Value() {
			region := subsetMapping(segMR, segMR.Intersect(mr), m.MappingSpace, m.AddrRange.Start, m.Writable)
			region.invalidate(opts)
		}
	}
}

// InvalidateAll calls MappingSpace.Invalidate for all mappings of s.
func (s *MappingSet) InvalidateAll(opts InvalidateOpts) {
	for seg := s.FirstSegment(); seg.Ok(); seg = seg.NextSegment() {
		for m := range seg.Value() {
			m.invalidate(opts)
		}
	}
}
