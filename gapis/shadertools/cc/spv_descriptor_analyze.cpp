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
#include "third_party/SPIRV-Tools/source/opt/function.h"
#include "third_party/SPIRV-Tools/source/opt/instruction.h"
#include "third_party/SPIRV-Tools/source/opt/ir_context.h"

#include <iostream>
#include <memory>
#include <stdexcept>
#include <string>
#include <unordered_map>
#include <vector>

namespace spv_descriptor_analyze {

using spvtools::ir::Function;
using spvtools::ir::Instruction;
using spvtools::ir::IRContext;
using spvtools::ir::Operand;

// Converts a SPIR-V string operand to std::string
std::string ParseStringOperand(const Operand& op) {
#if __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN
  return std::string(reinterpret_cast<const char*>(op.words().data()));
#else
  // manually convert from little-endian
  std::string res;
  res.reserve(op.words.size() * sizeof(uint32_t));
  for (uint32_t word : op.words) {
    while ((word & 0xff) != 0) {
      res += (char)(word & 0xff);
      word >>= 8;
    }
  }

  return res;
#endif
}

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
                "variable " + std::to_string(id) +
                +" has descriptor set but isn't a uniform");
          }
          annots[id].first = inst.GetSingleWordOperand(2);
          break;
        case SpvDecorationBinding:
          if (uniforms.find(id) == uniforms.end()) {
            throw std::runtime_error("variable " + std::to_string(id) +
                                     +" has binding but isn't a uniform");
          }
          annots[id].second = inst.GetSingleWordOperand(2);
          break;
      }
    }
  }
  return annots;
}

static std::unordered_set<uint32_t> subFunctions(const Function& func) {
  std::unordered_set<uint32_t> calls;
  func.ForEachInst([&calls](const Instruction* inst) {
    if (inst->opcode() == SpvOp::SpvOpFunctionCall) {
      calls.insert(inst->GetSingleWordOperand(2));
    }
  });
  return calls;
}

// Returns the set of variables used in the function that are in the set `from`
static std::unordered_set<uint32_t> usedVariables(
    const Function& func, const std::unordered_set<uint32_t>& from) {
  std::unordered_set<uint32_t> used;
  func.ForEachInst([&](const Instruction* inst) {
    auto try_add = [&used, &from, inst](uint32_t i) {
      uint32_t val = inst->GetSingleWordOperand(i);
      if (from.find(val) != from.end()) used.insert(val);
    };
    // Uniforms are pointers, so the only operands we need to look at are the
    // pointer ones.
    switch (inst->opcode()) {
      case SpvOp::SpvOpLoad:
      case SpvOp::SpvOpAccessChain:
      case SpvOp::SpvOpInBoundsAccessChain:
      case SpvOp::SpvOpPtrAccessChain:
      case SpvOp::SpvOpArrayLength:
      case SpvOp::SpvOpGenericPtrMemSemantics:
      case SpvOp::SpvOpInBoundsPtrAccessChain:
        try_add(2);
        break;
      case SpvOp::SpvOpStore:
        try_add(0);
        break;
      case SpvOp::SpvOpCopyMemory:
      case SpvOp::SpvOpCopyMemorySized:
        try_add(0);
        try_add(1);
        break;
      default:
        break;
    }
  });

  return used;
}

static std::unordered_map<uint32_t, std::unordered_set<uint32_t>>
staticallyUsed(const std::unique_ptr<IRContext>& context,
               const std::unordered_set<uint32_t>& from) {
  std::unordered_map<uint32_t, const Function*> funcs;

  for (const auto& func : *context->module()) {
    funcs[func.result_id()] = &func;
  }
  std::unordered_map<uint32_t, std::unordered_set<uint32_t>> dp;

  std::function<void(uint32_t)> dfs;
  dfs = [&](uint32_t func) {
    if (dp.find(func) != dp.end()) {
      throw std::runtime_error("module call graph recurses");
    }
    dp[func] = std::unordered_set<uint32_t>();
    for (uint32_t callee : subFunctions(*funcs[func])) {
      if (dp.find(callee) == dp.end()) dfs(callee);
      dp[func].insert(dp[callee].begin(), dp[callee].end());
    }
    const auto& usedVars = usedVariables(*funcs[func], from);
    dp[func].insert(usedVars.begin(), usedVars.end());
  };

  // call it on the entry points, and build a separate map of their results
  std::unordered_map<uint32_t, std::unordered_set<uint32_t>> entry_points;
  for (const auto& entry : context->module()->entry_points()) {
    uint32_t id = entry.GetSingleWordOperand(1);
    // TODO: determine if entry points can call other entry points
    if (dp.find(id) == dp.end()) {
      dfs(id);
    }
    entry_points[id] = dp[id];
  }
  return entry_points;
}

// FIXME
void AnalyzeModule(const std::vector<uint32_t>& spv_binary) {
  auto print_msg_to_stderr = [](spv_message_level_t, const char*,
                                const spv_position_t&, const char* m) {
    std::cerr << "error: " << m << std::endl;
  };

  const auto context =
      spvtools::BuildModule(SPV_ENV_UNIVERSAL_1_1, print_msg_to_stderr,
                            spv_binary.data(), spv_binary.size());

  const auto& uniforms = getUniforms(context);

  // map from variable id to <descriptor set, binding> pairs
  const auto& annots = getUniformAnnotations(context, uniforms);

  // map from entry point id to id's of statically used uniforms
  const auto& statically_used = staticallyUsed(context, uniforms);

  for (const auto& entry : annots) {
    std::cout << entry.first << ": (" << annots.at(entry.first).first << ", "
              << annots.at(entry.first).second << ")\n";
  }

  for (const auto& entry : context->module()->entry_points()) {
    uint32_t id = entry.GetSingleWordOperand(1);
    std::cout << ParseStringOperand(entry.GetOperand(2)) << std::endl;
    for (const auto& var : statically_used.at(id)) {
      std::cout << var << " ";
    }
    std::cout << std::endl;
  }
}

}  // namespace spv_descriptor_analyze
