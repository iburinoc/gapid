// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type memoryVerb MemoryFlags

func init() {
	verb := &memoryVerb{}
	app.AddVerb(&app.Verb{
		Name:      "memory",
		ShortHelp: "Prints memory metrics about a capture file",
		Action:    verb,
	})
}

func (verb *memoryVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	filepath, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return log.Errf(ctx, err, "Finding file: %v", flags.Arg(0))
	}

	client, err := getGapis(ctx, verb.Gapis, GapirFlags{})
	if err != nil {
		return log.Err(ctx, err, "Failed to connect to the GAPIS server")
	}

	capture, err := client.LoadCapture(ctx, filepath)
	if err != nil {
		return log.Errf(ctx, err, "LoadCapture(%v)", filepath)
	}

	if len(verb.At) == 0 {
		boxedCapture, err := client.Get(ctx, capture.Path())
		if err != nil {
			return log.Err(ctx, err, "Failed to load the capture")
		}
		verb.At = []uint64{uint64(boxedCapture.(*service.Capture).NumCommands) - 1}
	}

	boxedVal, err := client.Get(ctx, (&path.Metrics{
		Command: capture.Command(verb.At[0], verb.At[1:]...),
		Type:    path.Metrics_MEMORY,
	}).Path())
	if err != nil {
		return log.Errf(ctx, err, "Failed to load metrics")
	}

	mem := boxedVal.(*service.Metrics).Metrics.(*service.Metrics_MemoryLayout).MemoryLayout

	allocationFlags := []*service.Constant{}
	if mem.AllocationFlagsIndex != -1 {
		boxedConstants, err := client.Get(ctx, (&path.ConstantSet{
			Api:   mem.Api,
			Index: uint32(mem.AllocationFlagsIndex),
		}).Path())
		if err != nil {
			return log.Errf(ctx, err, "Failed to load allocation flag names")
		}
		constants := boxedConstants.(*service.ConstantSet)
		// If not a bitfield, we can't compare it against the flags
		if constants.IsBitfield {
			allocationFlags = constants.Constants
		}
	}

	fmt.Printf("%v memory allocations\n", len(mem.Allocations))
	sort.Slice(mem.Allocations, func(i, j int) bool {
		return mem.Allocations[i].Handle < mem.Allocations[j].Handle
	})
	for _, alloc := range mem.Allocations {
		fmt.Println("Handle:", alloc.Handle)
		fmt.Println("\tDevice:      ", alloc.Device)
		fmt.Println("\tMemory Type: ", alloc.MemoryType)
		fmt.Println("\tSize:        ", alloc.Size)

		if alloc.Flags != 0 && len(allocationFlags) != 0 {
			fmt.Println("\tFlags:")
			for _, f := range allocationFlags {
				if (alloc.Flags & uint32(f.Value)) != 0 {
					fmt.Printf("\t\t%v\n", f.Name)
				}
			}
		}

		if alloc.Mapping.Size != 0 {
			fmt.Printf("\tMapped into host memory at 0x%x\n",
				alloc.Mapping.HostAddress)
			fmt.Println("\t\tOffset:", alloc.Mapping.Offset)
			fmt.Println("\t\tSize:  ", alloc.Mapping.Size)
		}

		bindings := bindingSlice(alloc.Bindings)
		sort.Slice(bindings, bindings.BindingLess)
		fmt.Printf("\t%v Bindings:\n", len(bindings))
		for _, binding := range bindings {
			var typ string
			switch binding.Type {
			case service.MemoryBinding_BUFFER:
				typ = "Buffer"
			case service.MemoryBinding_IMAGE:
				typ = "Image"
			}
			fmt.Printf("\t%v: %v\n", typ, binding.Handle)

			fmt.Println("\t\tOffset: ", binding.Offset)
			fmt.Println("\t\tSize:   ", binding.Size)
		}

		aliases := bindings.computeAliasing()
		if len(aliases) == 0 {
			fmt.Println("\tNo aliased regions")
		} else {
			fmt.Printf("\t%v aliased regions:\n", len(aliases))
			for i, a := range aliases {
				fmt.Printf("\t%v:\n", i)
				fmt.Println("\t\tOffset: ", a.offset)
				fmt.Println("\t\tSize:   ", a.size)
				fmt.Println("\t\tShared by:")
				for _, s := range a.sharers {
					fmt.Printf("\t\t\t%v\n", s)
				}
			}
		}
	}
	return nil
}

type bindingSlice []*service.MemoryBinding

func (bindings bindingSlice) BindingLess(i, j int) bool {
	if bindings[i].Offset < bindings[j].Offset {
		return true
	} else if bindings[i].Offset > bindings[j].Offset {
		return false
	}
	if bindings[i].Size < bindings[j].Size {
		return true
	} else if bindings[i].Size > bindings[j].Size {
		return false
	}
	return bindings[i].Handle < bindings[j].Handle
}

type alias struct {
	offset uint64
	size   uint64

	sharers []uint64
}

func (bindings bindingSlice) computeAliasing() []alias {
	startsAt := map[uint64][]uint64{}
	endsAt := map[uint64][]uint64{}
	pointSet := map[uint64]struct{}{}

	for _, b := range bindings {
		start := b.Offset
		end := start + b.Size

		s, _ := startsAt[start]
		startsAt[start] = append(s, b.Handle)
		pointSet[start] = struct{}{}

		e, _ := endsAt[end]
		endsAt[end] = append(e, b.Handle)
		pointSet[end] = struct{}{}
	}

	points := make([]uint64, 0, len(pointSet))
	for k := range pointSet {
		points = append(points, k)
	}
	sort.Slice(points, func(i, j int) bool { return points[i] < points[j] })

	aliases := []alias{}
	active := map[uint64]struct{}{}
	for i, p := range points {
		e, _ := endsAt[p]
		for _, handle := range e {
			delete(active, handle)
		}
		s, _ := startsAt[p]
		for _, handle := range s {
			active[handle] = struct{}{}
		}

		if len(active) > 1 && i < len(points)-1 {
			sharers := []uint64{}
			for k := range active {
				sharers = append(sharers, k)
			}
			aliases = append(aliases, alias{
				offset:  p,
				size:    points[i+1] - p,
				sharers: sharers,
			})
		}
	}

	return aliases
}
