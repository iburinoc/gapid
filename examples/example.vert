#version 450
#extension GL_ARB_separate_shader_objects : enable

out gl_PerVertex {
	vec4 gl_Position;
};

layout(set = 0, binding = 4) uniform Val0 {
	mat4 val0;
} val0;

layout(set = 1, binding = 32) uniform Val1 {
	vec3 val1;
} val1;

layout(set = 2, binding = 2) uniform sampler2D val2;

void main() {
	gl_Position = val0.val0 * vec4(0., 0., 0., 1.);
}
