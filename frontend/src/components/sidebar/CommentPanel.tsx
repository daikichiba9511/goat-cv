import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { type Components } from "react-markdown";
import {
  BoxSelect,
  CircleAlert,
  CircleHelp,
  Image,
  MessageSquareText,
  RefreshCw,
  Send,
  StickyNote,
  Trash2,
} from "lucide-react";
import { useShallow } from "zustand/react/shallow";
import * as api from "../../api/client";
import { useAnnotationStore } from "../../stores/annotationStore";
import type { CommentType, QAComment } from "../../types";
import SafeMarkdown from "../markdown/SafeMarkdown";

type Props = {
  imageId: string | null;
};

type CommentScope = "all" | "image" | "selected";
type CommentTarget = "image" | "annotation";

const commentMarkdownComponents: Components = {
  h1: ({ children }) => <h1 className="mb-2 text-sm font-semibold text-gray-950">{children}</h1>,
  h2: ({ children }) => <h2 className="mb-2 text-sm font-semibold text-gray-900">{children}</h2>,
  h3: ({ children }) => <h3 className="mb-1 text-xs font-semibold text-gray-900">{children}</h3>,
  p: ({ children }) => <p className="mb-2 break-words text-xs leading-5 text-gray-700">{children}</p>,
  ul: ({ children }) => <ul className="mb-2 list-disc space-y-1 pl-4 text-xs leading-5 text-gray-700">{children}</ul>,
  ol: ({ children }) => <ol className="mb-2 list-decimal space-y-1 pl-4 text-xs leading-5 text-gray-700">{children}</ol>,
  li: ({ children }) => <li>{children}</li>,
  blockquote: ({ children }) => (
    <blockquote className="mb-2 border-l-2 border-amber-500 bg-amber-50 px-2 py-1 text-xs text-gray-700">
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
    <code className="break-words bg-gray-100 px-1 py-0.5 font-mono text-[11px] text-gray-800">
      {children}
    </code>
  ),
  pre: ({ children }) => (
    <pre className="mb-2 overflow-x-auto bg-gray-950 p-2 text-[11px] leading-5 text-gray-100">
      {children}
    </pre>
  ),
};

const typePresentation = {
  question: {
    label: "Question",
    icon: CircleHelp,
    className: "bg-amber-50 text-amber-800",
  },
  issue: {
    label: "Issue",
    icon: CircleAlert,
    className: "bg-red-50 text-red-700",
  },
  note: {
    label: "Note",
    icon: StickyNote,
    className: "bg-gray-100 text-gray-700",
  },
} satisfies Record<CommentType, {
  label: string;
  icon: typeof CircleHelp;
  className: string;
}>;

