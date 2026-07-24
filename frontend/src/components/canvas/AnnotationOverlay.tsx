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
  NormalizedPoint,
  PolygonCoordinates,
  Tool,
} from "../../types";
import { categoryLabel, EDGE_RELATIONS } from "../../edgeRelations";
import {
  type DisplayBBox,
  directedEdgePoints,
  readableTextColor,
  toDisplayBBox,
  toDisplayPolygon,
} from "./annotationGeometry";
import PolygonAnnotationShape from "./PolygonAnnotationShape";

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
  stageScale: number;
  compactLabels: boolean;
  onAnnotationClick: (annotationId: string) => void;
  onEdgeClick: (edgeId: string) => void;
  onBBoxCoordinatesChange: (
    annotationId: string,
    coordinates: BBoxCoordinates,
  ) => void;
  onPolygonPointChange: (
    annotationId: string,
    pointIndex: number,
    point: NormalizedPoint,
  ) => void;
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
  stageScale,
  compactLabels,
  onAnnotationClick,
  onEdgeClick,
  onBBoxCoordinatesChange,
  onPolygonPointChange,
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
  const displayPolygonByAnnotationId = useMemo(() => {
    const polygons = new Map<string, ReturnType<typeof toDisplayPolygon>>();
    for (const annotation of annotations) {
      if (annotation.type !== "polygon") continue;
      polygons.set(
        annotation.id,
        toDisplayPolygon(
          annotation.coordinates as PolygonCoordinates,
          imageWidth,
          imageHeight,
          scale,
        ),
      );
    }
    return polygons;
  }, [annotations, imageWidth, imageHeight, scale]);
  const displayBoundsByAnnotationId = useMemo(() => {
    const bounds = new Map(displayBBoxByAnnotationId);
    for (const [annotationId, polygon] of displayPolygonByAnnotationId) {
      bounds.set(annotationId, polygon.bounds);
    }
    return bounds;
  }, [displayBBoxByAnnotationId, displayPolygonByAnnotationId]);
  const displayedEdges = useMemo(() => {
    return edges.flatMap((edge) => {
      const sourceBox = displayBoundsByAnnotationId.get(edge.source_annotation_id);
      const targetBox = displayBoundsByAnnotationId.get(edge.target_annotation_id);
      if (!sourceBox || !targetBox) return [];

      const points = directedEdgePoints(sourceBox, targetBox);
      return [{
        edge,
        points,
        relation: EDGE_RELATIONS[edge.type],
        labelX: (points[0] + points[2]) / 2,
        labelY: (points[1] + points[3]) / 2,
      }];
    });
  }, [edges, displayBoundsByAnnotationId]);
  const annotationShapesListen = activeTool !== "bbox" && activeTool !== "polygon";

  return (
    <>
      {displayedEdges.map(({ edge, points, relation }) => {
        const isSelected = edge.id === selectedEdgeId;
        return (
          <Arrow
            key={edge.id}
            points={points}
            stroke={relation.color}
            fill={relation.color}
            strokeWidth={isSelected ? 4 : 2}
            dash={relation.dash}
            pointerLength={8}
            pointerWidth={8}
            hitStrokeWidth={14}
            opacity={isSelected ? 1 : 0.9}
            shadowColor={isSelected ? relation.color : undefined}
            shadowBlur={isSelected ? 8 : 0}
            listening={annotationShapesListen}
            onClick={(event) => {
              event.cancelBubble = true;
              onEdgeClick(edge.id);
            }}
          />
        );
      })}

      {annotations.map((annotation) => {
        const displayBox = displayBoundsByAnnotationId.get(annotation.id);
        const displayPolygon = displayPolygonByAnnotationId.get(annotation.id);
        if (!displayBox || (annotation.type === "polygon" && !displayPolygon)) return null;

        const label = annotation.label_id
          ? labelById.get(annotation.label_id)
          : undefined;
        const color = label?.color ?? "#64748B";
        const labelName = annotation.label_id
          ? label?.name ?? "Unknown label"
          : "No label";
        const displayLabel = label
          ? compactLabels && activeTool === "edge"
            ? categoryLabel(label.category)
            : `${labelName} / ${categoryLabel(label.category)}`
          : compactLabels && activeTool === "edge"
            ? "None"
            : labelName;
        const isSelected = annotation.id === selectedAnnotationId;
        const isEdgeSource = annotation.id === edgeSourceId;
        const labelY = displayBox.y >= 22 ? displayBox.y - 22 : displayBox.y + 4;

        return (
          <Fragment key={annotation.id}>
            {annotation.type === "bbox" ? (
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
                listening={annotationShapesListen}
                draggable={activeTool === "select"}
                onClick={(event) => {
                  event.cancelBubble = true;
                  onAnnotationClick(annotation.id);
                }}
                onDragEnd={(event) => {
                  const coordinates = annotation.coordinates as BBoxCoordinates;
                  const node = event.target;
                  const [x, y] = layerToNormalized(node.x(), node.y());
                  onBBoxCoordinatesChange(annotation.id, { ...coordinates, x, y });
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
                  onBBoxCoordinatesChange(annotation.id, { x, y, width, height });
                }}
              />
            ) : (
              displayPolygon && (
                <PolygonAnnotationShape
                  annotationId={annotation.id}
                  displayPolygon={displayPolygon}
                  color={color}
                  selected={isSelected}
                  edgeSource={isEdgeSource && activeTool === "edge"}
                  editable={activeTool === "select"}
                  listening={annotationShapesListen}
                  stageScale={stageScale}
                  imageDisplayWidth={imageWidth * scale}
                  imageDisplayHeight={imageHeight * scale}
                  onClick={onAnnotationClick}
                  onPointChange={onPolygonPointChange}
                />
              )
            )}
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
                text={displayLabel}
                width={compactLabels ? Math.max(46, displayBox.width) : undefined}
                ellipsis={compactLabels}
                wrap={compactLabels ? "none" : undefined}
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

      {displayedEdges.map(({ edge, relation, labelX, labelY }) => {
        const isSelected = edge.id === selectedEdgeId;
        // Why: Table内にあるCellへの線も、前面の短いrelation名から選択できるようにする。
        return (
          <KonvaLabel
            key={`${edge.id}-label`}
            x={labelX}
            y={labelY}
            offsetX={(relation.canvasLabel.length * 6 + 8) / 2}
            offsetY={compactLabels ? -4 : 10}
            opacity={isSelected ? 1 : 0.92}
            listening={annotationShapesListen}
            onClick={(event) => {
              event.cancelBubble = true;
              onEdgeClick(edge.id);
            }}
          >
            <Tag
              fill="#FFFFFF"
              stroke={relation.color}
              strokeWidth={isSelected ? 2 : 1}
              cornerRadius={3}
            />
            <Text
              text={relation.canvasLabel}
              padding={4}
              fontSize={10}
              fontStyle="bold"
              fill={relation.color}
            />
          </KonvaLabel>
        );
      })}
    </>
  );
}

export default memo(AnnotationOverlay);
