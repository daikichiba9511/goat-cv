import { Circle, Line } from "react-konva";
import type { NormalizedPoint } from "../../types";

type Props = {
  points: NormalizedPoint[];
  cursor: NormalizedPoint | null;
  imageDisplayWidth: number;
  imageDisplayHeight: number;
  stageScale: number;
  onComplete: () => void;
};

// PolygonDraftOverlayは保存前の輪郭previewと確定用の始点を描画する。
export default function PolygonDraftOverlay({
  points,
  cursor,
  imageDisplayWidth,
  imageDisplayHeight,
  stageScale,
  onComplete,
}: Props) {
  if (points.length === 0) return null;

  const displayPoints = points.flatMap((point) => [
    point.x * imageDisplayWidth,
    point.y * imageDisplayHeight,
  ]);
  const previewPoints = cursor
    ? [
        ...displayPoints,
        cursor.x * imageDisplayWidth,
        cursor.y * imageDisplayHeight,
      ]
    : displayPoints;

  return (
    <>
      <Line
        points={previewPoints}
        closed={points.length >= 2 && cursor !== null}
        stroke="#2563EB"
        strokeWidth={2 / stageScale}
        fill={points.length >= 2 && cursor ? "#2563EB20" : undefined}
        dash={[6 / stageScale, 4 / stageScale]}
        lineJoin="round"
        listening={false}
      />
      {points.map((point, pointIndex) => {
        const canClosePolygon = pointIndex === 0 && points.length >= 3;
        return (
          <Circle
            key={`${point.x}:${point.y}`}
            x={point.x * imageDisplayWidth}
            y={point.y * imageDisplayHeight}
            radius={(canClosePolygon ? 6 : 4) / stageScale}
            fill={canClosePolygon ? "#16A34A" : "#FFFFFF"}
            stroke={canClosePolygon ? "#166534" : "#2563EB"}
            strokeWidth={2 / stageScale}
            listening={canClosePolygon}
            onClick={(event) => {
              event.cancelBubble = true;
              onComplete();
            }}
          />
        );
      })}
    </>
  );
}
