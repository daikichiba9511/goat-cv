import { useEffect, useState, type FormEvent } from "react";
import { type Components } from "react-markdown";
import {
  BookOpen,
  FilePlus2,
  Pencil,
  Save,
  Trash2,
  X,
} from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import { useProjectStore } from "../../stores/projectStore";
import SafeMarkdown from "../markdown/SafeMarkdown";

type EditorMode = "view" | "create" | "edit";

const markdownComponents: Components = {
  h1: ({ children }) => <h1 className="mb-3 text-lg font-semibold text-gray-950">{children}</h1>,
  h2: ({ children }) => <h2 className="mb-2 mt-5 text-base font-semibold text-gray-900">{children}</h2>,
  h3: ({ children }) => <h3 className="mb-2 mt-4 text-sm font-semibold text-gray-900">{children}</h3>,
  p: ({ children }) => <p className="mb-3 text-sm leading-6 text-gray-700">{children}</p>,
  ul: ({ children }) => <ul className="mb-3 list-disc space-y-1 pl-5 text-sm leading-6 text-gray-700">{children}</ul>,
  ol: ({ children }) => <ol className="mb-3 list-decimal space-y-1 pl-5 text-sm leading-6 text-gray-700">{children}</ol>,
  li: ({ children }) => <li>{children}</li>,
  blockquote: ({ children }) => (
    <blockquote className="mb-3 border-l-2 border-amber-500 bg-amber-50 px-3 py-2 text-sm text-gray-700">
      {children}
    </blockquote>
  ),
  a: ({ href, children }) => href ? (
    <a
      href={href}
      target="_blank"
      rel="noreferrer noopener"
      className="text-blue-700 underline decoration-blue-300 underline-offset-2 hover:text-blue-900"
    >
      {children}
    </a>
  ) : <span>{children}</span>,
  code: ({ children }) => (
    <code className="break-words bg-gray-100 px-1 py-0.5 font-mono text-xs text-gray-800">
      {children}
    </code>
  ),
  pre: ({ children }) => (
    <pre className="mb-3 overflow-x-auto border bg-gray-950 p-3 text-xs leading-5 text-gray-100">
      {children}
    </pre>
  ),
  table: ({ children }) => (
    <div className="mb-3 overflow-x-auto">
      <table className="w-full border-collapse text-left text-xs">{children}</table>
    </div>
  ),
  th: ({ children }) => <th className="border bg-gray-100 px-2 py-1.5 font-semibold">{children}</th>,
  td: ({ children }) => <td className="border px-2 py-1.5 align-top">{children}</td>,
};

