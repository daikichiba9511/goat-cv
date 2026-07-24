import { useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  CircleAlert,
  Ellipsis,
  FlipHorizontal2,
  FlipVertical2,
  RotateCw,
  Save as SaveIcon,
} from "lucide-react";
import type { Tool } from "../../types";

type Props = {
  activeTool: Tool;
  onToolChange: (tool: Tool) => void;
  onSave: () => void;
  dirty: boolean;
  saving: boolean;
  saveError: string | null;
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
  saving,
  saveError,
  projectName,
  imageName,
  hasImage,
  onRotate,
  onFlipH,
  onFlipV,
}: Props) {
  const navigate = useNavigate();
  const transformMenuRef = useRef<HTMLDetailsElement>(null);
  const imageContext = imageName ? `${projectName} / ${imageName}` : projectName;
  const runTransform = (transform: () => void) => {
    transform();
    transformMenuRef.current?.removeAttribute("open");
  };

  return (
    <div className="flex h-11 flex-shrink-0 items-center gap-2 border-b bg-white px-2 sm:px-3">
      <button
        type="button"
        onClick={() => navigate("/")}
        aria-label="Back to projects"
        title="Back to projects"
        className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded text-gray-400 hover:bg-gray-100 hover:text-gray-600"
      >
        <ArrowLeft aria-hidden="true" size={17} strokeWidth={1.75} />
      </button>
      <span
        className="hidden min-w-0 flex-1 truncate text-sm font-medium text-gray-700 sm:block"
        title={imageContext}
      >
        {projectName}
        {imageName && <span className="text-gray-400"> / {imageName}</span>}
      </span>

      <select
        aria-label="Annotation tool"
        value={activeTool}
        onChange={(event) => onToolChange(event.target.value as Tool)}
        className="w-24 flex-shrink-0 rounded border bg-white px-2 py-1 text-sm lg:hidden"
      >
        {tools.map((tool) => (
          <option key={tool.key} value={tool.key}>{tool.label}</option>
        ))}
      </select>

      <div className="hidden flex-shrink-0 gap-1 lg:flex">
        {tools.map((t) => (
          <button
            type="button"
            key={t.key}
            onClick={() => onToolChange(t.key)}
            className={`rounded px-2.5 py-1 text-sm ${
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
        <div className="hidden flex-shrink-0 gap-1 border-l pl-2 lg:flex">
          <button
            type="button"
            onClick={onRotate}
            className="flex h-8 w-8 items-center justify-center rounded bg-gray-100 text-gray-700 hover:bg-gray-200"
            aria-label="Rotate 90 degrees"
            title="Rotate 90 degrees"
          >
            <RotateCw aria-hidden="true" size={15} strokeWidth={1.75} />
          </button>
          <button
            type="button"
            onClick={onFlipH}
            className="flex h-8 w-8 items-center justify-center rounded bg-gray-100 text-gray-700 hover:bg-gray-200"
            aria-label="Flip horizontally"
            title="Flip Horizontal"
          >
            <FlipHorizontal2 aria-hidden="true" size={15} strokeWidth={1.75} />
          </button>
          <button
            type="button"
            onClick={onFlipV}
            className="flex h-8 w-8 items-center justify-center rounded bg-gray-100 text-gray-700 hover:bg-gray-200"
            aria-label="Flip vertically"
            title="Flip Vertical"
          >
            <FlipVertical2 aria-hidden="true" size={15} strokeWidth={1.75} />
          </button>
        </div>
      )}

      {hasImage && (
        <details ref={transformMenuRef} className="relative flex-shrink-0 lg:hidden">
          <summary
            aria-label="Image transforms"
            title="Image transforms"
            className="flex h-8 w-8 cursor-pointer list-none items-center justify-center rounded bg-gray-100 text-gray-700 hover:bg-gray-200"
          >
            <Ellipsis aria-hidden="true" size={16} strokeWidth={1.75} />
          </summary>
          <div className="absolute right-0 top-9 z-30 w-44 border bg-white py-1 shadow-lg">
            <button
              type="button"
              onClick={() => runTransform(onRotate)}
              className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50"
            >
              <RotateCw aria-hidden="true" size={15} strokeWidth={1.75} />
              Rotate 90 degrees
            </button>
            <button
              type="button"
              onClick={() => runTransform(onFlipH)}
              className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50"
            >
              <FlipHorizontal2 aria-hidden="true" size={15} strokeWidth={1.75} />
              Flip horizontally
            </button>
            <button
              type="button"
              onClick={() => runTransform(onFlipV)}
              className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-50"
            >
              <FlipVertical2 aria-hidden="true" size={15} strokeWidth={1.75} />
              Flip vertically
            </button>
          </div>
        </details>
      )}

      {saveError && (
        <span
          role="alert"
          title={saveError}
          className="hidden max-w-64 truncate text-xs text-red-600 sm:block"
        >
          {saveError}
        </span>
      )}
      {saveError && (
        <span role="alert" title={saveError} className="text-red-600 sm:hidden">
          <CircleAlert aria-hidden="true" size={16} strokeWidth={1.75} />
          <span className="sr-only">{saveError}</span>
        </span>
      )}

      <button
        type="button"
        onClick={onSave}
        disabled={!dirty || saving}
        className={`hidden min-w-16 flex-shrink-0 rounded px-3 py-1 text-sm sm:block ${
          dirty && !saving
            ? "bg-green-600 text-white hover:bg-green-700"
            : "bg-gray-100 text-gray-400 cursor-not-allowed"
        }`}
      >
        {saving ? "Saving..." : "Save"}
      </button>
      <button
        type="button"
        aria-label={saving ? "Saving" : "Save"}
        title={saving ? "Saving" : "Save"}
        onClick={onSave}
        disabled={!dirty || saving}
        className={`flex h-8 w-8 flex-shrink-0 items-center justify-center rounded sm:hidden ${
          dirty && !saving
            ? "bg-green-600 text-white hover:bg-green-700"
            : "cursor-not-allowed bg-gray-100 text-gray-400"
        }`}
      >
        <SaveIcon aria-hidden="true" size={15} strokeWidth={1.75} />
      </button>
    </div>
  );
}
