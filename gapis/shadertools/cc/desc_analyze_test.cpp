#include <stdint.h>
#include <stdio.h>
#include <vector>

#include "spv_descriptor_analyze.h"

int main(int argc, char** argv) {
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

  spv_descriptor_analyze::analyze_module(spirv_binary);
}
