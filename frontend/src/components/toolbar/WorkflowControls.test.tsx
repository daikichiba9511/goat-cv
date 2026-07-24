// @vitest-environment jsdom

import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { ImageMeta, ImageStatus } from "../../types";
import WorkflowControls from "./WorkflowControls";

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

afterEach(cleanup);

it.each([
  ["pending", false, ["Complete annotation", "Escalate"]],
  ["annotated", false, ["Reopen annotation", "Start review", "Escalate"]],
  ["in_review", false, ["Cancel review", "Approve", "Reject", "Escalate"]],
  ["rejected", false, ["Complete revision", "Escalate"]],
  ["approved", false, ["Reopen approval"]],
  ["pending", true, ["Resolve escalation"]],
] satisfies [ImageStatus, boolean, string[]][])(
  "shows only allowed commands for %s/escalated=%s",
  (status, escalated, expectedCommands) => {
    render(
      <WorkflowControls
        image={imageWithWorkflow(status, escalated)}
        busy={false}
        error={null}
        onAction={vi.fn()}
      />,
    );

    const actionMenu = screen.getByRole("menu", { name: "Workflow actions" });
    expect(within(actionMenu).getAllByRole("menuitem").map((item) => item.textContent))
      .toEqual(expectedCommands);
    expect(screen.getByText(new RegExp(status.replace("_", " "), "i"))).toBeTruthy();
    if (escalated) {
      expect(screen.getByText("Escalated")).toBeTruthy();
    } else {
      expect(screen.queryByText("Escalated")).toBeNull();
    }
  },
);
