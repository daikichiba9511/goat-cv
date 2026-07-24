import { useMemo, useState, type FormEvent } from "react";
import type { LabelDefinition, LabelCategory } from "../../types";
import { useProjectStore } from "../../stores/projectStore";
import { useAnnotationStore } from "../../stores/annotationStore";

type Props = {
  labels: LabelDefinition[];
  activeLabel: string | null;
  onSelectLabel: (id: string | null) => void;
};

const DEFAULT_COLORS = [
  "#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4",
  "#FFEAA7", "#DDA0DD", "#98D8C8", "#F7DC6F",
];

const CATEGORIES: { value: LabelCategory; label: string }[] = [
  { value: "object", label: "Object" },
  { value: "entity", label: "Entity" },
  { value: "key", label: "Key" },
  { value: "value", label: "Value" },
  { value: "table", label: "Table" },
  { value: "cell", label: "Cell" },
];

export default function LabelPanel({ labels, activeLabel, onSelectLabel }: Props) {
  const { createLabel, updateLabel, deleteLabel } = useProjectStore();
  const { annotations, selectedId, setLabel } = useAnnotationStore();
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [color, setColor] = useState(DEFAULT_COLORS[0]);
  const [category, setCategory] = useState<LabelCategory>("object");
  const [editingLabelId, setEditingLabelId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");
  const [editColor, setEditColor] = useState(DEFAULT_COLORS[0]);
  const [editCategory, setEditCategory] = useState<LabelCategory>("object");
  const selectedAnnotation = annotations.find((annotation) => annotation.id === selectedId) ?? null;
  const selectedAnnotationLabelValue =
    selectedAnnotation && labels.some((label) => label.id === selectedAnnotation.label_id)
      ? selectedAnnotation.label_id ?? ""
      : "";
  const labelCounts = useMemo(() => {
    return annotations.reduce<Record<string, number>>((counts, annotation) => {
      const key = annotation.label_id ?? "";
      counts[key] = (counts[key] ?? 0) + 1;
      return counts;
    }, {});
  }, [annotations]);

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    await createLabel(name.trim(), color, category);
    setName("");
    setColor(DEFAULT_COLORS[(labels.length + 1) % DEFAULT_COLORS.length]);
    setShowForm(false);
  };

  const startEditing = (label: LabelDefinition) => {
    setEditingLabelId(label.id);
    setEditName(label.name);
    setEditColor(label.color);
    setEditCategory(label.category);
  };

  const cancelEditing = () => {
    setEditingLabelId(null);
    setEditName("");
    setEditColor(DEFAULT_COLORS[0]);
    setEditCategory("object");
  };

  const handleUpdate = async (e: FormEvent) => {
    e.preventDefault();
    if (!editingLabelId || !editName.trim()) return;
    await updateLabel(editingLabelId, editName.trim(), editColor, editCategory);
    cancelEditing();
  };

  return (
    <div className="flex h-full min-h-0 flex-col bg-white">
      <div className="p-3 border-b flex items-center justify-between">
        <span className="text-sm font-medium text-gray-700">Labels</span>
        <button
          onClick={() => setShowForm(!showForm)}
          className="text-blue-600 hover:text-blue-800 text-lg leading-none"
        >
          {showForm ? "−" : "+"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="p-3 border-b space-y-2">
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Label name"
            className="w-full border rounded px-2 py-1 text-sm"
            autoFocus
          />
          <div className="flex gap-1 flex-wrap">
            {DEFAULT_COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColor(c)}
                className={`w-5 h-5 rounded-full border-2 ${
                  color === c ? "border-gray-800" : "border-transparent"
                }`}
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value as LabelCategory)}
            className="w-full border rounded px-2 py-1 text-sm"
          >
            {CATEGORIES.map((c) => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </select>
          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-1 rounded text-sm hover:bg-blue-700"
          >
            Add
          </button>
        </form>
      )}

      {selectedAnnotation && (
        <div className="p-3 border-b bg-slate-50 space-y-2">
          <div className="flex items-center justify-between gap-2">
            <span className="text-xs font-semibold uppercase text-slate-500">
              Selected
            </span>
            <span className="max-w-24 truncate rounded bg-white px-2 py-0.5 text-xs text-slate-500 border">
              {selectedAnnotation.type.toUpperCase()}
            </span>
          </div>
          <select
            value={selectedAnnotationLabelValue}
            onChange={(e) => setLabel(selectedAnnotation.id, e.target.value || null)}
            className="w-full border rounded px-2 py-1.5 text-sm bg-white"
          >
            <option value="">No label</option>
            {labels.map((label) => (
              <option key={label.id} value={label.id}>
                {label.name} / {label.category}
              </option>
            ))}
          </select>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        <div
          onClick={() => onSelectLabel(null)}
          className={`px-3 py-2 cursor-pointer text-sm border-b flex items-center justify-between gap-2 ${
            activeLabel === null
              ? "bg-blue-50 text-blue-700 font-medium"
              : "hover:bg-gray-50"
          }`}
        >
          <span>(No label)</span>
          {labelCounts[""] > 0 && (
            <span className="ml-2 rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">
              {labelCounts[""]}
            </span>
          )}
        </div>
        {labels.map((label) => {
          if (label.id === editingLabelId) {
            return (
              <form
                key={label.id}
                onSubmit={handleUpdate}
                className="p-3 border-b bg-slate-50 space-y-2"
              >
                <input
                  type="text"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  className="w-full border rounded px-2 py-1 text-sm"
                  autoFocus
                />
                <div className="flex gap-1 flex-wrap">
                  {DEFAULT_COLORS.map((presetColor) => (
                    <button
                      key={presetColor}
                      type="button"
                      onClick={() => setEditColor(presetColor)}
                      className={`w-5 h-5 rounded-full border-2 ${
                        editColor === presetColor ? "border-gray-800" : "border-transparent"
                      }`}
                      style={{ backgroundColor: presetColor }}
                    />
                  ))}
                </div>
                <input
                  type="color"
                  value={editColor}
                  onChange={(e) => setEditColor(e.target.value)}
                  className="h-7 w-full border rounded bg-white"
                />
                <select
                  value={editCategory}
                  onChange={(e) => setEditCategory(e.target.value as LabelCategory)}
                  className="w-full border rounded px-2 py-1 text-sm bg-white"
                >
                  {CATEGORIES.map((categoryOption) => (
                    <option key={categoryOption.value} value={categoryOption.value}>
                      {categoryOption.label}
                    </option>
                  ))}
                </select>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="submit"
                    className="bg-blue-600 text-white py-1 rounded text-sm hover:bg-blue-700 disabled:bg-slate-300"
                    disabled={!editName.trim()}
                  >
                    Save
                  </button>
                  <button
                    type="button"
                    onClick={cancelEditing}
                    className="border py-1 rounded text-sm hover:bg-white"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            );
          }

          return (
            <div
              key={label.id}
              onClick={() => onSelectLabel(label.id)}
              className={`px-3 py-2 cursor-pointer text-sm border-b flex items-center gap-2 group ${
                label.id === activeLabel
                  ? "bg-blue-50 text-blue-700 font-medium"
                  : "hover:bg-gray-50"
              }`}
            >
              <span
                className="inline-block w-3 h-3 rounded-full flex-shrink-0"
                style={{ backgroundColor: label.color }}
              />
              <span className="flex-1 truncate">{label.name}</span>
              {(labelCounts[label.id] ?? 0) > 0 && (
                <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">
                  {labelCounts[label.id]}
                </span>
              )}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  startEditing(label);
                }}
                className="text-slate-400 hover:text-slate-700 text-xs opacity-0 group-hover:opacity-100"
              >
                Edit
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  deleteLabel(label.id);
                  if (activeLabel === label.id) onSelectLabel(null);
                  if (editingLabelId === label.id) cancelEditing();
                }}
                className="text-red-400 hover:text-red-600 text-xs opacity-0 group-hover:opacity-100"
              >
                ×
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
}
