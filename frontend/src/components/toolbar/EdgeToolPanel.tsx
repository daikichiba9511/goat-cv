import { KeyRound, ListOrdered, Table2, X } from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import { EDGE_RELATIONS, EDGE_TYPE_ORDER, categoryLabel } from "../../edgeRelations";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import type { EdgeType } from "../../types";

const relationIcons = {
  reading_order: ListOrdered,
  key_value: KeyRound,
  table_cell: Table2,
} satisfies Record<EdgeType, typeof ListOrdered>;

// EdgeToolPanelはrelation制約と2クリックで作成中のEdge状態を表示する。
export default function EdgeToolPanel() {
  const {
    annotations,
    edgeType,
    edgeSourceId,
    edgeDraftError,
    setEdgeType,
    cancelEdgeDraft,
  } = useAnnotationStore(useShallow((state) => ({
    annotations: state.annotations,
    edgeType: state.edgeType,
    edgeSourceId: state.edgeSourceId,
    edgeDraftError: state.edgeDraftError,
    setEdgeType: state.setEdgeType,
    cancelEdgeDraft: state.cancelEdgeDraft,
  })));
  const labels = useProjectStore((state) => state.labels);
  const relation = EDGE_RELATIONS[edgeType];
  const sourceAnnotation = annotations.find((annotation) => annotation.id === edgeSourceId);
  const sourceLabel = sourceAnnotation?.label_id
    ? labels.find((label) => label.id === sourceAnnotation.label_id)
    : undefined;
  const sourceText = sourceAnnotation
    ? sourceLabel
      ? `${sourceLabel.name} / ${categoryLabel(sourceLabel.category)}`
      : "Unlabeled annotation"
    : "Select a source annotation";

  return (
    <section
      aria-label="Edge relation"
      className="flex flex-shrink-0 flex-wrap items-center gap-x-3 gap-y-1 border-b bg-white px-2 py-1.5 sm:px-3"
    >
      <select
        aria-label="Edge relation type"
        value={edgeType}
        onChange={(event) => setEdgeType(event.target.value as EdgeType)}
        className="w-36 flex-shrink-0 rounded border bg-white px-2 py-1.5 text-xs md:hidden"
      >
        {EDGE_TYPE_ORDER.map((type) => (
          <option key={type} value={type}>{EDGE_RELATIONS[type].label}</option>
        ))}
      </select>

      <div
        role="group"
        aria-label="Edge relation type"
        className="hidden h-8 flex-shrink-0 overflow-hidden rounded border md:flex"
      >
        {EDGE_TYPE_ORDER.map((type) => {
          const definition = EDGE_RELATIONS[type];
          const Icon = relationIcons[type];
          const isActive = type === edgeType;
          return (
            <button
              key={type}
              type="button"
              aria-pressed={isActive}
              onClick={() => setEdgeType(type)}
              className={`inline-flex min-w-32 items-center justify-center gap-1.5 border-r px-2.5 text-xs last:border-r-0 ${
                isActive
                  ? "bg-gray-800 text-white"
                  : "bg-white text-gray-600 hover:bg-gray-50"
              }`}
            >
              <Icon aria-hidden="true" size={14} strokeWidth={1.75} />
              {definition.label}
            </button>
          );
        })}
      </div>

      <div className="min-w-48 flex-1 text-xs leading-5 text-gray-600">
        <span className="font-medium text-gray-800">{sourceText}</span>
        <span className="mx-1.5 text-gray-300">|</span>
        <span>{relation.instruction}</span>
      </div>

      {(edgeSourceId || edgeDraftError) && (
        <button
          type="button"
          aria-label="Cancel edge relation"
          title="Cancel edge relation"
          onClick={cancelEdgeDraft}
          className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-100 hover:text-gray-700"
        >
          <X aria-hidden="true" size={15} strokeWidth={1.75} />
        </button>
      )}

      {edgeDraftError && (
        <div role="alert" className="w-full text-xs leading-5 text-red-600">
          {edgeDraftError}
        </div>
      )}
    </section>
  );
}
