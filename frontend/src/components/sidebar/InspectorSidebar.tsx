import { lazy, Suspense, useEffect, useState } from "react";
import { BookOpen, ListTree, PanelRightClose, PanelRightOpen, Tags } from "lucide-react";
import type { LabelDefinition } from "../../types";
import { useAnnotationStore } from "../../stores/annotationStore";
import AnnotationInspector from "./AnnotationInspector";
import LabelPanel from "./LabelPanel";

const GuidelinePanel = lazy(() => import("./GuidelinePanel"));

type Props = {
  labels: LabelDefinition[];
  activeLabel: string | null;
  onSelectLabel: (labelId: string | null) => void;
  onSelectAnnotation: (annotationId: string) => void;
};

type InspectorTab = "objects" | "labels" | "guidelines";

// InspectorSidebar keeps object inspection and label management in one fixed-width rail.
export default function InspectorSidebar({
  labels,
  activeLabel,
  onSelectLabel,
  onSelectAnnotation,
}: Props) {
  const [activeTab, setActiveTab] = useState<InspectorTab>("objects");
  const [guidelinesLoaded, setGuidelinesLoaded] = useState(false);
  // Why: 狭いViewportではCanvas幅を優先し、必要な時だけInspectorを開く。
  const [collapsed, setCollapsed] = useState(
    () => window.matchMedia?.("(max-width: 900px)").matches ?? false,
  );

  useEffect(() => {
    // Why: Canvas側で新しく選択された時だけObjectsへ戻し、手動で開いた他tabは維持する。
    return useAnnotationStore.subscribe((state, previousState) => {
      if (state.selectedId && state.selectedId !== previousState.selectedId) {
        setActiveTab("objects");
        setCollapsed(false);
      }
    });
  }, []);

  return (
    <aside className={`flex flex-shrink-0 flex-col border-l bg-white ${
      collapsed
        ? "w-10"
        : "w-60 max-md:fixed max-md:inset-y-0 max-md:right-0 max-md:z-20 xl:w-64"
    }`}>
      {collapsed ? (
        <button
          type="button"
          aria-label="Open inspector"
          title="Open inspector"
          onClick={() => setCollapsed(false)}
          className="flex h-10 w-10 items-center justify-center border-b text-gray-500 hover:bg-gray-50 hover:text-gray-700"
        >
          <PanelRightOpen aria-hidden="true" size={16} strokeWidth={1.75} />
        </button>
      ) : (
        <div role="tablist" aria-label="Inspector" className="grid h-10 grid-cols-[1fr_1fr_1fr_2.5rem] border-b">
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "objects"}
            aria-controls="objects-panel"
            onClick={() => setActiveTab("objects")}
            className={`inline-flex items-center justify-center gap-1 border-b-2 text-[11px] font-medium ${
              activeTab === "objects"
                ? "border-blue-600 text-blue-700"
                : "border-transparent text-gray-500 hover:bg-gray-50 hover:text-gray-700"
            }`}
          >
            <ListTree aria-hidden="true" size={15} strokeWidth={1.75} />
            Objects
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "labels"}
            aria-controls="labels-panel"
            onClick={() => setActiveTab("labels")}
            className={`inline-flex items-center justify-center gap-1 border-b-2 text-[11px] font-medium ${
              activeTab === "labels"
                ? "border-blue-600 text-blue-700"
                : "border-transparent text-gray-500 hover:bg-gray-50 hover:text-gray-700"
            }`}
          >
            <Tags aria-hidden="true" size={15} strokeWidth={1.75} />
            Labels
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeTab === "guidelines"}
            aria-controls="guidelines-panel"
            onClick={() => {
              setGuidelinesLoaded(true);
              setActiveTab("guidelines");
            }}
            className={`inline-flex items-center justify-center gap-1 border-b-2 text-[11px] font-medium ${
              activeTab === "guidelines"
                ? "border-blue-600 text-blue-700"
                : "border-transparent text-gray-500 hover:bg-gray-50 hover:text-gray-700"
            }`}
          >
            <BookOpen aria-hidden="true" size={14} strokeWidth={1.75} />
            Guide
          </button>
          <button
            type="button"
            aria-label="Close inspector"
            title="Close inspector"
            onClick={() => setCollapsed(true)}
            className="flex h-10 w-10 items-center justify-center text-gray-500 hover:bg-gray-50 hover:text-gray-700"
          >
            <PanelRightClose aria-hidden="true" size={16} strokeWidth={1.75} />
          </button>
        </div>
      )}

      <div
        id="objects-panel"
        role="tabpanel"
        className={!collapsed && activeTab === "objects" ? "min-h-0 flex-1" : "hidden"}
      >
        <AnnotationInspector labels={labels} onSelectAnnotation={onSelectAnnotation} />
      </div>
      <div
        id="labels-panel"
        role="tabpanel"
        className={!collapsed && activeTab === "labels" ? "min-h-0 flex-1" : "hidden"}
      >
        <LabelPanel
          labels={labels}
          activeLabel={activeLabel}
          onSelectLabel={onSelectLabel}
        />
      </div>
      <div
        id="guidelines-panel"
        role="tabpanel"
        className={!collapsed && activeTab === "guidelines" ? "min-h-0 flex-1" : "hidden"}
      >
        {guidelinesLoaded && (
          <Suspense fallback={<div className="p-3 text-xs text-gray-400">Loading...</div>}>
            <GuidelinePanel />
          </Suspense>
        )}
      </div>
    </aside>
  );
}
