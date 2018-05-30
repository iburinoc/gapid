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

package vulkan

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/shadertools"
)

func (a API) GetGenerator() replay.Generator {
	return OverdrawCounter{a}
}

type OverdrawCounter struct {
	API
}

func (o OverdrawCounter) Replay(
	ctx context.Context,
	intent replay.Intent,
	cfg replay.Config,
	requests []replay.RequestAndResult,
	device *device.Instance,
	capture *capture.Capture,
	out transform.Writer) error {
	cmds := capture.Commands

	transforms := transform.Transforms{}
	transforms.Add(o)
	transforms.Transform(ctx, cmds, out)
	return nil
}

// Implement transform.Transformer

func (OverdrawCounter) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, output transform.Writer) {
	st := output.State()
	//s := GetState(st)
	l := st.MemoryLayout
	switch c := cmd.(type) {
	case *VkCreateShaderModule:
		fmt.Println(id, c)
		o := c.Extras().Observations()
		o.ApplyReads(st.Memory.ApplicationPool())
		info := c.PCreateInfo().MustRead(ctx, c, st, nil)
		size := uint64(info.CodeSize()) / 4
		fmt.Println("codesize:", size)
		words := info.PCode().Slice(0, size, l).MustRead(ctx, c, st, nil)
		dis := shadertools.DisassembleSpirvBinary(words)
		fmt.Println("code:\n", dis)
	}
	output.MutateAndWrite(ctx, id, cmd)
}
func (OverdrawCounter) Flush(ctx context.Context, output transform.Writer) {

}
