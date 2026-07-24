import { useRef, useEffect, useState, useCallback } from "react";
import {
  Stage,
  Layer,
  Image as KonvaImage,
  Rect,
  Transformer,
} from "react-konva";
import type Konva from "konva";
import { useShallow } from "zustand/react/shallow";
import type { ImageMeta, Tool } from "../../types";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import { imageFileUrl } from "../../api/client";
import AnnotationOverlay from "./AnnotationOverlay";
import { normalizeBBox } from "./annotationGeometry";

type Props = {
  image: ImageMeta;
  activeTool: Tool;
  activeLabel: string | null;
};

export default function AnnotationCanvas({ image, activeTool, activeLabel }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<Konva.Stage>(null);
  const transformerRef = useRef<Konva.Transformer>(null);
  const drawingRectRef = useRef<Konva.Rect>(null);
  const drawingStartRef = useRef<[number, number] | null>(null);
  const [img, setImg] = useState<HTMLImageElement | null>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });
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
  } = useAnnotationStore(useShallow((state) => ({
    annotations: state.annotations,
    edges: state.edges,
    selectedId: state.selectedId,
    selectedEdgeId: state.selectedEdgeId,
    edgeSourceId: state.edgeSourceId,
    select: state.select,
    selectEdge: state.selectEdge,
    setEdgeSource: state.setEdgeSource,
    addReadingOrderEdge: state.addReadingOrderEdge,
    addBBox: state.addBBox,
    updateCoordinates: state.updateCoordinates,
    remove: state.remove,
    removeEdge: state.removeEdge,
  })));
  const labels = useProjectStore((state) => state.labels);

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
    drawingStartRef.current = [nx, ny];
    const drawingRect = drawingRectRef.current;
    if (drawingRect) {
      drawingRect.setAttrs({
        x: nx * image.width * scale,
        y: ny * image.height * scale,
        width: 0,
        height: 0,
        visible: true,
      });
      drawingRect.getLayer()?.batchDraw();
    }
    select(null);
  };

  const handleMouseMove = () => {
    const drawingStart = drawingStartRef.current;
    if (!drawingStart || activeTool !== "bbox") return;
    const stage = stageRef.current;
    if (!stage) return;
    const pos = stage.getPointerPosition();
    if (!pos) return;

    const [nx, ny] = screenToNormalized(pos.x, pos.y);
    // Why: mousemoveごとのReact再描画を避け、入力中だけKonva Nodeを直接更新する。
    const drawingRect = drawingRectRef.current;
    drawingRect?.setAttrs({
      width: (nx - drawingStart[0]) * image.width * scale,
      height: (ny - drawingStart[1]) * image.height * scale,
    });
    drawingRect?.getLayer()?.batchDraw();
  };

  const handleMouseUp = () => {
    const drawingStart = drawingStartRef.current;
    if (!drawingStart || activeTool !== "bbox") return;
    const stage = stageRef.current;
    const pos = stage?.getPointerPosition();
    if (!pos) {
      drawingStartRef.current = null;
      drawingRectRef.current?.visible(false);
      drawingRectRef.current?.getLayer()?.batchDraw();
      return;
    }
    const [nx, ny] = screenToNormalized(pos.x, pos.y);

    // Why: annotatorが右下以外へドラッグしても、保存時は正の幅高さを持つBBoxへ正規化する。
    const coords = normalizeBBox({
      x: drawingStart[0],
      y: drawingStart[1],
      width: nx - drawingStart[0],
      height: ny - drawingStart[1],
    });
    // Why not: クリックや手ぶれでできる微小BBoxはAnnotationとして保存しない。
    if (coords.width > 0.005 && coords.height > 0.005) {
      addBBox(image.id, coords, activeLabel);
    }
    drawingStartRef.current = null;
    const drawingRect = drawingRectRef.current;
    drawingRect?.visible(false);
    drawingRect?.getLayer()?.batchDraw();
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

  const handleAnnotationClick = useCallback((annotationId: string) => {
    if (activeTool === "edge") {
      if (!edgeSourceId) {
        setEdgeSource(annotationId);
      } else {
        addReadingOrderEdge(image.id, edgeSourceId, annotationId);
      }
      return;
    }
    select(annotationId);
  }, [
    activeTool,
    edgeSourceId,
    setEdgeSource,
    addReadingOrderEdge,
    image.id,
    select,
  ]);

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

          <AnnotationOverlay
            annotations={annotations}
            edges={edges}
            labels={labels}
            selectedAnnotationId={selectedId}
            selectedEdgeId={selectedEdgeId}
            edgeSourceId={edgeSourceId}
            activeTool={activeTool}
            imageWidth={image.width}
            imageHeight={image.height}
            scale={scale}
            onAnnotationClick={handleAnnotationClick}
            onEdgeClick={selectEdge}
            onCoordinatesChange={updateCoordinates}
            layerToNormalized={layerToNormalized}
          />

          <Transformer
            ref={transformerRef}
            rotateEnabled={false}
            keepRatio={false}
            boundBoxFunc={(_, newBox) => newBox}
          />
        </Layer>

        <Layer listening={false}>
          <Rect
            ref={drawingRectRef}
            stroke="#3B82F6"
            strokeWidth={2}
            dash={[4, 4]}
          />
        </Layer>
      </Stage>
    </div>
  );
}
