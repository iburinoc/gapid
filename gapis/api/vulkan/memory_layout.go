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

package vulkan

import (
	"fmt"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

// MemoryLayout stores an overview of the state's memory layout into a
// service.MemoryLayout object.  The layout includes data on memory types,
// allocations, and which resources are bound to which locations in memory.
func (s *State) MemoryLayout() (*service.MemoryLayout, error) {
	allocations := []*service.MemoryAllocation{}
	// Serialize data on all allocations into protobufs
	for handle, info := range *(s.DeviceMemories().Map) {
		device := info.Device()
		typ := info.MemoryTypeIndex()
		flags, err := s.getMemoryTypeFlags(device, typ)
		if err != nil {
			return nil, err
		}
		bindings, err := s.getAllocationBindings(info.Get())
		if err != nil {
			return nil, err
		}

		mapping := service.MemoryMapping{
			Size:        uint64(info.MappedSize()),
			Offset:      uint64(info.MappedOffset()),
			HostAddress: uint64(info.MappedLocation()),
		}

		alloc := service.MemoryAllocation{
			Device:     uint64(info.Device()),
			MemoryType: uint32(typ),
			Flags:      uint32(flags),
			Handle:     uint64(handle),
			Size:       uint64(info.AllocationSize()),
			Mapping:    &mapping,
			Bindings:   bindings,
		}

		allocations = append(allocations, &alloc)
	}
	return &service.MemoryLayout{
		Api:                  path.NewAPI(id.ID(ID)),
		AllocationFlagsIndex: int32(VkMemoryPropertyFlagBitsConstants()),
		Allocations:          allocations,
	}, nil
}

func (s *State) getMemoryTypeFlags(device VkDevice, typeIndex uint32) (VkMemoryPropertyFlags, error) {
	deviceObject := s.Devices().Get(device)
	if deviceObject.IsNil() {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Failed to find device %v", device)
	}
	physicalDevice := deviceObject.PhysicalDevice()
	physicalDeviceObject := s.PhysicalDevices().Get(physicalDevice)
	if physicalDeviceObject.IsNil() {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Failed to find physical device %v", physicalDevice)
	}
	props := physicalDeviceObject.MemoryProperties()
	if props.MemoryTypeCount() <= typeIndex {
		return VkMemoryPropertyFlags(0), fmt.Errorf("Memory type %v is larger than physical device %v's number of memory types (%v)",
			typeIndex, physicalDevice, props.MemoryTypeCount())
	}
	return props.MemoryTypes().Get(int(typeIndex)).PropertyFlags(), nil
}

func (s *State) getAllocationBindings(allocation DeviceMemoryObject) ([]*service.MemoryBinding, error) {
	bindings := []*service.MemoryBinding{}
	for handle, offset := range allocation.BoundObjects().All() {
		var size VkDeviceSize
		var typ service.MemoryBinding_BindingType
		if buffer, ok := s.Buffers().Lookup(VkBuffer(handle)); ok {
			typ = service.MemoryBinding_BUFFER
			size = buffer.Info().Size()
		} else if image, ok := s.Images().Lookup(VkImage(handle)); ok {
			typ = service.MemoryBinding_IMAGE
			size = image.MemoryRequirements().Size()
		} else {
			return nil, fmt.Errorf("Bound object %v is not a buffer or an image", handle)
		}

		bindings = append(bindings, &service.MemoryBinding{
			Handle: handle,
			Type:   typ,
			Size:   uint64(size),
			Offset: uint64(offset),
		})
	}
	return bindings, nil
}