// CommentPanel manages QA Comments for the current Image and selected Annotation.
export default function CommentPanel({ imageId }: Props) {
  const {
    loadedImageId,
    annotations,
    selectedId,
  } = useAnnotationStore(useShallow((state) => ({
    loadedImageId: state.loadedImageId,
    annotations: state.annotations,
    selectedId: state.selectedId,
  })));
  const [comments, setComments] = useState<QAComment[]>([]);
  const [scope, setScope] = useState<CommentScope>(() =>
    loadedImageId === imageId && selectedId && !selectedId.startsWith("temp-")
      ? "selected"
      : "all"
  );
  const [target, setTarget] = useState<CommentTarget>("image");
  const [author, setAuthor] = useState("");
  const [body, setBody] = useState("");
  const [commentType, setCommentType] = useState<CommentType>("question");
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [pendingCommentId, setPendingCommentId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const loadSequence = useRef(0);
  const mutationSequence = useRef(0);
  const previousImageId = useRef(imageId);
  const previousSelectedId = useRef(selectedId);

  const selectedAnnotation = loadedImageId === imageId
    ? annotations.find((annotation) => annotation.id === selectedId) ?? null
    : null;
  const selectedPosition = selectedAnnotation
    ? annotations.findIndex((annotation) => annotation.id === selectedAnnotation.id) + 1
    : 0;
  const canTargetSelected = selectedAnnotation !== null
    && !selectedAnnotation.id.startsWith("temp-");

  const loadComments = useCallback(async () => {
    const sequence = ++loadSequence.current;
    if (!imageId) {
      setComments([]);
      setLoading(false);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const response = await api.listComments(imageId);
      if (loadSequence.current === sequence) {
        setComments(response.items);
      }
    } catch (loadError) {
      if (loadSequence.current === sequence) {
        setError(loadError instanceof Error ? loadError.message : "Failed to load comments");
      }
    } finally {
      if (loadSequence.current === sequence) {
        setLoading(false);
      }
    }
  }, [imageId]);

  useEffect(() => {
    void loadComments();
    return () => {
      loadSequence.current += 1;
    };
  }, [loadComments]);

  useEffect(() => {
    if (imageId !== previousImageId.current) {
      // Why: Image切替後に直前のImage向け本文や処理中表示を引き継がない。
      mutationSequence.current += 1;
      previousImageId.current = imageId;
      setComments([]);
      setBody("");
      setTarget("image");
      setScope("all");
      setSubmitting(false);
      setPendingCommentId(null);
    }
    if (selectedId !== previousSelectedId.current) {
      setScope(canTargetSelected ? "selected" : "all");
      previousSelectedId.current = selectedId;
    }
    if (!canTargetSelected && target === "annotation") {
      setTarget("image");
    }
    if (!selectedAnnotation && scope === "selected") {
      setScope("all");
    }
  }, [canTargetSelected, imageId, scope, selectedAnnotation, selectedId, target]);

  const visibleComments = useMemo(() => comments.filter((comment) => {
    if (scope === "image") return comment.annotation_id === null;
    if (scope === "selected") return comment.annotation_id === selectedAnnotation?.id;
    return true;
  }), [comments, scope, selectedAnnotation]);
  const positionByAnnotationId = useMemo(
    () => new Map(annotations.map((annotation, index) => [annotation.id, index + 1])),
    [annotations],
  );
  const canSubmit = imageId !== null
    && author.trim().length > 0
    && body.trim().length > 0
    && (target === "image" || canTargetSelected);
  const busy = loading || submitting || pendingCommentId !== null;

  const addComment = async (event: FormEvent) => {
    event.preventDefault();
    if (!imageId || !canSubmit || busy) return;
    const sequence = ++mutationSequence.current;
    setSubmitting(true);
    setError(null);
    try {
      const created = await api.createComment(imageId, {
        annotation_id: target === "annotation" ? selectedAnnotation?.id ?? null : null,
        author: author.trim(),
        body,
        type: commentType,
      });
      if (mutationSequence.current === sequence) {
        setComments((currentComments) => [...currentComments, created]);
        setBody("");
      }
    } catch (createError) {
      if (mutationSequence.current === sequence) {
        setError(createError instanceof Error ? createError.message : "Failed to add comment");
      }
    } finally {
      if (mutationSequence.current === sequence) {
        setSubmitting(false);
      }
    }
  };

  const changeResolved = async (comment: QAComment, resolved: boolean) => {
    if (!imageId || busy) return;
    const sequence = ++mutationSequence.current;
    setPendingCommentId(comment.id);
    setError(null);
    try {
      const updated = await api.setCommentResolved(imageId, comment.id, resolved);
      if (mutationSequence.current === sequence) {
        setComments((currentComments) => currentComments.map((currentComment) =>
          currentComment.id === updated.id ? updated : currentComment,
        ));
      }
    } catch (updateError) {
      if (mutationSequence.current === sequence) {
        setError(updateError instanceof Error ? updateError.message : "Failed to update comment");
      }
    } finally {
      if (mutationSequence.current === sequence) {
        setPendingCommentId(null);
      }
    }
  };

  const removeComment = async (comment: QAComment) => {
    if (!imageId || busy) return;
    if (!window.confirm(`Delete comment by ${comment.author}?`)) return;
    const sequence = ++mutationSequence.current;
    setPendingCommentId(comment.id);
    setError(null);
    try {
      await api.deleteComment(imageId, comment.id);
      if (mutationSequence.current === sequence) {
        setComments((currentComments) => currentComments.filter(
          (currentComment) => currentComment.id !== comment.id,
        ));
      }
    } catch (deleteError) {
      if (mutationSequence.current === sequence) {
        setError(deleteError instanceof Error ? deleteError.message : "Failed to delete comment");
      }
    } finally {
      if (mutationSequence.current === sequence) {
        setPendingCommentId(null);
      }
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col bg-white">
      <div className="flex h-11 flex-shrink-0 items-center justify-between border-b px-3">
        <span className="inline-flex min-w-0 items-center gap-2 text-xs font-semibold text-gray-800">
          <MessageSquareText aria-hidden="true" size={15} strokeWidth={1.75} />
          Comments
          <span className="font-normal tabular-nums text-gray-400">{comments.length}</span>
        </span>
        <button
          type="button"
          aria-label="Reload comments"
          title="Reload comments"
          onClick={() => void loadComments()}
          disabled={busy || !imageId}
          className="flex h-8 w-8 items-center justify-center rounded text-gray-500 hover:bg-gray-100 hover:text-gray-800 disabled:cursor-not-allowed disabled:text-gray-300"
        >
          <RefreshCw aria-hidden="true" size={14} strokeWidth={1.75} />
        </button>
      </div>

      <div role="group" aria-label="Comment scope" className="grid h-9 flex-shrink-0 grid-cols-3 border-b bg-gray-50 p-1">
        {(["all", "image", "selected"] as const).map((scopeOption) => (
          <button
            key={scopeOption}
            type="button"
            aria-pressed={scope === scopeOption}
            disabled={scopeOption === "selected" && !selectedAnnotation}
            onClick={() => setScope(scopeOption)}
            className={`text-[11px] font-medium ${
              scope === scopeOption
                ? "bg-white text-gray-900 shadow-sm"
                : "text-gray-500 hover:text-gray-800 disabled:text-gray-300"
            }`}
          >
            {scopeOption === "all" ? "All" : scopeOption === "image" ? "Image" : "Selected"}
          </button>
        ))}
      </div>

      <form onSubmit={addComment} className="flex-shrink-0 space-y-2 border-b p-3">
        <div className="grid grid-cols-[1fr_6.5rem] gap-2">
          <label className="block text-[11px] font-medium text-gray-600">
            Author
            <input
              aria-label="Author"
              value={author}
              onChange={(event) => setAuthor(event.target.value)}
              className="mt-1 w-full rounded border px-2 py-1.5 text-xs text-gray-800 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
          </label>
          <label className="block text-[11px] font-medium text-gray-600">
            Type
            <select
              aria-label="Comment type"
              value={commentType}
              onChange={(event) => setCommentType(event.target.value as CommentType)}
              className="mt-1 w-full rounded border bg-white px-2 py-1.5 text-xs text-gray-700 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            >
              <option value="question">Question</option>
              <option value="issue">Issue</option>
              <option value="note">Note</option>
            </select>
          </label>
        </div>
        <fieldset>
          <legend className="mb-1 text-[11px] font-medium text-gray-600">Target</legend>
          <div className="grid grid-cols-2 border">
            <label className={`flex h-8 items-center justify-center gap-1 text-[11px] ${
              target === "image" ? "bg-blue-50 font-medium text-blue-800" : "text-gray-600"
            }`}>
              <input
                type="radio"
                name="comment-target"
                value="image"
                checked={target === "image"}
                onChange={() => setTarget("image")}
                className="sr-only"
              />
              <Image aria-hidden="true" size={13} strokeWidth={1.75} />
              Image
            </label>
            <label className={`flex h-8 items-center justify-center gap-1 border-l text-[11px] ${
              target === "annotation"
                ? "bg-blue-50 font-medium text-blue-800"
                : canTargetSelected ? "text-gray-600" : "text-gray-300"
            }`}>
              <input
                type="radio"
                name="comment-target"
                value="annotation"
                checked={target === "annotation"}
                disabled={!canTargetSelected}
                onChange={() => setTarget("annotation")}
                className="sr-only"
              />
              <BoxSelect aria-hidden="true" size={13} strokeWidth={1.75} />
              {selectedPosition > 0 ? `Selected #${String(selectedPosition).padStart(3, "0")}` : "Selected"}
            </label>
          </div>
        </fieldset>
        <label className="block text-[11px] font-medium text-gray-600">
          Comment
          <textarea
            aria-label="Comment body"
            rows={3}
            value={body}
            onChange={(event) => setBody(event.target.value)}
            className="mt-1 w-full resize-y rounded border px-2 py-1.5 text-xs leading-5 text-gray-800 outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
          />
        </label>
        <button
          type="submit"
          disabled={!canSubmit || busy}
          className="inline-flex h-8 w-full items-center justify-center gap-1.5 rounded bg-blue-600 px-3 text-xs font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-gray-300"
        >
          <Send aria-hidden="true" size={13} strokeWidth={1.75} />
          Add comment
        </button>
      </form>

      {error && <p role="alert" className="flex-shrink-0 border-b bg-red-50 px-3 py-2 text-xs text-red-700">{error}</p>}

      <div role="list" aria-label="Comments" className="min-h-0 flex-1 overflow-y-auto">
        {loading && comments.length === 0 && (
          <div className="px-3 py-8 text-center text-xs text-gray-400">Loading...</div>
        )}
        {!loading && visibleComments.length === 0 && (
          <div className="px-3 py-8 text-center text-xs text-gray-400">No comments</div>
        )}
        {visibleComments.map((comment) => {
          const presentation = typePresentation[comment.type];
          const TypeIcon = presentation.icon;
          const annotationPosition = comment.annotation_id
            ? positionByAnnotationId.get(comment.annotation_id)
            : undefined;
          const targetDeleted = comment.target_deleted || (
            comment.annotation_id !== null
            && loadedImageId === imageId
            && annotationPosition === undefined
          );
          const targetLabel = targetDeleted
            ? "Deleted annotation"
            : comment.annotation_id
              ? annotationPosition ? `Annotation #${String(annotationPosition).padStart(3, "0")}` : "Annotation"
              : "Image";
          return (
            <article key={comment.id} role="listitem" className={`border-b px-3 py-3 ${comment.resolved ? "bg-gray-50" : "bg-white"}`}>
              <div className="mb-2 flex items-start gap-2">
                <span className={`inline-flex h-5 items-center gap-1 px-1.5 text-[10px] font-medium ${presentation.className}`}>
                  <TypeIcon aria-hidden="true" size={11} strokeWidth={1.75} />
                  {presentation.label}
                </span>
                <span className="min-w-0 flex-1 truncate text-xs font-medium text-gray-800" title={comment.author}>
                  {comment.author}
                </span>
                <button
                  type="button"
                  aria-label={`Delete comment by ${comment.author}`}
                  title={`Delete comment by ${comment.author}`}
                  onClick={() => void removeComment(comment)}
                  disabled={busy}
                  className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded text-gray-400 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:text-gray-300"
                >
                  <Trash2 aria-hidden="true" size={13} strokeWidth={1.75} />
                </button>
              </div>
              <div className="[&>*:last-child]:mb-0">
                <SafeMarkdown body={comment.body} components={commentMarkdownComponents} />
              </div>
              <div className="mt-2 flex items-center gap-2 border-t pt-2">
                <span className={`min-w-0 flex-1 truncate text-[10px] ${targetDeleted ? "text-red-600" : "text-gray-400"}`}>
                  {targetLabel}
                </span>
                <label className="inline-flex flex-shrink-0 items-center gap-1 text-[10px] text-gray-600">
                  <input
                    type="checkbox"
                    aria-label={`Mark comment by ${comment.author} as resolved`}
                    checked={comment.resolved}
                    disabled={busy}
                    onChange={(event) => void changeResolved(comment, event.target.checked)}
                  />
                  Resolved
                </label>
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
