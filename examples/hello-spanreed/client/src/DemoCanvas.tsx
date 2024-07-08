import React, { useEffect, useRef, useState } from 'react';
import { DemoRenderer } from './DemoRenderer';
import { COLORS } from './colors';

export interface DemoCanvasProps {}

export const DemoCanvas: React.FC = () => {
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

    const renderer = DemoRenderer.Create(gl);
    if (!renderer) {
      return;
    }

    renderer.start(gl);

    renderer.setInstances(gl, [
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
    ]);

    setRenderer(renderer);

    return () => {
      renderer.destroy();
    };
  }, [gl]);

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
    >
      HTML5 Canvas not supported!
    </canvas>
  );
};
