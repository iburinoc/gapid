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
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

func Metrics(ctx context.Context, p *path.Metrics) (*service.Metrics, error) {
	switch p.Type {
	case path.Metrics_MEMORY:
		return memoryMetrics(ctx, p)
	default:
		return nil, fmt.Errorf("Metrics type %v not implemented", p.Type)
	}

}

// DO NOT MERGE
// Not sure where the best spot for this is
type MemoryMetrics interface {
	MemoryLayout() (*service.MemoryLayout, error)
}

func memoryMetrics(ctx context.Context, p *path.Metrics) (*service.Metrics, error) {
	cmd, err := Cmd(ctx, p.Command)
	if err != nil {
		return nil, err
	}
	api := cmd.API()
	if api == nil {
		return nil, &service.ErrDataUnavailable{Reason: messages.ErrStateUnavailable()}
	}

	state, err := GlobalState(ctx, p.Command.GlobalStateAfter())
	if err != nil {
		return nil, err
	}

	apiState, ok := state.APIs[api.ID()]
	if !ok {
		return nil, fmt.Errorf("API state not found for command %v", p.Command)
	}

	if mm, ok := apiState.(MemoryMetrics); ok {
		val, err := mm.MemoryLayout()
		if err != nil {
			return nil, err
		}
		return &service.Metrics{&service.Metrics_MemoryLayout{val}}, nil
	} else {
		return nil, fmt.Errorf("Memory breakdown not supported for API %v", api.Name())
	}

	return nil, fmt.Errorf("No APIs found with support for memory layout reporting")
}
