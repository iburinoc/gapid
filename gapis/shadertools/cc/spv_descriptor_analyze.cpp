/*
 * Copyright (C) 2018 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "third_party/SPIRV-Headers/include/spirv/unified1/spirv.hpp"
#include "third_party/SPIRV-Tools/source/opt/build_module.h"
#include "third_party/SPIRV-Tools/source/opt/ir_context.h"

#include <iostream>
#include <memory>
#include <stdexcept>
#include <unordered_map>
#include <vector>

namespace spv_descriptor_analyze {

using spvtools::ir::IRContext;

static std::unordered_set<uint32_t> getUniforms(
    const std::unique_ptr<IRContext>& context) {
  std::unordered_set<uint32_t> uniforms;

  for (const auto& inst : context->types_values()) {
    if (inst.opcode() == SpvOp::SpvOpVariable) {
      auto storage_class = inst.GetSingleWordOperand(2);
      if (storage_class == SpvStorageClassUniform ||
          storage_class == SpvStorageClassUniformConstant) {
        uniforms.insert(inst.result_id());
      }
    }
  }
  return uniforms;
}

static std::unordered_map<uint32_t, std::pair<uint32_t, uint32_t>>
getUniformAnnotations(const std::unique_ptr<IRContext>& context,
                      const std::unordered_set<uint32_t>& uniforms) {
  std::unordered_map<uint32_t, std::pair<uint32_t, uint32_t>> annots;
  for (const auto& inst : context->annotations()) {
    if (inst.opcode() == SpvOp::SpvOpDecorate) {
      uint32_t id = inst.GetSingleWordOperand(0);

      switch (inst.GetSingleWordOperand(1)) {
        case SpvDecorationDescriptorSet:
          if (uniforms.find(id) == uniforms.end()) {
            throw std::runtime_error(
                "error: variable " + std::to_string(id) +
                +" has descriptor set but isn't a uniform");
          }
          annots[id].first = inst.GetSingleWordOperand(2);
          break;
        case SpvDecorationBinding:
          if (uniforms.find(id) == uniforms.end()) {
            throw std::runtime_error("error: variable " + std::to_string(id) +
                                     +" has binding but isn't a uniform");
          }
          annots[id].second = inst.GetSingleWordOperand(2);
          break;
      }
    }
  }
  return annots;
}

void analyze_module(const std::vector<uint32_t>& spv_binary) {
  auto print_msg_to_stderr = [](spv_message_level_t, const char*,
                                const spv_position_t&, const char* m) {
    std::cerr << "error: " << m << std::endl;
  };

  const auto context =
      spvtools::BuildModule(SPV_ENV_UNIVERSAL_1_1, print_msg_to_stderr,
                            spv_binary.data(), spv_binary.size());

  std::unordered_set<uint32_t> uniforms = getUniforms(context);

  // map from variable id to <descriptor set, binding> pairs
  std::unordered_map<uint32_t, std::pair<uint32_t, uint32_t>> annots =
      getUniformAnnotations(context, uniforms);

  for (const auto& entry : annots) {
    std::cout << entry.first << ": (" << annots[entry.first].first << ", "
              << annots[entry.first].second << ")\n";
  }
}

}  // namespace spv_descriptor_analyze
