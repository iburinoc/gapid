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

package resolve

import (
	"context"
	"fmt"

	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/devices"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type ReplaySource interface {
	GetGenerator() replay.Generator
}

func Overdraw(ctx context.Context, p *path.Overdraw) (*service.Overdraw, error) {
	devices, err := devices.ForReplay(ctx, p.Command.Capture)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("No compatible replay devices found")
	}
	device := devices[0]

	intent := replay.Intent{
		Device:  device,
		Capture: p.Command.Capture,
	}

	cmd, err := Cmd(ctx, p.Command)
	if err != nil {
		return nil, err
	}

	api := cmd.API()
	if api == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrFramebufferUnavailable()}
	}

	rs, ok := api.(ReplaySource)
	if !ok {
		return nil, fmt.Errorf("API not compatible")
	}

	gen := rs.GetGenerator()
	mgr := replay.GetManager(ctx)

	fmt.Println("replaying")
	res, err := mgr.Replay(ctx, intent, struct{}{}, struct{}{}, gen, nil)
	fmt.Println("replaying done")
	if err != nil {
		return nil, err
	}
	_ = res
	return &service.Overdraw{}, nil
}
