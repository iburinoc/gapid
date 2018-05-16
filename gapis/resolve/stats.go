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
	"strconv"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Stats resolves and returns the stats list from the path p.
func Stats(ctx context.Context, p *path.Stats) (*service.ConstantSet, error) {
	d, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}
	cmds, err := Cmds(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	st, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}
	flags := make([]api.CmdFlags, len(cmds))

	// Get the present calls
	events, err := Events(ctx, &path.Events{
		Capture:     p.Capture,
		LastInFrame: true,
	})
	if err != nil {
		return nil, err
	}

	drawsPerFrame := make([]uint64, len(events.List))

	drawsSinceLastFrame := uint64(0)
	processed := map[sync.SyncIdx]struct{}{}

	var process func(pt sync.SyncIdx) error
	process = func(pt sync.SyncIdx) error {
		if _, ok := processed[pt]; ok {
			return nil
		}
		processed[pt] = struct{}{}

		ptObj := d.SyncPoints[pt]
		if cmdIdx, ok := ptObj.(sync.CmdIdx); ok {
			idx := cmdIdx.Idx
			cmd, err := Cmd(ctx, &path.Command{
				Capture: p.Capture,
				Indices: []uint64(idx),
			})
			if err != nil {
				return err
			}
			// If the command has subcommands, ignore it (vkQueueSubmit or similar)
			if _, ok := d.SubcommandReferences[api.CmdID(idx[0])]; len(idx) > 1 || !ok {
				var cmdflags api.CmdFlags
				if len(idx) == 1 {
					cmdflags = flags[idx[0]]
				} else {
					// NOTE: For subcommands its not clear
					// what the "correct" state to present
					// to CmdFlags is.  Since Vulkan
					// currently does not use the state,
					// pass nil here instead of a
					// potentially "incorrect" state.
					cmdflags = cmd.CmdFlags(ctx, api.CmdID(idx[0]), nil)
				}
				if (len(idx) == 1 && cmdflags.IsDrawCall()) ||
					(len(idx) > 1 && cmdflags.IsExecutedDraw()) {

					drawsSinceLastFrame += 1
				}
			}
		}

		deps, ok := d.SyncDependencies[pt]
		if ok {
			for _, dep := range deps {
				err := process(dep)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	cmdIdx := uint64(0)
	for i, event := range events.List {
		for cmdIdx <= event.Command.Indices[0] {
			cmd := cmds[cmdIdx]
			err := cmd.Mutate(ctx, api.CmdID(cmdIdx), st, nil)
			if err != nil {
				return nil, err
			}
			flags[cmdIdx] = cmd.CmdFlags(ctx, api.CmdID(cmdIdx), st)

			// For commands from non-synchronized API's, just
			// process the draw calls between each frame boundary.
			if _, ok := cmd.API().(sync.SynchronizedAPI); !ok {
				// NOTE: see above on CmdFlags
				if flags[cmdIdx].IsDrawCall() {
					drawsSinceLastFrame += 1
				}
			}
			cmdIdx += 1
		}
		id := api.CmdID(event.Command.Indices[0])
		cmd := cmds[id]
		// If the frame boundary was on a synchronized api, process its dependencies
		if _, ok := cmd.API().(sync.SynchronizedAPI); ok {
			pt := d.CmdSyncPoints[id]
			err := process(pt)
			if err != nil {
				return nil, err
			}
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
