import { useRef } from "react";
import {
  CheckCircle2,
  ChevronDown,
  CircleDot,
  GitBranch,
  LoaderCircle,
  TriangleAlert,
} from "lucide-react";
import type { ImageMeta, ImageStatus, ImageWorkflowEvent } from "../../types";
import { workflowActionsFor } from "../../workflow";

type Props = {
  image: ImageMeta;
  busy: boolean;
  error: string | null;
  onAction: (event: ImageWorkflowEvent) => void;
};

const STATUS_LABELS: Record<ImageStatus, string> = {
  pending: "Pending",
  annotated: "Annotated",
  in_review: "In review",
  rejected: "Rejected",
  approved: "Approved",
};

const STATUS_STYLES: Record<ImageStatus, string> = {
  pending: "border-gray-300 bg-gray-50 text-gray-700",
  annotated: "border-blue-200 bg-blue-50 text-blue-700",
  in_review: "border-amber-200 bg-amber-50 text-amber-800",
  rejected: "border-red-200 bg-red-50 text-red-700",
  approved: "border-green-200 bg-green-50 text-green-700",
};

export default function WorkflowControls({ image, busy, error, onAction }: Props) {
  const actionMenuRef = useRef<HTMLDetailsElement>(null);
  const actions = workflowActionsFor(image);

  const handleAction = (event: ImageWorkflowEvent) => {
    actionMenuRef.current?.removeAttribute("open");
    onAction(event);
  };

  return (
    <div aria-label="Image workflow" className="flex min-w-0 flex-shrink-0 items-center gap-1.5">
      <span
        aria-label={`Lifecycle status: ${STATUS_LABELS[image.status]}`}
        className={`inline-flex h-7 items-center gap-1 border px-2 text-xs font-medium ${STATUS_STYLES[image.status]}`}
      >
        <CircleDot aria-hidden="true" size={12} strokeWidth={2} />
        <span>{STATUS_LABELS[image.status]}</span>
      </span>

      <span
        aria-label={`Escalation: ${image.escalated ? "Escalated" : "Clear"}`}
        title={image.escalated ? "Escalated" : "Escalation clear"}
        className={`inline-flex h-7 items-center gap-1 border px-2 text-xs font-medium ${
          image.escalated
            ? "border-orange-300 bg-orange-50 text-orange-800"
            : "border-gray-200 bg-white text-gray-500"
        }`}
      >
        {image.escalated ? (
          <TriangleAlert aria-hidden="true" size={12} strokeWidth={2} />
        ) : (
          <CheckCircle2 aria-hidden="true" size={12} strokeWidth={1.75} />
        )}
        <span className="hidden xl:inline">
          {image.escalated ? "Escalated" : "Clear"}
        </span>
      </span>

      <details ref={actionMenuRef} className="relative">
        <summary
          aria-label="Open workflow actions"
          className="flex h-7 cursor-pointer list-none items-center gap-1 border bg-white px-2 text-xs font-medium text-gray-700 hover:bg-gray-50"
        >
          {busy ? (
            <LoaderCircle aria-hidden="true" className="animate-spin" size={13} strokeWidth={1.75} />
          ) : (
            <GitBranch aria-hidden="true" size={13} strokeWidth={1.75} />
          )}
          <span className="hidden xl:inline">Workflow</span>
          <ChevronDown aria-hidden="true" size={12} strokeWidth={1.75} />
        </summary>
        <div
          role="menu"
          aria-label="Workflow actions"
          className="absolute right-0 top-8 z-40 w-48 border bg-white py-1 shadow-lg"
        >
          {actions.map((action) => (
            <button
              key={action.event}
              type="button"
              role="menuitem"
              disabled={busy}
              onClick={() => handleAction(action.event)}
              className="block w-full px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:text-gray-400"
            >
              {action.label}
            </button>
          ))}
        </div>
      </details>

      {error && (
        <span role="alert" title={error} className="max-w-40 truncate text-xs text-red-600">
          {error}
        </span>
      )}
    </div>
  );
}
