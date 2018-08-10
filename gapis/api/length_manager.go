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

package api

import "github.com/google/gapid/gapis/memory"

type CommandManager map[uint64]uint64

func (cm CommandManager) Slicing(p memory.Pointer, start, end uint64) {
	v, _ := cm[p.Address()]
	if v <= end {
		cm[p.Address()] = end
	}
}

type Manager map[Cmd]CommandManager

func (m Manager) Get(c Cmd) CommandManager {
	if c == nil || m == nil {
		return nil
	}

	cm, ok := m[c]
	if !ok {
		cm = CommandManager{}
		m[c] = cm
	}
	return cm
}
