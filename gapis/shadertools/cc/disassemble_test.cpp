/*
 * Copyright (C) 2017 Google Inc.
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

#include "spv_manager.h"
#include "libmanager.h"

#include <stdint.h>
#include <stdio.h>
#include <vector>

void overdraw_test(const std::vector<uint32_t> &spirv_binary) {
  // TODO: build shaders
  spvmanager::SpvManager manager(spirv_binary);

  manager.addOverdrawStorageBuffer();

  std::vector<uint32_t> result = manager.getSpvBinary();
  if (FILE *fp = fopen("out.spv", "wb")) {
    if (fwrite(&result[0], sizeof(uint32_t), result.size(), fp) != result.size()) {
      fprintf(stderr, "error: failed to write file\n");
    }
    fclose(fp);
  } else {
    fprintf(stderr, "error: failed to open output file\n");
  }
  const char *dis_text = getDisassembleText(result.data(), result.size());
  if (dis_text) {
  	printf("%s\n", dis_text);
  } else {
    printf("Disassemble error.\n");
    return;
  }
  deleteDisassembleText(dis_text);
}

int main(int argc, char* argv[]) {

  const char* filename = argv[1];

  std::vector<uint32_t> spirv_binary;
  const int buf_size = 1024;
  if (FILE* fp = fopen(filename, "rb")) {
    uint32_t buf[buf_size];
    while (size_t len = fread(buf, sizeof(uint32_t), buf_size, fp)) {
      spirv_binary.insert(spirv_binary.end(), buf, buf + len);
    }
    if (ftell(fp) == -1L) {
      if (ferror(fp)) {
        fprintf(stderr, "error: error reading file '%s'\n", filename);
        return 1;
      }
    } else {
      if (sizeof(uint32_t) != 1 && (ftell(fp) % sizeof(uint32_t))) {
        fprintf(stderr, "error: corrupted word found in file '%s'\n", filename);
        return 1;
      }
    }
    fclose(fp);
  } else {
    fprintf(stderr, "error: file does not exist '%s'\n", filename);
    return 1;
  }

  overdraw_test(spirv_binary);
  return 0;

  const char* dis_text = getDisassembleText(spirv_binary.data(), spirv_binary.size());

  if (dis_text) {
  	printf("%s\n", dis_text);
  } else {
    printf("Disassemble error.\n");
    return 2;
  }

  deleteDisassembleText(dis_text);

  return 0;
}
