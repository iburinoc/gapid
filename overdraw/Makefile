VULKAN_SDK_PATH = ${HOME}/VulkanSDK/1.1.73.0/x86_64
CFLAGS = -std=c++11 -I$(VULKAN_SDK_PATH)/include
LDFLAGS = -L$(VULKAN_SDK_PATH)/lib `pkg-config --static --libs glfw3` -lvulkan

GLSLANG_VALIDATOR = $(VULKAN_SDK_PATH)/bin/glslangValidator

NAME = HelloTriangle

$(NAME): main.cpp triangle.frag.inc triangle.vert.inc
	/usr/bin/clang++ $(CFLAGS) -o $@ $< $(LDFLAGS)

%.inc: %
	$(GLSLANG_VALIDATOR) -V -x -o $@ $<

.PHONY: test clean

test: $(NAME)
	LD_LIBRARY_PATH=$(VULKAN_SDK_PATH)/lib VK_LAYER_PATH=$(VULKAN_SDK_PATH)/etc/explicit_layer.d ./$<

clean:
	rm -f $(NAME)
