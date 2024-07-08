export function buildCircleVertexBufferData(segmentCount: number) {
  const vertexData = [];

  for (let i = 0; i < segmentCount; i++) {
    const vertex1Angle = (i * Math.PI * 2) / segmentCount;
    const vertex2Angle = ((i + 1) * Math.PI * 2) / segmentCount;

    const x1 = Math.cos(vertex1Angle);
    const y1 = Math.sin(vertex1Angle);
    const x2 = Math.cos(vertex2Angle);
    const y2 = Math.sin(vertex2Angle);

    // Center vertex is a light blue color, and in the middle of the shape
    vertexData.push(0, 0);

    // Other two vertices are along the edges of the circle, and are a darker blue shape
    vertexData.push(x1, y1);
    vertexData.push(x2, y2);
  }

  return new Float32Array(vertexData);
}
