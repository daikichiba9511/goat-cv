import { useRef, useEffect, useState, useCallback } from "react";
import { Stage, Layer, Image as KonvaImage, Rect, Transformer } from "react-konva";
import type Konva from "konva";
import type { ImageMeta, BBoxCoordinates } from "../../types";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import { imageFileUrl } from "../../api/client";
import type { Tool } from "../../pages/Annotator";

type Props = {
  image: ImageMeta;
  activeTool: Tool;
  activeLabel: string | null;
};

export default function AnnotationCanvas({ image, activeTool, activeLabel }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<Konva.Stage>(null);
  const transformerRef = useRef<Konva.Transformer>(null);
  const [img, setImg] = useState<HTMLImageElement | null>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });
  const [drawing, setDrawing] = useState<BBoxCoordinates | null>(null);
  const [stageScale, setStageScale] = useState(1);
  const [stagePos, setStagePos] = useState({ x: 0, y: 0 });

  const { annotations, selectedId, select, addBBox, updateCoordinates, remove } =
    useAnnotationStore();
  const { labels } = useProjectStore();

  // Load image
  useEffect(() => {
    const el = new window.Image();
    el.src = imageFileUrl(image.id);
    el.onload = () => setImg(el);
    return () => { el.onload = null; };
  }, [image.id]);

  // Resize stage to fill container
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const obs = new ResizeObserver((entries) => {
      const { width, height } = entries[0].contentRect;
      setStageSize({ width, height });
    });
    obs.observe(container);
    return () => obs.disconnect();
  }, []);

  // Compute scale to fit image in stage
  const scale = img && stageSize.width > 0
    ? Math.min(stageSize.width / image.width, stageSize.height / image.height)
    : 1;

  // Update transformer when selection changes
  useEffect(() => {
    const tr = transformerRef.current;
    const stage = stageRef.current;
    if (!tr || !stage) return;

    if (selectedId && activeTool === "select") {
      const node = stage.findOne(`#${CSS.escape(selectedId)}`);
      if (node) {
        tr.nodes([node]);
        tr.getLayer()?.batchDraw();
        return;
      }
    }
    tr.nodes([]);
    tr.getLayer()?.batchDraw();
  }, [selectedId, activeTool, annotations]);

  // Convert screen pixel position to normalized image coordinates (0-1)
  const screenToNormalized = useCallback(
    (screenX: number, screenY: number): [number, number] => {
      const imageX = (screenX - stagePos.x) / stageScale;
      const imageY = (screenY - stagePos.y) / stageScale;
      return [imageX / (image.width * scale), imageY / (image.height * scale)];
    },
    [image.width, image.height, scale, stageScale, stagePos],
  );

  // Convert local layer coordinates to normalized (used by drag/transform handlers)
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

  const handleMouseDown = (e: Konva.KonvaEventObject<MouseEvent>) => {
    if (activeTool !== "bbox") return;
    const stage = stageRef.current;
    if (!stage) return;
    const pos = stage.getPointerPosition();
    if (!pos) return;

    const [nx, ny] = screenToNormalized(pos.x, pos.y);
    setDrawing({ x: nx, y: ny, width: 0, height: 0 });
    select(null);
  };

  const handleMouseMove = (e: Konva.KonvaEventObject<MouseEvent>) => {
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
    // Normalize negative width/height
    const coords: BBoxCoordinates = {
      x: drawing.width < 0 ? drawing.x + drawing.width : drawing.x,
      y: drawing.height < 0 ? drawing.y + drawing.height : drawing.y,
      width: Math.abs(drawing.width),
      height: Math.abs(drawing.height),
    };
    // Ignore tiny boxes
    if (coords.width > 0.005 && coords.height > 0.005) {
      addBBox(image.id, coords, activeLabel);
    }
    setDrawing(null);
  };

  const handleStageClick = (e: Konva.KonvaEventObject<MouseEvent>) => {
    if (activeTool !== "select") return;
    if (e.target === stageRef.current || e.target.getClassName() === "Image") {
      select(null);
    }
  };

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.key === "Delete" || e.key === "Backspace") && selectedId) {
        remove(selectedId);
      }
    },
    [selectedId, remove],
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  const getLabelColor = (labelId: string | null): string => {
    if (!labelId) return "#3B82F6";
    const label = labels.find((l) => l.id === labelId);
    return label?.color ?? "#3B82F6";
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
        style={{ cursor: activeTool === "bbox" ? "crosshair" : activeTool === "pan" ? "grab" : "default" }}
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

          {annotations.map((ann) => {
            if (ann.type !== "bbox") return null;
            const coords = ann.coordinates as BBoxCoordinates;
            const color = getLabelColor(ann.label_id);
            return (
              <Rect
                key={ann.id}
                id={ann.id}
                x={coords.x * image.width * scale}
                y={coords.y * image.height * scale}
                width={coords.width * image.width * scale}
                height={coords.height * image.height * scale}
                stroke={color}
                strokeWidth={2}
                fill={`${color}20`}
                draggable={activeTool === "select"}
                onClick={(e) => {
                  e.cancelBubble = true;
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
