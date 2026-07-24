import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useProjectStore } from "../stores/projectStore";
import { useAnnotationStore } from "../stores/annotationStore";
import * as api from "../api/client";
import type { ImageMeta, Tool } from "../types";
import AnnotationCanvas from "../components/canvas/AnnotationCanvas";
import Sidebar from "../components/sidebar/Sidebar";
import Toolbar from "../components/toolbar/Toolbar";
import EdgeToolPanel from "../components/toolbar/EdgeToolPanel";
import PolygonToolPanel from "../components/toolbar/PolygonToolPanel";
import InspectorSidebar from "../components/sidebar/InspectorSidebar";

export default function Annotator() {
  const { projectId } = useParams<{ projectId: string }>();
  const { currentProject, selectProject, labels, images } = useProjectStore();
  const {
    loadAnnotations,
    save,
    dirty,
    saving,
    saveError,
    select,
    clear,
    cancelEdgeDraft,
    cancelPolygonDraft,
  } = useAnnotationStore();
  const [currentImage, setCurrentImage] = useState<ImageMeta | null>(null);
  const [activeTool, setActiveTool] = useState<Tool>("select");
  const [activeLabel, setActiveLabel] = useState<string | null>(null);

  useEffect(() => {
    if (projectId) {
      selectProject(projectId);
    }
  }, [projectId, selectProject]);

  useEffect(() => {
    if (currentImage) {
      loadAnnotations(currentImage.id);
    } else {
      clear();
    }
  }, [currentImage, loadAnnotations, clear]);

  const handleSelectImage = (img: ImageMeta) => {
    cancelPolygonDraft();
    setCurrentImage(img);
  };

  const handleSave = async () => {
    if (currentImage) {
      await save(currentImage.id);
    }
  };

  const handleToolChange = (tool: Tool) => {
    if (tool !== "edge") {
      // Why: Edge toolを離れた後のクリックで、非表示の始点からEdgeが作られないようにする。
      cancelEdgeDraft();
    }
    if (tool !== "polygon") {
      // Why: 非表示になった作成途中の輪郭が、後で誤って確定されないようにする。
      cancelPolygonDraft();
    }
    setActiveTool(tool);
  };

  const handleSelectAnnotation = (annotationId: string) => {
    // Why: Inspector選択時はSelect toolへ戻し、Canvas上の選択枠と編集Handleを同時に表示する。
    handleToolChange("select");
    select(annotationId);
  };

  const handleRotate = async () => {
    if (!currentImage) return;
    const nextRotation = ((currentImage.rotation + 90) % 360) as 0 | 90 | 180 | 270;
    const updated = await api.updateImageTransform(
      currentImage.id, nextRotation, currentImage.flip_h, currentImage.flip_v,
    );
    setCurrentImage(updated);
  };

  const handleFlipH = async () => {
    if (!currentImage) return;
    const updated = await api.updateImageTransform(
      currentImage.id, currentImage.rotation, !currentImage.flip_h, currentImage.flip_v,
    );
    setCurrentImage(updated);
  };

  const handleFlipV = async () => {
    if (!currentImage) return;
    const updated = await api.updateImageTransform(
      currentImage.id, currentImage.rotation, currentImage.flip_h, !currentImage.flip_v,
    );
    setCurrentImage(updated);
  };

  if (!currentProject) {
    return <div className="p-8">Loading...</div>;
  }

  return (
    <div className="flex h-screen w-full overflow-hidden">
      <Sidebar
        images={images}
        currentImageId={currentImage?.id ?? null}
        onSelectImage={handleSelectImage}
      />

      <div className="flex min-w-0 flex-1 flex-col">
        <Toolbar
          activeTool={activeTool}
          onToolChange={handleToolChange}
          onSave={handleSave}
          dirty={dirty}
          saving={saving}
          saveError={saveError}
          projectName={currentProject.name}
          imageName={currentImage?.filename ?? null}
          hasImage={currentImage !== null}
          onRotate={handleRotate}
          onFlipH={handleFlipH}
          onFlipV={handleFlipV}
        />

        {activeTool === "edge" && currentImage && <EdgeToolPanel />}
        {activeTool === "polygon" && currentImage && (
          <PolygonToolPanel imageId={currentImage.id} activeLabelId={activeLabel} />
        )}

        <div className="flex-1 relative overflow-hidden bg-gray-200">
          {currentImage ? (
            <AnnotationCanvas
              image={currentImage}
              activeTool={activeTool}
              activeLabel={activeLabel}
            />
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              Select an image from the sidebar
            </div>
          )}
        </div>
      </div>

      <InspectorSidebar
        labels={labels}
        activeLabel={activeLabel}
        onSelectLabel={setActiveLabel}
        onSelectAnnotation={handleSelectAnnotation}
        currentImageId={currentImage?.id ?? null}
      />
    </div>
  );
}
