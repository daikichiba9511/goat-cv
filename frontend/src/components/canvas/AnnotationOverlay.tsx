import { Fragment, memo, useMemo } from "react";
import {
  Arrow,
  Label as KonvaLabel,
  Rect,
  Tag,
  Text,
} from "react-konva";
import type {
  Annotation,
  BBoxCoordinates,
  Edge,
  LabelDefinition,
  Tool,
} from "../../types";
import {
  type DisplayBBox,
  readableTextColor,
  readingOrderEdgePoints,
  toDisplayBBox,
} from "./annotationGeometry";

type Props = {
  annotations: Annotation[];
  edges: Edge[];
  labels: LabelDefinition[];
  selectedAnnotationId: string | null;
  selectedEdgeId: string | null;
  edgeSourceId: string | null;
  activeTool: Tool;
  imageWidth: number;
  imageHeight: number;
  scale: number;
  onAnnotationClick: (annotationId: string) => void;
  onEdgeClick: (edgeId: string) => void;
  onCoordinatesChange: (annotationId: string, coordinates: BBoxCoordinates) => void;
  layerToNormalized: (x: number, y: number) => [number, number];
};

function AnnotationOverlay({
  annotations,
  edges,
  labels,
  selectedAnnotationId,
  selectedEdgeId,
  edgeSourceId,
  activeTool,
  imageWidth,
  imageHeight,
  scale,
  onAnnotationClick,
  onEdgeClick,
  onCoordinatesChange,
  layerToNormalized,
}: Props) {
  const labelById = useMemo(
    () => new Map(labels.map((label) => [label.id, label])),
    [labels],
  );
  const displayBBoxByAnnotationId = useMemo(() => {
    const boxes = new Map<string, DisplayBBox>();
    for (const annotation of annotations) {
      if (annotation.type !== "bbox") continue;
      boxes.set(
        annotation.id,
        toDisplayBBox(
          annotation.coordinates as BBoxCoordinates,
          imageWidth,
          imageHeight,
          scale,
        ),
      );
    }
    return boxes;
  }, [annotations, imageWidth, imageHeight, scale]);

  return (
    <>
      {edges.map((edge) => {
        if (edge.type !== "reading_order") return null;
        const sourceBox = displayBBoxByAnnotationId.get(edge.source_annotation_id);
        const targetBox = displayBBoxByAnnotationId.get(edge.target_annotation_id);
        if (!sourceBox || !targetBox) return null;

        const isSelected = edge.id === selectedEdgeId;
        return (
          <Arrow
            key={edge.id}
            points={readingOrderEdgePoints(sourceBox, targetBox)}
            stroke={isSelected ? "#2563EB" : "#7C3AED"}
            fill={isSelected ? "#2563EB" : "#7C3AED"}
            strokeWidth={isSelected ? 3 : 2}
            pointerLength={8}
            pointerWidth={8}
            hitStrokeWidth={14}
            opacity={0.9}
            onClick={(event) => {
              event.cancelBubble = true;
              onEdgeClick(edge.id);
            }}
          />
        );
      })}

      {annotations.map((annotation) => {
        if (annotation.type !== "bbox") return null;
        const coordinates = annotation.coordinates as BBoxCoordinates;
        const displayBox = displayBBoxByAnnotationId.get(annotation.id);
        if (!displayBox) return null;

        const label = annotation.label_id
          ? labelById.get(annotation.label_id)
          : undefined;
        const color = label?.color ?? "#64748B";
        const labelName = annotation.label_id
          ? label?.name ?? "Unknown label"
          : "No label";
        const isSelected = annotation.id === selectedAnnotationId;
        const isEdgeSource = annotation.id === edgeSourceId;
        const labelY = displayBox.y >= 22 ? displayBox.y - 22 : displayBox.y + 4;

        return (
          <Fragment key={annotation.id}>
            <Rect
              id={annotation.id}
              x={displayBox.x}
              y={displayBox.y}
              width={displayBox.width}
              height={displayBox.height}
              stroke={color}
              strokeWidth={isSelected || isEdgeSource ? 3 : 2}
              fill={`${color}20`}
              dash={isEdgeSource && activeTool === "edge" ? [6, 3] : undefined}
              shadowColor={isSelected || isEdgeSource ? color : undefined}
              shadowBlur={isSelected || isEdgeSource ? 8 : 0}
              draggable={activeTool === "select"}
              onClick={(event) => {
                event.cancelBubble = true;
                onAnnotationClick(annotation.id);
              }}
              onDragEnd={(event) => {
                const node = event.target;
                const [x, y] = layerToNormalized(node.x(), node.y());
                onCoordinatesChange(annotation.id, { ...coordinates, x, y });
              }}
              onTransformEnd={(event) => {
                const node = event.target;
                const scaleX = node.scaleX();
                const scaleY = node.scaleY();
                // Why: Konva Transformerはwidth/heightではなくscaleを変えるため、保存前に実寸へ畳み込む。
                node.scaleX(1);
                node.scaleY(1);
                const [x, y] = layerToNormalized(node.x(), node.y());
                const [width] = layerToNormalized(node.width() * scaleX, 0);
                const [, height] = layerToNormalized(0, node.height() * scaleY);
                onCoordinatesChange(annotation.id, { x, y, width, height });
              }}
            />
            <KonvaLabel
              x={displayBox.x}
              y={labelY}
              listening={false}
              opacity={isSelected || isEdgeSource ? 1 : 0.92}
            >
              <Tag
                fill={color}
                cornerRadius={3}
                stroke={isSelected || isEdgeSource ? "#111827" : color}
                strokeWidth={isSelected || isEdgeSource ? 1 : 0}
              />
              <Text
                text={labelName}
                fontSize={12}
                fontStyle="bold"
                lineHeight={1}
                padding={5}
                fill={readableTextColor(color)}
              />
            </KonvaLabel>
          </Fragment>
        );
      })}
    </>
  );
}

export default memo(AnnotationOverlay);
