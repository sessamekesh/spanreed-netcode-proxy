import { buildCircleVertexBufferData } from './geomtry';
import { createDynamicVertexBuffer, createStaticVertexBuffer } from './glutils';
import { DefaultLogCb, LogCb, LogLevel, WrapLogFn } from './log';
import { SolidShader } from './solid.shader';

const CIRCLE_SEGMENT_COUNT = 20;
const MAX_INSTANCE_COUNT = 200;

export class DemoRenderer {
  private raf: ReturnType<(typeof globalThis)['requestAnimationFrame']> | null =
    null;
  private constructor(
    private readonly solidShader: SolidShader,
    private readonly vao: WebGLVertexArrayObject,
    private readonly gpuInstanceBuffer: WebGLBuffer,
    private readonly cpuInstanceBuffer: ArrayBuffer,
    private readonly log: LogCb,
    private instanceCt: number
  ) { }

  static Create(
    gl: WebGL2RenderingContext,
    logFn: LogCb = DefaultLogCb
  ): DemoRenderer | null {
    const log = WrapLogFn('DemoRenderer', logFn);
    const circleGeo = buildCircleVertexBufferData(CIRCLE_SEGMENT_COUNT);
    const solidShader = SolidShader.Create(gl, log);

    if (!solidShader) {
      log(
        'Failed to create solid shader, cannot create DemoRenderer',
        LogLevel.Error
      );
      return null;
    }

    const circleBuffer = createStaticVertexBuffer(gl, circleGeo);
    if (!circleBuffer) {
      log(
        'Failed to create dot vertex buffer, cannot create DemoRenderer',
        LogLevel.Error
      );
      return null;
    }

    const instanceBuffer = createDynamicVertexBuffer(
      gl,
      16 * MAX_INSTANCE_COUNT
    );
    if (!instanceBuffer) {
      log(
        'Failed to create instance buffer, cannot create DemoRenderer',
        LogLevel.Error
      );
      return null;
    }

    const vao = solidShader.createVao(gl, circleBuffer, instanceBuffer);
    if (!vao) {
      log(
        'Failed to create dot VAO, cannot create DemoRenderer',
        LogLevel.Error
      );
      return null;
    }

    return new DemoRenderer(
      solidShader,
      vao,
      instanceBuffer,
      new ArrayBuffer(16 * MAX_INSTANCE_COUNT),
      log,
      0
    );
  }

  setInstances(
    gl: WebGL2RenderingContext,
    instances: Array<{
      x: number;
      y: number;
      radius: number;
      color: Uint8Array;
    }>
  ) {
    const toUse = instances.slice(0, MAX_INSTANCE_COUNT);
    for (let i = 0; i < instances.length; i++) {
      const instance = instances[i];

      const f32 = new Float32Array(this.cpuInstanceBuffer, 16 * i);
      f32[0] = instance.x;
      f32[1] = 400 - instance.y;
      f32[2] = instance.radius;

      const u8 = new Uint8Array(this.cpuInstanceBuffer, 16 * i + 12);
      u8[0] = instance.color[0];
      u8[1] = instance.color[1];
      u8[2] = instance.color[2];
    }

    gl.bindBuffer(gl.ARRAY_BUFFER, this.gpuInstanceBuffer);
    gl.bufferSubData(gl.ARRAY_BUFFER, 0, this.cpuInstanceBuffer);
    gl.bindBuffer(gl.ARRAY_BUFFER, null);

    this.instanceCt = toUse.length;
  }

  start(gl: WebGL2RenderingContext) {
    let lastFrame = performance.now();
    const frame = () => {
      const thisFrame = performance.now();
      const dt = (thisFrame - lastFrame) / 1000;
      lastFrame = thisFrame;

      if (gl.canvas instanceof HTMLCanvasElement) {
        gl.canvas.width = gl.canvas.clientWidth * devicePixelRatio;
        gl.canvas.height = gl.canvas.clientHeight * devicePixelRatio;
      } else {
        gl.canvas.width = 400;
        gl.canvas.height = 400;
      }

      gl.clearColor(0.2, 0.333, 0.4666, 1);
      gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
      gl.viewport(0, 0, gl.canvas.width, gl.canvas.height);

      if (this.instanceCt > 0) {
        this.solidShader.draw(
          gl,
          this.vao,
          CIRCLE_SEGMENT_COUNT * 3,
          this.instanceCt,
          400, 400
        );
      }

      this.raf = requestAnimationFrame(frame);
    };
    this.raf = requestAnimationFrame(frame);
  }

  destroy() {
    if (this.raf !== null) {
      cancelAnimationFrame(this.raf);
    }
  }
}
