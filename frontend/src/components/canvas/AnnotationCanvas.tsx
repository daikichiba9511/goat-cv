import { Fragment, useMemo, useRef, useEffect, useState, useCallback } from "react";
import {
  Arrow,
  Stage,
  Layer,
  Image as KonvaImage,
  Label as KonvaLabel,
  Rect,
  Tag,
  Text,
  Transformer,
} from "react-konva";
import type Konva from "konva";
import type { ImageMeta, BBoxCoordinates, LabelDefinition } from "../../types";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import { imageFileUrl } from "../../api/client";
import type { Tool } from "../../pages/Annotator";

type Props = {
  image: ImageMeta;
  activeTool: Tool;
  activeLabel: string | null;
};

type DisplayBBox = {
  x: number;
  y: number;
  width: number;
  height: number;
  centerX: number;
  centerY: number;
};

function edgeEndpointScale(deltaX: number, deltaY: number, width: number, height: number): number {
  const widthScale = deltaX === 0 ? Number.POSITIVE_INFINITY : Math.abs((width / 2) / deltaX);
  const heightScale = deltaY === 0 ? Number.POSITIVE_INFINITY : Math.abs((height / 2) / deltaY);
  return Math.min(widthScale, heightScale);
}

export default function AnnotationCanvas({ image, activeTool, activeLabel }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<Konva.Stage>(null);
  const transformerRef = useRef<Konva.Transformer>(null);
  const [img, setImg] = useState<HTMLImageElement | null>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });
  const [drawing, setDrawing] = useState<BBoxCoordinates | null>(null);
  const [stageScale, setStageScale] = useState(1);
  const [stagePos, setStagePos] = useState({ x: 0, y: 0 });

  const {
    annotations,
    edges,
    selectedId,
    selectedEdgeId,
    edgeSourceId,
    select,
    selectEdge,
    setEdgeSource,
    addReadingOrderEdge,
    addBBox,
    updateCoordinates,
    remove,
    removeEdge,
  } = useAnnotationStore();
  const { labels } = useProjectStore();

  // Why: KonvaはHTMLImageElementを描画元にするため、API URLをReact state上の画像要素へ変換する。
  useEffect(() => {
    const imageElement = new window.Image();
    imageElement.src = imageFileUrl(image.id);
    imageElement.onload = () => setImg(imageElement);
    return () => { imageElement.onload = null; };
  }, [image.id]);

  // Why: Canvasサイズはレイアウトに従わせ、画像の保存座標を画面ピクセルへ依存させない。
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const resizeObserver = new ResizeObserver((entries) => {
      const { width, height } = entries[0].contentRect;
      setStageSize({ width, height });
    });
    resizeObserver.observe(container);
    return () => resizeObserver.disconnect();
  }, []);

  // Why: 初期表示は全体が見えるfitにし、手動Zoom/PanはStage変換として別に持つ。
  const scale = img && stageSize.width > 0
    ? Math.min(stageSize.width / image.width, stageSize.height / image.height)
    : 1;

  // Why: Transformerは選択ツール中だけ表示し、BBox作成やPan操作とハンドル操作が競合しないようにする。
  useEffect(() => {
    const transformer = transformerRef.current;
    const stage = stageRef.current;
    if (!transformer || !stage) return;

    if (selectedId && activeTool === "select") {
      const node = stage.findOne(`#${CSS.escape(selectedId)}`);
      if (node) {
        transformer.nodes([node]);
        transformer.getLayer()?.batchDraw();
        return;
      }
    }
    transformer.nodes([]);
    transformer.getLayer()?.batchDraw();
  }, [selectedId, activeTool, annotations]);

  // Why: 保存座標は正規化値なので、Zoom/Pan後の画面座標を画像座標へ戻してから保存する。
  const screenToNormalized = useCallback(
    (screenX: number, screenY: number): [number, number] => {
      const imageX = (screenX - stagePos.x) / stageScale;
      const imageY = (screenY - stagePos.y) / stageScale;
      return [imageX / (image.width * scale), imageY / (image.height * scale)];
    },
    [image.width, image.height, scale, stageScale, stagePos],
  );

  // Why: Drag/Transform後のNode座標はLayer座標なので、StageのZoom/Pan補正を二重適用しない。
  const layerToNormalized = useCallback(
    (lx: number, ly: number): [number, number] => {
      return [lx / (image.width * scale), ly / (image.height * scale)];
    },
    [image.width, image.height, scale],
  );

  const bboxDisplayByAnnotationID = useMemo(() => {
    const boxes = new Map<string, DisplayBBox>();
    for (const annotation of annotations) {
      if (annotation.type !== "bbox") continue;
      const coords = annotation.coordinates as BBoxCoordinates;
      const x = coords.x * image.width * scale;
      const y = coords.y * image.height * scale;
      const width = coords.width * image.width * scale;
      const height = coords.height * image.height * scale;
      boxes.set(annotation.id, {
        x,
        y,
        width,
        height,
        centerX: x + width / 2,
        centerY: y + height / 2,
      });
    }
    return boxes;
  }, [annotations, image.width, image.height, scale]);

  const handleWheel = (e: Konva.KonvaEventObject<WheelEvent>) => {
    e.evt.preventDefault();
    const stage = stageRef.current;
    if (!stage) return;

    const pointer = stage.getPointerPosition();
    if (!pointer) return;

    const scaleBy = 1.1;
    const oldScale = stageScale;
    const newScale = e.evt.deltaY < 0 ? oldScale * scaleBy : oldScale / scaleBy;
    const clampedScale = Math.max(0.1, Math.min(10, newScale));

    // Why: ポインタ位置を基準にZoomし、注目しているAnnotationが画面外へ飛ばないようにする。
    const mousePointTo = {
      x: (pointer.x - stagePos.x) / oldScale,
      y: (pointer.y - stagePos.y) / oldScale,
    };

    setStageScale(clampedScale);
    setStagePos({
      x: pointer.x - mousePointTo.x * clampedScale,
      y: pointer.y - mousePointTo.y * clampedScale,
    });
  };

  const handleMouseDown = () => {
    if (activeTool !== "bbox") return;
    const stage = stageRef.current;
    if (!stage) return;
    const pos = stage.getPointerPosition();
    if (!pos) return;

    const [nx, ny] = screenToNormalized(pos.x, pos.y);
    setDrawing({ x: nx, y: ny, width: 0, height: 0 });
    select(null);
  };

  const handleMouseMove = () => {
    if (!drawing || activeTool !== "bbox") return;
    const stage = stageRef.current;
    if (!stage) return;
    const pos = stage.getPointerPosition();
    if (!pos) return;

    const [nx, ny] = screenToNormalized(pos.x, pos.y);
    setDrawing({
      ...drawing,
      width: nx - drawing.x,
      height: ny - drawing.y,
    });
  };

  const handleMouseUp = () => {
    if (!drawing || activeTool !== "bbox") return;
    // Why: annotatorが右下以外へドラッグしても、保存時は正の幅高さを持つBBoxへ正規化する。
    const coords: BBoxCoordinates = {
      x: drawing.width < 0 ? drawing.x + drawing.width : drawing.x,
      y: drawing.height < 0 ? drawing.y + drawing.height : drawing.y,
      width: Math.abs(drawing.width),
      height: Math.abs(drawing.height),
    };
    // Why not: クリックや手ぶれでできる微小BBoxはAnnotationとして保存しない。
    if (coords.width > 0.005 && coords.height > 0.005) {
      addBBox(image.id, coords, activeLabel);
    }
    setDrawing(null);
  };

  const handleStageClick = (e: Konva.KonvaEventObject<MouseEvent>) => {
    if (activeTool === "select" && (e.target === stageRef.current || e.target.getClassName() === "Image")) {
      select(null);
    }
    if (activeTool === "edge" && (e.target === stageRef.current || e.target.getClassName() === "Image")) {
      setEdgeSource(null);
      selectEdge(null);
    }
  };

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.key !== "Delete" && e.key !== "Backspace")) return;
      if (selectedEdgeId) {
        removeEdge(selectedEdgeId);
        return;
      }
      if (selectedId) {
        remove(selectedId);
      }
    },
    [selectedId, selectedEdgeId, remove, removeEdge],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  const findLabel = (labelId: string | null): LabelDefinition | null => {
    if (!labelId) return null;
    const label = labels.find((l) => l.id === labelId);
    return label ?? null;
  };

  const getLabelColor = (labelId: string | null): string => {
    return findLabel(labelId)?.color ?? "#64748B";
  };

  const getLabelName = (labelId: string | null): string => {
    if (!labelId) return "No label";
    return findLabel(labelId)?.name ?? "Unknown label";
  };

  const getLabelTextColor = (backgroundColor: string): string => {
    const hex = backgroundColor.replace("#", "");
    if (hex.length !== 6) return "#FFFFFF";
    const red = parseInt(hex.slice(0, 2), 16);
    const green = parseInt(hex.slice(2, 4), 16);
    const blue = parseInt(hex.slice(4, 6), 16);
    const luminance = (red * 299 + green * 587 + blue * 114) / 1000;
    return luminance > 160 ? "#111827" : "#FFFFFF";
  };

  const getEdgePoints = (sourceBox: DisplayBBox, targetBox: DisplayBBox): [number, number, number, number] => {
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
  };

  return (
    <div ref={containerRef} className="w-full h-full">
      <Stage
        ref={stageRef}
        width={stageSize.width}
        height={stageSize.height}
        scaleX={stageScale}
        scaleY={stageScale}
        x={stagePos.x}
        y={stagePos.y}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onClick={handleStageClick}
        draggable={activeTool === "pan"}
        onDragEnd={(e) => {
          if (e.target === stageRef.current) {
            setStagePos({ x: e.target.x(), y: e.target.y() });
          }
        }}
        style={{ cursor: activeTool === "bbox" || activeTool === "edge" ? "crosshair" : activeTool === "pan" ? "grab" : "default" }}
      >
        <Layer>
          {img && (
            <KonvaImage
              image={img}
              width={image.original_width * scale}
              height={image.original_height * scale}
              offsetX={(image.original_width * scale) / 2}
              offsetY={(image.original_height * scale) / 2}
              x={(image.width * scale) / 2}
              y={(image.height * scale) / 2}
              rotation={image.rotation}
              scaleX={image.flip_h ? -1 : 1}
              scaleY={image.flip_v ? -1 : 1}
            />
          )}

          {edges.map((edge) => {
            if (edge.type !== "reading_order") return null;
            const sourceBox = bboxDisplayByAnnotationID.get(edge.source_annotation_id);
            const targetBox = bboxDisplayByAnnotationID.get(edge.target_annotation_id);
            if (!sourceBox || !targetBox) return null;
            const isSelected = edge.id === selectedEdgeId;
            return (
              <Arrow
                key={edge.id}
                points={getEdgePoints(sourceBox, targetBox)}
                stroke={isSelected ? "#2563EB" : "#7C3AED"}
                fill={isSelected ? "#2563EB" : "#7C3AED"}
                strokeWidth={isSelected ? 3 : 2}
                pointerLength={8}
                pointerWidth={8}
                hitStrokeWidth={14}
                opacity={0.9}
                onClick={(e) => {
                  e.cancelBubble = true;
                  selectEdge(edge.id);
                }}
              />
            );
          })}

          {annotations.map((ann) => {
            if (ann.type !== "bbox") return null;
            const coords = ann.coordinates as BBoxCoordinates;
            const color = getLabelColor(ann.label_id);
            const labelName = getLabelName(ann.label_id);
            const isSelected = ann.id === selectedId;
            const isEdgeSource = ann.id === edgeSourceId;
            const x = coords.x * image.width * scale;
            const y = coords.y * image.height * scale;
            const width = coords.width * image.width * scale;
            const height = coords.height * image.height * scale;
            const labelY = y >= 22 ? y - 22 : y + 4;
            return (
              <Fragment key={ann.id}>
                <Rect
                  id={ann.id}
                  x={x}
                  y={y}
                  width={width}
                  height={height}
                  stroke={color}
                  strokeWidth={isSelected || isEdgeSource ? 3 : 2}
                  fill={`${color}20`}
                  dash={isEdgeSource && activeTool === "edge" ? [6, 3] : undefined}
                  shadowColor={isSelected || isEdgeSource ? color : undefined}
                  shadowBlur={isSelected || isEdgeSource ? 8 : 0}
                  draggable={activeTool === "select"}
                  onClick={(e) => {
                    e.cancelBubble = true;
                    if (activeTool === "edge") {
                      if (!edgeSourceId) {
                        setEdgeSource(ann.id);
                      } else {
                        addReadingOrderEdge(image.id, edgeSourceId, ann.id);
                      }
                      return;
                    }
                    select(ann.id);
                  }}
                  onDragEnd={(e) => {
                    const node = e.target;
                    const [nx, ny] = layerToNormalized(node.x(), node.y());
                    updateCoordinates(ann.id, {
                      ...coords,
                      x: nx,
                      y: ny,
                    });
                  }}
                  onTransformEnd={(e) => {
                    const node = e.target;
                    const scaleX = node.scaleX();
                    const scaleY = node.scaleY();
                    // Why: Konva Transformerはwidth/heightではなくscaleを変えるため、保存前に実寸へ畳み込む。
                    node.scaleX(1);
                    node.scaleY(1);
                    const [nx, ny] = layerToNormalized(node.x(), node.y());
                    const [nw] = layerToNormalized(node.width() * scaleX, 0);
                    const [, nh] = layerToNormalized(0, node.height() * scaleY);
                    updateCoordinates(ann.id, {
                      x: nx,
                      y: ny,
                      width: nw,
                      height: nh,
                    });
                  }}
                />
                <KonvaLabel
                  x={x}
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
                    fill={getLabelTextColor(color)}
                  />
                </KonvaLabel>
              </Fragment>
            );
          })}

          {drawing && (
            <Rect
              x={drawing.x * image.width * scale}
              y={drawing.y * image.height * scale}
              width={drawing.width * image.width * scale}
              height={drawing.height * image.height * scale}
              stroke="#3B82F6"
              strokeWidth={2}
              dash={[4, 4]}
            />
          )}

          <Transformer
            ref={transformerRef}
            rotateEnabled={false}
            keepRatio={false}
            boundBoxFunc={(_, newBox) => newBox}
          />
        </Layer>
      </Stage>
    </div>
  );
}
