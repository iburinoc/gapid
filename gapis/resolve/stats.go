// Copyright (C) 2017 Google Inc.
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

package resolve

import (
	"context"
	"sort"
	"strconv"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Stats resolves and returns the stats list from the path p.
func Stats(ctx context.Context, p *path.Stats) (*service.ConstantSet, error) {
	d, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	// Get the present calls
	events, err := Events(ctx, &path.Events{
		Capture:     p.Capture,
		LastInFrame: true,
	})
	if err != nil {
		return nil, err
	}

	subcmdsExecuted := computeSubcmdExecutor(d)

	/*
		keys := make([]api.CmdID, 0, len(subcmdsExecuted))
		for k := range subcmdsExecuted {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, k := range keys {
			fmt.Println(k)
			for _, subcmd := range subcmdsExecuted[k] {
				fmt.Println("\t", subcmd)
			}
		}
	*/

	drawsPerFrame := make([]uint64, len(events.List))

	drawsSinceLastFrame := uint64(0)
	processed := api.CmdIDSet{}

	var process func(id api.CmdID) error
	process = func(id api.CmdID) error {
		if processed.Contains(id) {
			return nil
		}
		processed.Add(id)

		deps, ok := d.SyncDependencies[id]
		if ok {
			for _, dep := range deps {
				err := process(dep)
				if err != nil {
					return err
				}
			}
		}

		cmd, err := Cmd(ctx, &path.Command{
			Capture: p.Capture,
			Indices: []uint64{uint64(id)},
		})
		if err != nil {
			return err
		}

		subcmds, ok := subcmdsExecuted[id]
		if ok {
			// There are subcommands, so don't count the parent command
			// (since vkQueueSubmit is considered a "draw call")
			for _, indices := range subcmds {
				subcmd, err := Cmd(ctx, &path.Command{
					Capture: p.Capture,
					Indices: indices,
				})
				if err != nil {
					return err
				}
				if subcmd.CmdFlags(ctx, id, nil).IsExecutedDraw() {
					drawsSinceLastFrame++
				}
			}
		} else {
			// NOTE: nil is being used as the state to avoid having
			// to compute a mutated state.  This works for now as
			// CmdFlags doesn't reference the state, but theoretically
			// in the future it could, and that would break this.
			if cmd.CmdFlags(ctx, id, nil).IsDrawCall() {
				drawsSinceLastFrame++
			}
		}

		return nil
	}

	hostSyncIdx := 0
	for i, event := range events.List {
		id := api.CmdID(event.Command.Indices[0])
		// Need to get to the present call
		for hostSyncIdx < len(d.HostSyncBarriers) && d.HostSyncBarriers[hostSyncIdx] < id {
			err := process(d.HostSyncBarriers[hostSyncIdx])
			if err != nil {
				return nil, err
			}
			hostSyncIdx++
		}
		err := process(id)
		if err != nil {
			return nil, err
		}
		drawsPerFrame[i] = drawsSinceLastFrame
		drawsSinceLastFrame = uint64(0)
	}

	constants := make([]*service.Constant, len(drawsPerFrame))
	for i, val := range drawsPerFrame {
		constants[i] = &service.Constant{
			Name:  strconv.Itoa(i),
			Value: val,
		}
	}

	return &service.ConstantSet{
		Constants: constants,
	}, nil
}

// Compute a map from command id's to the subcommands they will actually execute.
// I.e. a vkSetEvent will list all subcommands from the queue it executed.
func computeSubcmdExecutor(d *sync.Data) map[api.CmdID][][]uint64 {
	type commandExecutor struct {
		lastIdx  api.SubCmdIdx
		executor api.CmdID
	}

	keys := d.SortedKeys()
	res := map[api.CmdID][][]uint64{}

	for _, k := range keys {
		subcmds := d.SubcommandReferences[k]
		executors := make([]commandExecutor, 0, len(d.CommandRanges[k].Ranges))
		for executor, lastIdx := range d.CommandRanges[k].Ranges {
			executors = append(executors, commandExecutor{
				lastIdx:  lastIdx,
				executor: executor,
			})
		}
		sort.Slice(executors, func(i, j int) bool {
			return executors[i].lastIdx.LessThan(executors[j].lastIdx)
		})
		// Make sure there's a (possibly empty) entry for the original
		// command, so we can use that to distinguish it as a queue submit.
		// We need to do this so it doesn't get counted as a draw call.
		if _, ok := res[k]; !ok {
			res[k] = [][]uint64{}
		}
		rangeIdx := 0
		for _, val := range subcmds {
			for executors[rangeIdx].lastIdx.LessThan(val.Index) {
				rangeIdx++
			}
			executor := executors[rangeIdx].executor
			indices := append([]uint64{uint64(k)}, val.Index...)
			val, ok := res[executor]
			if !ok {
				val = [][]uint64{}
			}
			res[executor] = append(val, indices)
		}
	}

	return res
}
