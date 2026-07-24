import type { ImageMeta, ImageStatus, ImageWorkflowEvent } from "./types";

export type ImageWorkflowAction = {
  event: ImageWorkflowEvent;
  label: string;
};

export type ImageWorkflowCapabilities = {
  graphEditable: boolean;
  transformEditable: boolean;
};

type ExecuteWorkflowTransitionInput = {
  image: ImageMeta;
  event: ImageWorkflowEvent;
  dirty: boolean;
  saveGraph: () => Promise<boolean>;
  applyEvent: (imageId: string, event: ImageWorkflowEvent) => Promise<ImageMeta>;
  refreshImage: (imageId: string) => Promise<ImageMeta>;
};

type ExecuteWorkflowTransitionResult = {
  image: ImageMeta;
  error: string | null;
  transitionSent: boolean;
};

const ACTION_BY_EVENT: Record<ImageWorkflowEvent, ImageWorkflowAction> = {
  annotation_completed: {
    event: "annotation_completed",
    label: "Complete annotation",
  },
  annotation_reopened: {
    event: "annotation_reopened",
    label: "Reopen annotation",
  },
  review_started: {
    event: "review_started",
    label: "Start review",
  },
  review_cancelled: {
    event: "review_cancelled",
    label: "Cancel review",
  },
  review_approved: {
    event: "review_approved",
    label: "Approve",
  },
  review_rejected: {
    event: "review_rejected",
    label: "Reject",
  },
  approval_reopened: {
    event: "approval_reopened",
    label: "Reopen approval",
  },
  escalation_started: {
    event: "escalation_started",
    label: "Escalate",
  },
  escalation_resolved: {
    event: "escalation_resolved",
    label: "Resolve escalation",
  },
};

const EVENTS_BY_STATUS: Record<ImageStatus, ImageWorkflowEvent[]> = {
  pending: ["annotation_completed", "escalation_started"],
  annotated: ["annotation_reopened", "review_started", "escalation_started"],
  in_review: [
    "review_cancelled",
    "review_approved",
    "review_rejected",
    "escalation_started",
  ],
  rejected: ["annotation_completed", "escalation_started"],
  approved: ["approval_reopened"],
};

// workflowActionsFor returns only the commands allowed by the canonical workflow state table.
export function workflowActionsFor(image: ImageMeta): ImageWorkflowAction[] {
  const events = image.escalated
    ? ["escalation_resolved" as const]
    : EVENTS_BY_STATUS[image.status];
  return events.map((event) => {
    if (event === "annotation_completed" && image.status === "rejected") {
      return { event, label: "Complete revision" };
    }
    return ACTION_BY_EVENT[event];
  });
}

// workflowCapabilitiesFor describes which persisted image operations the current state permits.
export function workflowCapabilitiesFor(image: ImageMeta): ImageWorkflowCapabilities {
  if (image.escalated) {
    return { graphEditable: false, transformEditable: false };
  }
  return {
    graphEditable: image.status === "pending" || image.status === "rejected",
    transformEditable: image.status === "pending",
  };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

// executeWorkflowTransition preserves the required Graph-save-before-transition ordering.
export async function executeWorkflowTransition({
  image,
  event,
  dirty,
  saveGraph,
  applyEvent,
  refreshImage,
}: ExecuteWorkflowTransitionInput): Promise<ExecuteWorkflowTransitionResult> {
  const eventIsAllowed = workflowActionsFor(image).some((action) => action.event === event);
  if (!eventIsAllowed) {
    throw new Error(`workflow event ${event} is not allowed from the current state`);
  }

  const { graphEditable } = workflowCapabilitiesFor(image);
  const saveBeforeTransition = dirty && graphEditable && (
    event === "annotation_completed" || event === "escalation_started"
  );
  if (saveBeforeTransition && !await saveGraph()) {
    // Why: 保存失敗時はeventを送らず、Storeが保持する未保存Graphと現在状態をそのまま残す。
    return { image, error: null, transitionSent: false };
  }

  try {
    const transitionedImage = await applyEvent(image.id, event);
    return { image: transitionedImage, error: null, transitionSent: true };
  } catch (error) {
    // Why: 競合後は保存済みGraphを再読込せず、workflow状態だけをserverの現在値へ合わせる。
    const refreshedImage = await refreshImage(image.id);
    return {
      image: refreshedImage,
      error: errorMessage(error),
      transitionSent: true,
    };
  }
}
