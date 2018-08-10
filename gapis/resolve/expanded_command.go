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
	"reflect"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/service/path"
)

func ExpandedCommand(ctx context.Context, p *path.ExpandedCommand) (*api.Command, error) {
	data, err := database.Build(ctx, &ExpandedCommandResolvable{Path: p})
	if err != nil {
		return nil, err
	}
	c, ok := data.(*api.Command)
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not get expanded command data")
	}
	return c, nil
}

func (r *ExpandedCommandResolvable) Resolve(ctx context.Context) (interface{}, error) {
	st, err := GlobalState(ctx, &path.GlobalState{After: r.Path.Command})
	cmd, err := Cmd(ctx, r.Path.Command)
	if err != nil {
		return nil, err
	}

	m, err := Manager(ctx, r.Path.Command.Capture)
	if err != nil {
		return nil, err
	}
	cm := m.Get(cmd)

	cmd.Extras().Observations().ApplyReads(st.Memory.ApplicationPool())
	cmd.Extras().Observations().ApplyWrites(st.Memory.ApplicationPool())

	for _, prop := range cmd.CmdParams() {
		printParamTree(ctx, cmd, "", prop, st, cm)
	}

	return nil, err
}

func printParamTree(ctx context.Context, cmd api.Cmd, indent string, prop *api.Property, s *api.GlobalState, cm api.CommandManager) {
	fmt.Printf("%v%v: ", indent, prop.Name)
	val := prop.Get()
	ptr, isPtr := val.(memory.Pointer)
	fmt.Printf("%v\n", val)
	if isPtr {
		if alen, ok := cm[ptr.Address()]; ok {
			// Need to get the slice of values
			rval := reflect.ValueOf(val)
			slice := rval.MethodByName("Slice").Call(
				[]reflect.Value{reflect.ValueOf(uint64(0)), reflect.ValueOf(alen), reflect.ValueOf(s.MemoryLayout), reflect.ValueOf(api.CommandManager(nil))},
			)[0]
			read := slice.MethodByName("MustRead")
			if read.IsValid() {
				values := read.Call(
					[]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(cmd), reflect.ValueOf(s), reflect.ValueOf((*builder.Builder)(nil))},
				)[0]
				for i := 0; i < values.Len(); i++ {
					val := values.Index(i)
					fmt.Printf("%v%v:", indent, i)
					if val.Type().Kind() == reflect.Struct {
						fmt.Printf("\n")
						props := val.MethodByName("Properties").Call([]reflect.Value{})[0].Interface().(api.Properties)
						for _, prop := range props {
							printParamTree(ctx, cmd, indent+"\t", prop, s, cm)
						}
					} else {
						fmt.Printf(" %v\n", val.Interface())
					}
				}
			}
		}
	}
	_ = ptr
}

func Manager(ctx context.Context, c *path.Capture) (api.Manager, error) {
	data, err := database.Build(ctx, &ManagerResolvable{Path: c})
	if err != nil {
		return nil, err
	}
	m, ok := data.(api.Manager)
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not get capture manager")
	}
	return m, nil
}

func (r *ManagerResolvable) Resolve(ctx context.Context) (interface{}, error) {
	return api.Manager{}, nil
}
