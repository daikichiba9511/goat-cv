import { useEffect, useMemo, useRef, useState } from "react";
import { GitBranch, Trash2 } from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import type { Annotation, LabelDefinition } from "../../types";
import { categoryLabel } from "../../edgeRelations";
import { useAnnotationStore } from "../../stores/annotationStore";

type Props = {
  labels: LabelDefinition[];
  onSelectAnnotation: (annotationId: string) => void;
};

type ShapeFilter = "all" | Annotation["type"];

// AnnotationInspector lists and filters the current image's annotations without changing graph data.
export default function AnnotationInspector({ labels, onSelectAnnotation }: Props) {
  const {
    annotations,
    edges,
    selectedId,
    remove,
  } = useAnnotationStore(useShallow((state) => ({
    annotations: state.annotations,
    edges: state.edges,
    selectedId: state.selectedId,
    remove: state.remove,
  })));
  const [labelFilter, setLabelFilter] = useState("all");
  const [shapeFilter, setShapeFilter] = useState<ShapeFilter>("all");
  const rowByAnnotationId = useRef(new Map<string, HTMLDivElement>());

  const labelById = useMemo(
    () => new Map(labels.map((label) => [label.id, label])),
    [labels],
  );
  const connectionCountByAnnotationId = useMemo(() => {
    const counts = new Map<string, number>();
    for (const edge of edges) {
      counts.set(edge.source_annotation_id, (counts.get(edge.source_annotation_id) ?? 0) + 1);
      counts.set(edge.target_annotation_id, (counts.get(edge.target_annotation_id) ?? 0) + 1);
    }
    return counts;
  }, [edges]);
  const filteredAnnotations = useMemo(() => {
    return annotations.filter((annotation) => {
      const matchesLabel = labelFilter === "all"
        || (labelFilter === "unlabeled" && annotation.label_id === null)
        || annotation.label_id === labelFilter;
      const matchesShape = shapeFilter === "all" || annotation.type === shapeFilter;
      return matchesLabel && matchesShape;
    });
  }, [annotations, labelFilter, shapeFilter]);
  const selectedAnnotation = annotations.find((annotation) => annotation.id === selectedId);
  const selectedAnnotationIsFilteredOut = selectedAnnotation !== undefined
    && !filteredAnnotations.some((annotation) => annotation.id === selectedAnnotation.id);
  const visibleAnnotations = useMemo(
    () => selectedAnnotationIsFilteredOut
      ? [selectedAnnotation, ...filteredAnnotations]
      : filteredAnnotations,
    [filteredAnnotations, selectedAnnotation, selectedAnnotationIsFilteredOut],
  );
  const annotationPositionById = useMemo(
    () => new Map(annotations.map((annotation, index) => [annotation.id, index + 1])),
    [annotations],
  );

  useEffect(() => {
    if (!selectedId) return;
    rowByAnnotationId.current.get(selectedId)?.scrollIntoView({ block: "nearest" });
  }, [selectedId, visibleAnnotations]);

  return (
    <div className="flex h-full min-h-0 flex-col bg-white">
      <div className="grid grid-cols-2 gap-2 border-b p-3">
        <select
          aria-label="Filter by label"
          value={labelFilter}
          onChange={(event) => setLabelFilter(event.target.value)}
          className="min-w-0 rounded border bg-white px-2 py-1.5 text-xs text-gray-700"
        >
          <option value="all">All labels</option>
          <option value="unlabeled">No label</option>
          {labels.map((label) => (
            <option key={label.id} value={label.id}>{label.name}</option>
          ))}
        </select>
        <select
          aria-label="Filter by shape"
          value={shapeFilter}
          onChange={(event) => setShapeFilter(event.target.value as ShapeFilter)}
          className="min-w-0 rounded border bg-white px-2 py-1.5 text-xs text-gray-700"
        >
          <option value="all">All shapes</option>
          <option value="bbox">BBox</option>
          <option value="polygon">Polygon</option>
        </select>
        <div className="col-span-2 text-xs text-gray-500">
          {filteredAnnotations.length} of {annotations.length}
          {selectedAnnotationIsFilteredOut ? " + selected" : ""}
        </div>
      </div>

      <div
        role="list"
        aria-label="Annotations"
        className="min-h-0 flex-1 overflow-y-auto"
      >
        {visibleAnnotations.length === 0 && (
          <div className="px-3 py-8 text-center text-sm text-gray-500">
            No matching annotations
          </div>
        )}
        {visibleAnnotations.map((annotation) => {
          const label = annotation.label_id ? labelById.get(annotation.label_id) : undefined;
          const labelName = annotation.label_id ? label?.name ?? "Unknown label" : "No label";
          const labelColor = label?.color ?? "#64748B";
          const shapeName = annotation.type === "bbox" ? "BBox" : "Polygon";
          const categoryName = label ? categoryLabel(label.category) : "Unlabeled";
          const connectionCount = connectionCountByAnnotationId.get(annotation.id) ?? 0;
          const connectionText = `${connectionCount} connection${connectionCount === 1 ? "" : "s"}`;
          const isSelected = annotation.id === selectedId;
          const annotationPosition = annotationPositionById.get(annotation.id) ?? 0;

          return (
            <div
              key={annotation.id}
              ref={(row) => {
                if (row) rowByAnnotationId.current.set(annotation.id, row);
                else rowByAnnotationId.current.delete(annotation.id);
              }}
              role="listitem"
              className={`group flex min-h-14 items-center border-b pr-2 ${
                isSelected ? "bg-blue-50 text-blue-700" : "hover:bg-gray-50"
              }`}
            >
              <button
                type="button"
                aria-pressed={isSelected}
                aria-label={`Select ${labelName} ${shapeName} annotation ${annotationPosition}, ${connectionText}`}
                onClick={() => onSelectAnnotation(annotation.id)}
                className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 self-stretch px-3 py-2 text-left outline-none focus:bg-blue-50"
              >
                <span className="w-8 flex-shrink-0 font-mono text-xs text-gray-400">
                  {String(annotationPosition).padStart(3, "0")}
                </span>
                <span
                  className="h-3 w-3 flex-shrink-0 rounded-full"
                  style={{ backgroundColor: labelColor }}
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium" title={labelName}>
                    {labelName}
                  </span>
                  <span className="flex items-center gap-2 text-xs text-gray-500">
                    <span>{categoryName} / {shapeName}</span>
                    <span className="inline-flex items-center gap-1">
                      <GitBranch aria-hidden="true" size={12} strokeWidth={1.75} />
                      {connectionText}
                    </span>
                  </span>
                </span>
              </button>
              <button
                type="button"
                aria-label={`Delete ${labelName} annotation ${annotationPosition}`}
                title={`Delete ${labelName} annotation ${annotationPosition}`}
                onClick={() => remove(annotation.id)}
                className="flex h-8 w-8 flex-shrink-0 cursor-pointer items-center justify-center rounded text-gray-400 hover:bg-red-50 hover:text-red-600 focus:bg-red-50 focus:text-red-600"
              >
                <Trash2 aria-hidden="true" size={15} strokeWidth={1.75} />
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
