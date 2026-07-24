import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { useProjectStore } from "../stores/projectStore";
import { useAnnotationStore } from "../stores/annotationStore";
import * as api from "../api/client";
import type { ImageMeta, Tool } from "../types";
import AnnotationCanvas from "../components/canvas/AnnotationCanvas";
import Sidebar from "../components/sidebar/Sidebar";
import Toolbar from "../components/toolbar/Toolbar";
import LabelPanel from "../components/sidebar/LabelPanel";

export default function Annotator() {
  const { projectId } = useParams<{ projectId: string }>();
  const { currentProject, selectProject, labels, images } = useProjectStore();
  const { loadAnnotations, save, dirty, clear } = useAnnotationStore();
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
    setCurrentImage(img);
  };

  const handleSave = async () => {
    if (currentImage) {
      await save(currentImage.id);
    }
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
    <div className="flex h-screen">
      <Sidebar
        images={images}
        currentImageId={currentImage?.id ?? null}
        onSelectImage={handleSelectImage}
      />

      <div className="flex-1 flex flex-col">
        <Toolbar
          activeTool={activeTool}
          onToolChange={setActiveTool}
          onSave={handleSave}
          dirty={dirty}
          projectName={currentProject.name}
          imageName={currentImage?.filename ?? null}
          hasImage={currentImage !== null}
          onRotate={handleRotate}
          onFlipH={handleFlipH}
          onFlipV={handleFlipV}
        />

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

      <LabelPanel
        labels={labels}
        activeLabel={activeLabel}
        onSelectLabel={setActiveLabel}
      />
    </div>
  );
}
