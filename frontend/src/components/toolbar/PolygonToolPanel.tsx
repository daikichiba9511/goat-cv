import { Check, Pentagon, Undo2, X } from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";

type Props = {
  imageId: string;
  activeLabelId: string | null;
};

export default function PolygonToolPanel({ imageId, activeLabelId }: Props) {
  const {
    polygonDraftPoints,
    undoPolygonDraftPoint,
    cancelPolygonDraft,
    finishPolygon,
  } = useAnnotationStore(useShallow((state) => ({
    polygonDraftPoints: state.polygonDraftPoints,
    undoPolygonDraftPoint: state.undoPolygonDraftPoint,
    cancelPolygonDraft: state.cancelPolygonDraft,
    finishPolygon: state.finishPolygon,
  })));
  const labels = useProjectStore((state) => state.labels);
  const activeLabel = labels.find((label) => label.id === activeLabelId);
  const canFinish = polygonDraftPoints.length >= 3;
  const hasDraft = polygonDraftPoints.length > 0;

  return (
    <section
      aria-label="Polygon drawing"
      className="flex min-h-11 flex-shrink-0 flex-nowrap items-center gap-1 border-b bg-white px-2 py-1.5 sm:gap-2 sm:px-3"
    >
      <span className="inline-flex flex-shrink-0 items-center gap-1.5 text-xs font-medium text-gray-800">
        <Pentagon aria-hidden="true" size={15} strokeWidth={1.75} />
        <span className="hidden sm:inline">Polygon</span>
      </span>
      <span className="flex-shrink-0 text-xs tabular-nums text-gray-600">
        {polygonDraftPoints.length} point{polygonDraftPoints.length === 1 ? "" : "s"}
      </span>
      <span className="hidden min-w-0 flex-1 truncate text-xs text-gray-500 sm:block">
        {activeLabel ? `Label: ${activeLabel.name}` : "No label"}
      </span>

      <button
        type="button"
        aria-label="Undo last polygon point"
        title="Undo last point"
        disabled={!hasDraft}
        onClick={undoPolygonDraftPoint}
        className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded border text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-300"
      >
        <Undo2 aria-hidden="true" size={15} strokeWidth={1.75} />
      </button>
      <button
        type="button"
        aria-label="Complete polygon"
        title="Complete polygon"
        disabled={!canFinish}
        onClick={() => finishPolygon(imageId, activeLabelId)}
        className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded border border-green-700 bg-green-700 text-white hover:bg-green-800 disabled:cursor-not-allowed disabled:border-gray-200 disabled:bg-gray-100 disabled:text-gray-300"
      >
        <Check aria-hidden="true" size={16} strokeWidth={2} />
      </button>
      <button
        type="button"
        aria-label="Cancel polygon"
        title="Cancel polygon"
        disabled={!hasDraft}
        onClick={cancelPolygonDraft}
        className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded border text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-300"
      >
        <X aria-hidden="true" size={16} strokeWidth={1.75} />
      </button>
    </section>
  );
}
