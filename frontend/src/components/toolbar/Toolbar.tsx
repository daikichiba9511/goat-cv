import { useNavigate } from "react-router-dom";
import type { Tool } from "../../types";

type Props = {
  activeTool: Tool;
  onToolChange: (tool: Tool) => void;
  onSave: () => void;
  dirty: boolean;
  projectName: string;
  imageName: string | null;
  hasImage: boolean;
  onRotate: () => void;
  onFlipH: () => void;
  onFlipV: () => void;
};

const tools: { key: Tool; label: string }[] = [
  { key: "select", label: "Select" },
  { key: "bbox", label: "BBox" },
  { key: "edge", label: "Edge" },
  { key: "pan", label: "Pan" },
];

export default function Toolbar({
  activeTool,
  onToolChange,
  onSave,
  dirty,
  projectName,
  imageName,
  hasImage,
  onRotate,
  onFlipH,
  onFlipV,
}: Props) {
  const navigate = useNavigate();

  return (
    <div className="flex items-center gap-3 px-4 py-2 border-b bg-white">
      <button
        onClick={() => navigate("/")}
        className="text-gray-400 hover:text-gray-600 text-sm"
      >
        ← Projects
      </button>
      <span className="text-sm font-medium text-gray-700">
        {projectName}
        {imageName && <span className="text-gray-400"> / {imageName}</span>}
      </span>

      <div className="flex-1" />

      <div className="flex gap-1">
        {tools.map((t) => (
          <button
            key={t.key}
            onClick={() => onToolChange(t.key)}
            className={`px-3 py-1 text-sm rounded ${
              activeTool === t.key
                ? "bg-blue-600 text-white"
                : "bg-gray-100 hover:bg-gray-200"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {hasImage && (
        <div className="flex gap-1 border-l pl-3">
          <button
            onClick={onRotate}
            className="px-2 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded"
            title="Rotate 90°"
          >
            ↻
          </button>
          <button
            onClick={onFlipH}
            className="px-2 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded"
            title="Flip Horizontal"
          >
            ⇔
          </button>
          <button
            onClick={onFlipV}
            className="px-2 py-1 text-sm bg-gray-100 hover:bg-gray-200 rounded"
            title="Flip Vertical"
          >
            ⇕
          </button>
        </div>
      )}

      <button
        onClick={onSave}
        disabled={!dirty}
        className={`px-3 py-1 text-sm rounded ${
          dirty
            ? "bg-green-600 text-white hover:bg-green-700"
            : "bg-gray-100 text-gray-400 cursor-not-allowed"
        }`}
      >
        Save
      </button>
    </div>
  );
}
