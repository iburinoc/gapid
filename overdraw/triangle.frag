#version 450
#extension GL_ARB_separate_shader_objects : enable

layout(location = 0) in vec3 fragColor;
layout(location = 0) out vec4 outColor;

layout(binding = 1) buffer written
{
	uint values[];
}

void main() {
	outColor = vec4(fragColor, 1.0);
	pos = ivec2(gl_Fragcoord.xy);
	atomicAdd(values[pos.x + 800 * pos.y], 1);
}
