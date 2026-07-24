import { describe, expect, it, vi } from "vitest";
import type { ImageMeta, ImageStatus, ImageWorkflowEvent } from "./types";
import {
  executeWorkflowTransition,
  workflowCapabilitiesFor,
} from "./workflow";

function imageWithWorkflow(status: ImageStatus, escalated = false): ImageMeta {
  return {
    id: "image-1",
    project_id: "project-1",
    filename: "sample.png",
    original_width: 640,
    original_height: 480,
    width: 640,
    height: 480,
    rotation: 0,
    flip_h: false,
    flip_v: false,
    status,
    escalated,
    uploaded_at: "2026-07-24T00:00:00Z",
  };
}

describe("Image workflow presentation", () => {
  it.each<{
    status: ImageStatus;
    escalated?: boolean;
    graphEditable: boolean;
    transformEditable: boolean;
  }>([
    {
      status: "pending",
      graphEditable: true,
      transformEditable: true,
    },
    {
      status: "annotated",
      graphEditable: false,
      transformEditable: false,
    },
    {
      status: "in_review",
      graphEditable: false,
      transformEditable: false,
    },
    {
      status: "rejected",
      graphEditable: true,
      transformEditable: false,
    },
    {
      status: "approved",
      graphEditable: false,
      transformEditable: false,
    },
    {
      status: "pending",
      escalated: true,
      graphEditable: false,
      transformEditable: false,
    },
  ])("derives editing capability for $status/$escalated", ({
    status,
    escalated = false,
    graphEditable,
    transformEditable,
  }) => {
    const image = imageWithWorkflow(status, escalated);

    expect(workflowCapabilitiesFor(image)).toEqual({
      graphEditable,
      transformEditable,
    });
  });
});

describe("Image workflow transition sequencing", () => {
  it.each<{
    name: string;
    status: ImageStatus;
    event: ImageWorkflowEvent;
  }>([
    {
      name: "complete pending annotation",
      status: "pending",
      event: "annotation_completed",
    },
    {
      name: "escalate editable graph",
      status: "pending",
      event: "escalation_started",
    },
  ])("saves a dirty Graph before it can $name", async ({ status, event }) => {
    const calls: string[] = [];
    const currentImage = imageWithWorkflow(status);
    const transitionedImage = imageWithWorkflow(
      event === "annotation_completed" ? "annotated" : status,
      event === "escalation_started",
    );

    const result = await executeWorkflowTransition({
      image: currentImage,
      event,
      dirty: true,
      saveGraph: vi.fn(async () => {
        calls.push("save");
        return true;
      }),
      applyEvent: vi.fn(async () => {
        calls.push("transition");
        return transitionedImage;
      }),
      refreshImage: vi.fn(),
    });

    expect(calls).toEqual(["save", "transition"]);
    expect(result).toEqual({
      image: transitionedImage,
      error: null,
      transitionSent: true,
    });
  });

  it("keeps the current state and does not transition when Graph save fails", async () => {
    const currentImage = imageWithWorkflow("pending");
    const applyEvent = vi.fn();

    const result = await executeWorkflowTransition({
      image: currentImage,
      event: "annotation_completed",
      dirty: true,
      saveGraph: vi.fn().mockResolvedValue(false),
      applyEvent,
      refreshImage: vi.fn(),
    });

    expect(applyEvent).not.toHaveBeenCalled();
    expect(result).toEqual({
      image: currentImage,
      error: null,
      transitionSent: false,
    });
  });

  it("does not send an action that is not allowed by the displayed workflow state", async () => {
    const currentImage = imageWithWorkflow("approved");
    const applyEvent = vi.fn();

    await expect(executeWorkflowTransition({
      image: currentImage,
      event: "annotation_completed",
      dirty: false,
      saveGraph: vi.fn(),
      applyEvent,
      refreshImage: vi.fn(),
    })).rejects.toThrow("workflow event annotation_completed is not allowed");

    expect(applyEvent).not.toHaveBeenCalled();
  });

  it("refreshes the server state without rolling back the saved Graph after a conflict", async () => {
    const currentImage = imageWithWorkflow("pending");
    const serverImage = imageWithWorkflow("annotated");
    const refreshImage = vi.fn().mockResolvedValue(serverImage);

    const result = await executeWorkflowTransition({
      image: currentImage,
      event: "annotation_completed",
      dirty: false,
      saveGraph: vi.fn(),
      applyEvent: vi.fn().mockRejectedValue(new Error("workflow transition not allowed")),
      refreshImage,
    });

    expect(refreshImage).toHaveBeenCalledWith(currentImage.id);
    expect(result).toEqual({
      image: serverImage,
      error: "workflow transition not allowed",
      transitionSent: true,
    });
  });
});
