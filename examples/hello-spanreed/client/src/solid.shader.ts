import { DefaultLogCb, LogCb, LogLevel } from './log';
import { createProgram, createStaticVertexBuffer } from './glutils';

const vsSrc = `#version 300 es
precision mediump float;

in vec2 vertexPosition;
in vec2 shapeLocation;
in float shapeRadius;
in vec3 shapeColor;

out vec3 fragmentColor;

uniform vec2 canvasSize;

void main() {
  fragmentColor = shapeColor;

  vec2 logicalPosition = vertexPosition * shapeRadius + shapeLocation;
  gl_Position = vec4((logicalPosition / canvasSize) * 2. - 1., 0., 1.);
}`;

const fsSrc = `#version 300 es
precision mediump float;

in vec3 fragmentColor;
out vec4 outColor;

void main() {
  outColor = vec4(fragmentColor, 1.);
}`;

interface Attrib {
  VertexPosition: number;
  ShapeLocation: number;
  ShapeRadius: number;
  ShapeColor: number;
}

interface Uniform {
  CanvasSize: WebGLUniformLocation;
}

export class SolidShader {
  private constructor(
    private readonly program: WebGLProgram,
    private readonly attribs: Attrib,
    private readonly uniforms: Uniform
  ) {}

  static Create(
    gl: WebGL2RenderingContext,
    log: LogCb = DefaultLogCb
  ): SolidShader | null {
    const program = createProgram(gl, vsSrc, fsSrc, log);
    if (!program) return null;

    const VertexPosition = gl.getAttribLocation(program, 'vertexPosition');
    const ShapeLocation = gl.getAttribLocation(program, 'shapeLocation');
    const ShapeRadius = gl.getAttribLocation(program, 'shapeRadius');
    const ShapeColor = gl.getAttribLocation(program, 'shapeColor');

    const attribs: Attrib = {
      ShapeColor,
      ShapeLocation,
      ShapeRadius,
      VertexPosition,
    };

    const CanvasSize = gl.getUniformLocation(program, 'canvasSize');
    if (!CanvasSize) {
      log('SolidShader - failed to find CanvasSize uniform', LogLevel.Error);
      return null;
    }

    const uniforms: Uniform = {
      CanvasSize,
    };

    return new SolidShader(program, attribs, uniforms);
  }

  createVao(
    gl: WebGL2RenderingContext,
    geoBuffer: WebGLBuffer,
    instanceBuffer: WebGLBuffer,
    log: LogCb = DefaultLogCb
  ) {
    const vao = gl.createVertexArray();
    if (!vao) {
      log('Failed to allocate VAO for SolidShader', LogLevel.Error);
      return null;
    }

    gl.bindVertexArray(vao);

    gl.enableVertexAttribArray(this.attribs.VertexPosition);
    gl.enableVertexAttribArray(this.attribs.ShapeColor);
    gl.enableVertexAttribArray(this.attribs.ShapeLocation);
    gl.enableVertexAttribArray(this.attribs.ShapeRadius);

    gl.bindBuffer(gl.ARRAY_BUFFER, geoBuffer);
    gl.vertexAttribPointer(
      this.attribs.VertexPosition,
      2,
      gl.FLOAT,
      false,
      0,
      0
    );

    gl.bindBuffer(gl.ARRAY_BUFFER, instanceBuffer);
    gl.vertexAttribPointer(
      this.attribs.ShapeLocation,
      2,
      gl.FLOAT,
      false,
      16,
      0
    );
    gl.vertexAttribPointer(this.attribs.ShapeRadius, 1, gl.FLOAT, false, 16, 8);
    gl.vertexAttribPointer(
      this.attribs.ShapeColor,
      3,
      gl.UNSIGNED_BYTE,
      true,
      16,
      12
    );
    gl.vertexAttribDivisor(this.attribs.ShapeLocation, 1);
    gl.vertexAttribDivisor(this.attribs.ShapeRadius, 1);
    gl.vertexAttribDivisor(this.attribs.ShapeColor, 1);
    gl.bindVertexArray(null);

    return vao;
  }

  draw(
    gl: WebGL2RenderingContext,
    vao: WebGLVertexArrayObject,
    vertexCount: number,
    instanceCount: number,
    canvasWidth: number,
    canvasHeight: number
  ) {
    gl.useProgram(this.program);
    gl.bindVertexArray(vao);

    gl.uniform2f(this.uniforms.CanvasSize, canvasWidth, canvasHeight);
    gl.drawArraysInstanced(gl.TRIANGLES, 0, vertexCount, instanceCount);

    gl.bindVertexArray(null);
    gl.useProgram(null);
  }
}