// GuidelinePanel keeps Project manuals available without changing Canvas editing state.
export default function GuidelinePanel() {
  const {
    currentProjectId,
    guidelines,
    createGuideline,
    updateGuideline,
    deleteGuideline,
  } = useProjectStore(useShallow((state) => ({
    currentProjectId: state.currentProject?.id ?? null,
    guidelines: state.guidelines,
    createGuideline: state.createGuideline,
    updateGuideline: state.updateGuideline,
    deleteGuideline: state.deleteGuideline,
  })));
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [editorMode, setEditorMode] = useState<EditorMode>("view");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [displayOrder, setDisplayOrder] = useState("0");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Why: Projectを切り替えた後に、直前のProjectの選択や編集中本文を表示・保存させない。
    setSelectedId(null);
    setEditorMode("view");
    setTitle("");
    setBody("");
    setDisplayOrder("0");
    setError(null);
  }, [currentProjectId]);

  useEffect(() => {
    if (selectedId && guidelines.some((guideline) => guideline.id === selectedId)) {
      return;
    }
    setSelectedId(guidelines[0]?.id ?? null);
  }, [guidelines, selectedId]);

  const selectedGuideline = guidelines.find((guideline) => guideline.id === selectedId) ?? null;
  const parsedDisplayOrder = Number(displayOrder);
  const canSave = title.trim().length > 0 && displayOrder.trim().length > 0 &&
    Number.isInteger(parsedDisplayOrder) && parsedDisplayOrder >= 0;

  const startCreate = () => {
    const nextOrder = guidelines.reduce(
      (maximumOrder, guideline) => Math.max(maximumOrder, guideline.display_order + 1),
      0,
    );
    setTitle("");
    setBody("");
    setDisplayOrder(String(nextOrder));
    setError(null);
    setEditorMode("create");
  };

  const startEdit = () => {
    if (!selectedGuideline) return;
    setTitle(selectedGuideline.title);
    setBody(selectedGuideline.body);
    setDisplayOrder(String(selectedGuideline.display_order));
    setError(null);
    setEditorMode("edit");
  };

  const cancelEdit = () => {
    setError(null);
    setEditorMode("view");
  };

  const saveGuideline = async (event: FormEvent) => {
    event.preventDefault();
    if (!canSave || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      if (editorMode === "edit" && selectedGuideline) {
        const updated = await updateGuideline(
          selectedGuideline.id,
          title.trim(),
          body,
          parsedDisplayOrder,
        );
        setSelectedId(updated.id);
      } else {
        const created = await createGuideline(title.trim(), body, parsedDisplayOrder);
        setSelectedId(created.id);
      }
      setEditorMode("view");
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : "Failed to save guideline");
    } finally {
      setSubmitting(false);
    }
  };

  const removeSelectedGuideline = async () => {
    if (!selectedGuideline || submitting) return;
    if (!window.confirm(`Delete "${selectedGuideline.title}"?`)) return;
    setSubmitting(true);
    setError(null);
    try {
      await deleteGuideline(selectedGuideline.id);
      setEditorMode("view");
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : "Failed to delete guideline");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-11 flex-shrink-0 items-center justify-between border-b px-3">
        <span className="inline-flex min-w-0 items-center gap-2 text-xs font-semibold text-gray-800">
          <BookOpen aria-hidden="true" size={15} strokeWidth={1.75} />
          Guidelines
        </span>
        <button
          type="button"
          aria-label="Add guideline"
          title="Add guideline"
          onClick={startCreate}
          disabled={submitting || editorMode !== "view"}
          className="flex h-8 w-8 items-center justify-center rounded border text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-300"
        >
          <FilePlus2 aria-hidden="true" size={15} strokeWidth={1.75} />
        </button>
      </div>

      {editorMode === "view" && guidelines.length > 0 && (
        <nav aria-label="Guideline pages" className="max-h-36 flex-shrink-0 overflow-y-auto border-b">
          {guidelines.map((guideline) => (
            <button
              key={guideline.id}
              type="button"
              onClick={() => setSelectedId(guideline.id)}
              className={`flex w-full items-center gap-2 border-b px-3 py-2 text-left last:border-b-0 ${
                selectedId === guideline.id
                  ? "bg-blue-50 text-blue-900"
                  : "text-gray-700 hover:bg-gray-50"
              }`}
            >
              <span className="w-5 flex-shrink-0 text-right text-[10px] tabular-nums text-gray-400">
                {guideline.display_order}
              </span>
              <span className="min-w-0 flex-1 truncate text-xs font-medium">{guideline.title}</span>
            </button>
          ))}
        </nav>
      )}

      {editorMode === "view" ? (
        selectedGuideline ? (
          <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex flex-shrink-0 items-start gap-2 border-b px-3 py-2.5">
              <h2 className="min-w-0 flex-1 break-words text-sm font-semibold text-gray-900">
                {selectedGuideline.title}
              </h2>
              <button
                type="button"
                aria-label="Edit guideline"
                title="Edit guideline"
                onClick={startEdit}
                disabled={submitting}
                className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded text-gray-500 hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:text-gray-300"
              >
                <Pencil aria-hidden="true" size={14} strokeWidth={1.75} />
              </button>
              <button
                type="button"
                aria-label="Delete guideline"
                title="Delete guideline"
                onClick={removeSelectedGuideline}
                disabled={submitting}
                className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded text-gray-500 hover:bg-red-50 hover:text-red-700 disabled:cursor-not-allowed disabled:text-gray-300"
              >
                <Trash2 aria-hidden="true" size={14} strokeWidth={1.75} />
              </button>
            </div>
            {error && <p role="alert" className="border-b bg-red-50 px-3 py-2 text-xs text-red-700">{error}</p>}
            <article className="min-h-0 flex-1 overflow-y-auto px-3 py-4">
              <SafeMarkdown body={selectedGuideline.body} components={markdownComponents} />
            </article>
          </div>
        ) : (
          <div className="flex flex-1 items-center justify-center px-4 text-center text-xs text-gray-400">
            No guidelines
          </div>
        )
      ) : (
        <form onSubmit={saveGuideline} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3">
            <label className="block text-xs font-medium text-gray-700">
              Title
              <input
                aria-label="Guideline title"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                className="mt-1 w-full rounded border px-2 py-1.5 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </label>
            <label className="block text-xs font-medium text-gray-700">
              Display order
              <input
                type="number"
                min={0}
                step={1}
                aria-label="Guideline display order"
                value={displayOrder}
                onChange={(event) => setDisplayOrder(event.target.value)}
                className="mt-1 w-24 rounded border px-2 py-1.5 text-sm tabular-nums outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </label>
            <label className="block text-xs font-medium text-gray-700">
              Markdown
              <textarea
                aria-label="Guideline body"
                value={body}
                onChange={(event) => setBody(event.target.value)}
                rows={14}
                className="mt-1 w-full resize-y rounded border px-2 py-2 font-mono text-xs leading-5 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </label>
            {error && <p role="alert" className="text-xs text-red-700">{error}</p>}
          </div>
          <div className="flex flex-shrink-0 justify-end gap-2 border-t p-3">
            <button
              type="button"
              aria-label="Cancel guideline edit"
              title="Cancel"
              onClick={cancelEdit}
              disabled={submitting}
              className="flex h-8 w-8 items-center justify-center rounded border text-gray-600 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-300"
            >
              <X aria-hidden="true" size={16} strokeWidth={1.75} />
            </button>
            <button
              type="submit"
              aria-label="Save guideline"
              title="Save guideline"
              disabled={!canSave || submitting}
              className="flex h-8 w-8 items-center justify-center rounded border border-blue-700 bg-blue-700 text-white hover:bg-blue-800 disabled:cursor-not-allowed disabled:border-gray-200 disabled:bg-gray-100 disabled:text-gray-300"
            >
              <Save aria-hidden="true" size={15} strokeWidth={1.75} />
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
