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
				// NOTE: nil is being used as the state to avoid having
				// to compute a mutated state.  This works for now as
				// CmdFlags doesn't reference the state, but theoretically
				// in the future it could, and that would break this.
				flags := cmd.CmdFlags(ctx, api.CmdID(idx[0]), nil)
				if (len(idx) == 1 && flags.IsDrawCall()) ||
					(len(idx) > 1 && flags.IsExecutedDraw()) {

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

	for i, event := range events.List {
		id := api.CmdID(event.Command.Indices[0])
		pt := d.CmdSyncPoints[id]
		// Need to get to the present call
		err := process(pt)
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
