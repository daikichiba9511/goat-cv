import { Circle, Line } from "react-konva";
import type Konva from "konva";
import type { NormalizedPoint } from "../../types";
import type { DisplayPolygon } from "./annotationGeometry";

type Props = {
  annotationId: string;
  displayPolygon: DisplayPolygon;
  color: string;
  selected: boolean;
  edgeSource: boolean;
  editable: boolean;
  listening: boolean;
  stageScale: number;
  imageDisplayWidth: number;
  imageDisplayHeight: number;
  onClick: (annotationId: string) => void;
  onPointChange: (
    annotationId: string,
    pointIndex: number,
    point: NormalizedPoint,
  ) => void;
};

export default function PolygonAnnotationShape({
  annotationId,
  displayPolygon,
  color,
  selected,
  edgeSource,
  editable,
  listening,
  stageScale,
  imageDisplayWidth,
  imageDisplayHeight,
  onClick,
  onPointChange,
}: Props) {
  const moveRenderedPoint = (
    event: Konva.KonvaEventObject<DragEvent>,
    pointIndex: number,
  ) => {
    const pointHandle = event.target;
    const x = Math.max(0, Math.min(imageDisplayWidth, pointHandle.x()));
    const y = Math.max(0, Math.min(imageDisplayHeight, pointHandle.y()));
    pointHandle.position({ x, y });

    // Why: drag中はKonva nodeだけを更新し、Storeのrevision更新はdragEndの1回に抑える。
    const polygonLine = pointHandle.getLayer()?.findOne(
      `#${CSS.escape(annotationId)}`,
    ) as Konva.Line | undefined;
    if (!polygonLine) return;
    const points = polygonLine.points().slice();
    points[pointIndex * 2] = x;
    points[pointIndex * 2 + 1] = y;
    polygonLine.points(points);
    polygonLine.getLayer()?.batchDraw();
  };

  return (
    <>
      <Line
        id={annotationId}
        points={displayPolygon.points}
        closed
        stroke={color}
        strokeWidth={selected || edgeSource ? 3 : 2}
        fill={`${color}20`}
        lineJoin="round"
        dash={edgeSource ? [6, 3] : undefined}
        shadowColor={selected || edgeSource ? color : undefined}
        shadowBlur={selected || edgeSource ? 8 : 0}
        hitStrokeWidth={12}
        listening={listening}
        onClick={(event) => {
          event.cancelBubble = true;
          onClick(annotationId);
        }}
      />
      {selected && editable && Array.from(
        { length: displayPolygon.points.length / 2 },
        (_, pointIndex) => {
          const valueIndex = pointIndex * 2;
          return (
            <Circle
              key={pointIndex}
              x={displayPolygon.points[valueIndex]}
              y={displayPolygon.points[valueIndex + 1]}
              radius={5 / stageScale}
              fill="#FFFFFF"
              stroke={color}
              strokeWidth={2 / stageScale}
              draggable
              onDragMove={(event) => moveRenderedPoint(event, pointIndex)}
              onDragEnd={(event) => {
                moveRenderedPoint(event, pointIndex);
                onPointChange(annotationId, pointIndex, {
                  x: event.target.x() / imageDisplayWidth,
                  y: event.target.y() / imageDisplayHeight,
                });
              }}
            />
          );
        },
      )}
    </>
  );
}
