import type { BBoxCoordinates } from "../../types";

export type DisplayBBox = {
  x: number;
  y: number;
  width: number;
  height: number;
  centerX: number;
  centerY: number;
};

export function toDisplayBBox(
  coordinates: BBoxCoordinates,
  imageWidth: number,
  imageHeight: number,
  scale: number,
): DisplayBBox {
  const x = coordinates.x * imageWidth * scale;
  const y = coordinates.y * imageHeight * scale;
  const width = coordinates.width * imageWidth * scale;
  const height = coordinates.height * imageHeight * scale;
  return {
    x,
    y,
    width,
    height,
    centerX: x + width / 2,
    centerY: y + height / 2,
  };
}

export function normalizeBBox(coordinates: BBoxCoordinates): BBoxCoordinates {
  return {
    x: coordinates.width < 0 ? coordinates.x + coordinates.width : coordinates.x,
    y: coordinates.height < 0 ? coordinates.y + coordinates.height : coordinates.y,
    width: Math.abs(coordinates.width),
    height: Math.abs(coordinates.height),
  };
}

export function readingOrderEdgePoints(
  sourceBox: DisplayBBox,
  targetBox: DisplayBBox,
): [number, number, number, number] {
  const deltaX = targetBox.centerX - sourceBox.centerX;
  const deltaY = targetBox.centerY - sourceBox.centerY;
  if (deltaX === 0 && deltaY === 0) {
    return [sourceBox.centerX, sourceBox.centerY, targetBox.centerX, targetBox.centerY];
  }

  const sourceScale = edgeEndpointScale(deltaX, deltaY, sourceBox.width, sourceBox.height);
  const targetScale = edgeEndpointScale(deltaX, deltaY, targetBox.width, targetBox.height);
  return [
    sourceBox.centerX + deltaX * sourceScale,
    sourceBox.centerY + deltaY * sourceScale,
    targetBox.centerX - deltaX * targetScale,
    targetBox.centerY - deltaY * targetScale,
  ];
}

export function readableTextColor(backgroundColor: string): string {
  const hex = backgroundColor.replace("#", "");
  if (hex.length !== 6) return "#FFFFFF";

  const red = parseInt(hex.slice(0, 2), 16);
  const green = parseInt(hex.slice(2, 4), 16);
  const blue = parseInt(hex.slice(4, 6), 16);
  const luminance = (red * 299 + green * 587 + blue * 114) / 1000;
  return luminance > 160 ? "#111827" : "#FFFFFF";
}

function edgeEndpointScale(
  deltaX: number,
  deltaY: number,
  width: number,
  height: number,
): number {
  const widthScale = deltaX === 0
    ? Number.POSITIVE_INFINITY
    : Math.abs((width / 2) / deltaX);
  const heightScale = deltaY === 0
    ? Number.POSITIVE_INFINITY
    : Math.abs((height / 2) / deltaY);
  return Math.min(widthScale, heightScale);
}
