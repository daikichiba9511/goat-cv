import { useState } from "react";
import type { LabelDefinition, LabelCategory } from "../../types";
import { useProjectStore } from "../../stores/projectStore";

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
  const { createLabel, deleteLabel } = useProjectStore();
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [color, setColor] = useState(DEFAULT_COLORS[0]);
  const [category, setCategory] = useState<LabelCategory>("object");

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    await createLabel(name.trim(), color, category);
    setName("");
    setColor(DEFAULT_COLORS[(labels.length + 1) % DEFAULT_COLORS.length]);
    setShowForm(false);
  };

  return (
    <div className="w-48 border-l bg-white flex flex-col">
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

      <div className="flex-1 overflow-y-auto">
        <div
          onClick={() => onSelectLabel(null)}
          className={`px-3 py-2 cursor-pointer text-sm border-b ${
            activeLabel === null
              ? "bg-blue-50 text-blue-700 font-medium"
              : "hover:bg-gray-50"
          }`}
        >
          (No label)
        </div>
        {labels.map((l) => (
          <div
            key={l.id}
            onClick={() => onSelectLabel(l.id)}
            className={`px-3 py-2 cursor-pointer text-sm border-b flex items-center gap-2 group ${
              l.id === activeLabel
                ? "bg-blue-50 text-blue-700 font-medium"
                : "hover:bg-gray-50"
            }`}
          >
            <span
              className="inline-block w-3 h-3 rounded-full flex-shrink-0"
              style={{ backgroundColor: l.color }}
            />
            <span className="flex-1 truncate">{l.name}</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                deleteLabel(l.id);
                if (activeLabel === l.id) onSelectLabel(null);
              }}
              className="text-red-400 hover:text-red-600 text-xs opacity-0 group-hover:opacity-100"
            >
              ×
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
