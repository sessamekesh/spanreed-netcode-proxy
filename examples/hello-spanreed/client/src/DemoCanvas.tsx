import React, { useEffect, useRef, useState } from 'react';
import { DemoRenderer } from './DemoRenderer';
import { COLORS } from './colors';
import { DefaultLogCb, LogCb, LogLevel, WrapLogFn } from './log';

export interface DemoCanvasProps {
  logFn?: LogCb;
  dots?: Array<{
    x: number;
    y: number;
    radius: number;
    color: Uint8Array;
  }>,
  onClick?: (x: number, y: number) => void;
}

const DEFAULT_DOT_STATE = [
  {
    color: COLORS.Red,
    radius: 20,
    x: 80,
    y: 200,
  },
  {
    color: COLORS.Green,
    radius: 20,
    x: 140,
    y: 200,
  },
  {
    color: COLORS.Blue,
    radius: 20,
    x: 200,
    y: 200,
  },
  {
    color: COLORS.Indigo,
    radius: 20,
    x: 260,
    y: 200,
  },
  {
    color: COLORS.FireOrange,
    radius: 20,
    x: 320,
    y: 200,
  },
];

export const DemoCanvas: React.FC<DemoCanvasProps> = ({ logFn, dots, onClick }) => {
  const log = WrapLogFn('DemoCanvas', logFn ?? DefaultLogCb);
  const ref = useRef<HTMLCanvasElement>(null);

  const [gl, setGl] = useState<WebGL2RenderingContext>();
  const [renderer, setRenderer] = useState<DemoRenderer>();

  useEffect(() => {
    if (ref.current === null) {
      return;
    }

    const gl = ref.current.getContext('webgl2');
    if (gl !== null) {
      setGl(gl);
    }

    return () => setGl(undefined);
  }, [ref]);

  useEffect(() => {
    if (!gl) return;

    log('Creating new renderer!');
    const renderer = DemoRenderer.Create(gl, logFn);
    if (!renderer) {
      return;
    }

    renderer.start(gl);

    renderer.setInstances(gl, DEFAULT_DOT_STATE);

    setRenderer(renderer);

    return () => {
      renderer.destroy();
    };
  }, [gl]);

  useEffect(() => {
    if (gl === undefined) return;
    if (renderer === undefined) return;

    renderer.setInstances(gl, dots ?? DEFAULT_DOT_STATE);
  }, [gl, renderer, dots]);

  return (
    <canvas
      style={{
        backgroundColor: '#357',
        width: '400px',
        height: '400px',
        margin: '12px',
        alignSelf: 'center',
      }}
      ref={ref}
      onClick={(evt) => {
        if (!(evt.target instanceof HTMLCanvasElement)) return;

        const rect = evt.target.getBoundingClientRect();
        onClick?.((evt.clientX - rect.left) / rect.width * 400, (evt.clientY - rect.top) / rect.height * 400);
      }}
    >
      HTML5 Canvas not supported!
    </canvas>
  );
};
