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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// Stats resolves and returns the stats list from the path p.
func Stats(ctx context.Context, p *path.Stats) (*service.ConstantSet, error) {
	d, err := SyncData(ctx, p.Capture)
	if err != nil {
		return nil, err
	}

	// Get the vkQueuePresentKHR's
	events, err := Events(ctx, &path.Events{
		LastInFrame: true,
	})
	if err != nil {
		return nil, err
	}

	drawsPerFrame := make([]uint64, len(events.List))

	drawsSinceLastFrame := uint64(0)
	processed := api.CmdIDSet{}

	var process func(id api.CmdID)
	process = func(id api.CmdID) {
		if processed.Contains(id) {
			return
		}
		processed.Add(id)

		deps, ok := d.SyncDependencies[id]
		if ok {
			for _, dep := range deps {
				process(dep)
			}
		}
	}

	hostSyncIdx := 0
	for i, event := range events.List {
		id := api.CmdID(event.Command.Indices[0])
		// Need to get to the present call
		for d.HostSyncBarriers[hostSyncIdx] < id {
			process(d.HostSyncBarriers[hostSyncIdx])
			hostSyncIdx++
		}
		process(id)
		drawsPerFrame[i] = drawsSinceLastFrame
		drawsSinceLastFrame = uint64(0)
	}

	constants := make([]*service.Constant, len(drawsPerFrame))
	for i, val := range drawsPerFrame {
		constants[i] = &service.Constant{
			Value: val,
		}
	}

	return &service.ConstantSet{
		Constants: constants,
	}, nil
}
