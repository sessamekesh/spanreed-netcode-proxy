import { DefaultLogCb, LogCb, LogLevel } from './log';

export function createStaticVertexBuffer(
  gl: WebGL2RenderingContext,
  data: ArrayBuffer
) {
  const buffer = gl.createBuffer();
  if (!buffer) {
    return null;
  }

  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, data, gl.STATIC_DRAW);
  gl.bindBuffer(gl.ARRAY_BUFFER, null);

  return buffer;
}

export function createDynamicVertexBuffer(
  gl: WebGL2RenderingContext,
  size: number
) {
  const buffer = gl.createBuffer();

  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, size, gl.DYNAMIC_DRAW);
  gl.bindBuffer(gl.ARRAY_BUFFER, null);

  return buffer;
}

export function createProgram(
  gl: WebGL2RenderingContext,
  vertexShaderSource: string,
  fragmentShaderSource: string,
  log: LogCb = DefaultLogCb
) {
  const vertexShader = gl.createShader(gl.VERTEX_SHADER);
  const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
  const program = gl.createProgram();

  if (!vertexShader || !fragmentShader || !program) {
    log(
      `Failed to allocate GL objects (` +
        `vs=${!!vertexShader}, ` +
        `fs=${!!fragmentShader}, ` +
        `program=${!!program})`,
      LogLevel.Error
    );
    return null;
  }

  gl.shaderSource(vertexShader, vertexShaderSource);
  gl.compileShader(vertexShader);
  if (!gl.getShaderParameter(vertexShader, gl.COMPILE_STATUS)) {
    const errorMessage = gl.getShaderInfoLog(vertexShader);
    log(`Failed to compile vertex shader: ${errorMessage}`, LogLevel.Error);
    return null;
  }

  gl.shaderSource(fragmentShader, fragmentShaderSource);
  gl.compileShader(fragmentShader);
  if (!gl.getShaderParameter(fragmentShader, gl.COMPILE_STATUS)) {
    const errorMessage = gl.getShaderInfoLog(fragmentShader);
    LogLevel.Error;
    log(`Failed to compile fragment shader: ${errorMessage}`, LogLevel.Error);
    return null;
  }

  gl.attachShader(program, vertexShader);
  gl.attachShader(program, fragmentShader);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    const errorMessage = gl.getProgramInfoLog(program);
    log(`Failed to link GPU program: ${errorMessage}`, LogLevel.Error);
    return null;
  }

  return program;
}
