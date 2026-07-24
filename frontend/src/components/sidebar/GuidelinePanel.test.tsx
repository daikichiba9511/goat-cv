// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../../api/client";
import { useProjectStore } from "../../stores/projectStore";
import type { Guideline, Project } from "../../types";
import GuidelinePanel from "./GuidelinePanel";

vi.mock("../../api/client", () => ({
  createGuideline: vi.fn(),
  listGuidelines: vi.fn(),
  updateGuideline: vi.fn(),
  deleteGuideline: vi.fn(),
}));

const project: Project = {
  id: "project-1",
  name: "Project One",
  created_at: "2026-07-24T00:00:00Z",
};

const unsafeGuideline: Guideline = {
  id: "guideline-1",
  project_id: project.id,
  title: "Boundary rules",
  body: [
    "# Safe heading",
    "",
    "<script>window.__guidelineXSS = true</script>",
    "",
    "![tracking pixel](https://example.com/pixel.png)",
    "",
    "[unsafe link](javascript:alert(1))",
  ].join("\n"),
  display_order: 0,
  updated_at: "2026-07-24T00:00:00Z",
};

describe("GuidelinePanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useProjectStore.setState({ currentProject: project, guidelines: [] });
  });

  afterEach(() => {
    cleanup();
  });

  it("renders Markdown without executable HTML or embedded images", () => {
    useProjectStore.setState({ guidelines: [unsafeGuideline] });
    const { container } = render(<GuidelinePanel />);

    expect(screen.getByRole("heading", { name: "Safe heading" })).toBeTruthy();
    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("img")).toBeNull();
    const unsafeLink = screen.getByText("unsafe link").closest("a");
    expect(unsafeLink).toBeNull();
  });

  it("creates, edits, and deletes a Guideline from the panel", async () => {
    const created: Guideline = {
      id: "guideline-created",
      project_id: project.id,
      title: "Page",
      body: "# Rules",
      display_order: 2,
      updated_at: "2026-07-24T00:00:00Z",
    };
    vi.mocked(api.createGuideline).mockResolvedValue(created);
    vi.mocked(api.listGuidelines).mockResolvedValueOnce({ items: [created] });

    render(<GuidelinePanel />);
    fireEvent.click(screen.getByRole("button", { name: "Add guideline" }));
    fireEvent.change(screen.getByLabelText("Guideline title"), { target: { value: " Page " } });
    fireEvent.change(screen.getByLabelText("Guideline display order"), { target: { value: "2" } });
    fireEvent.change(screen.getByLabelText("Guideline body"), { target: { value: "# Rules" } });
    fireEvent.click(screen.getByRole("button", { name: "Save guideline" }));

    await waitFor(() => {
      expect(api.createGuideline).toHaveBeenCalledWith(project.id, "Page", "# Rules", 2);
    });
    expect(await screen.findByText("Rules")).toBeTruthy();

    const updated = { ...created, title: "Updated page", body: "Updated body", display_order: 0 };
    vi.mocked(api.updateGuideline).mockResolvedValue(updated);
    vi.mocked(api.listGuidelines).mockResolvedValueOnce({ items: [updated] });
    fireEvent.click(screen.getByRole("button", { name: "Edit guideline" }));
    fireEvent.change(screen.getByLabelText("Guideline title"), { target: { value: updated.title } });
    fireEvent.change(screen.getByLabelText("Guideline display order"), { target: { value: "0" } });
    fireEvent.change(screen.getByLabelText("Guideline body"), { target: { value: updated.body } });
    fireEvent.click(screen.getByRole("button", { name: "Save guideline" }));

    await waitFor(() => {
      expect(api.updateGuideline).toHaveBeenCalledWith(
        project.id,
        created.id,
        updated.title,
        updated.body,
        updated.display_order,
      );
    });
    expect(await screen.findByText(updated.body)).toBeTruthy();

    vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.mocked(api.deleteGuideline).mockResolvedValue(undefined);
    vi.mocked(api.listGuidelines).mockResolvedValueOnce({ items: [] });
    fireEvent.click(screen.getByRole("button", { name: "Delete guideline" }));

    await waitFor(() => {
      expect(api.deleteGuideline).toHaveBeenCalledWith(project.id, created.id);
    });
    expect(await screen.findByText("No guidelines")).toBeTruthy();
  });

  it("discards editing state when the current Project changes", async () => {
    const otherProject = { ...project, id: "project-2", name: "Project Two" };
    const otherGuideline = {
      ...unsafeGuideline,
      id: "guideline-2",
      project_id: otherProject.id,
      title: "Other rules",
      body: "Other body",
    };
    useProjectStore.setState({ guidelines: [unsafeGuideline] });
    render(<GuidelinePanel />);

    fireEvent.click(screen.getByRole("button", { name: "Edit guideline" }));
    fireEvent.change(screen.getByLabelText("Guideline body"), {
      target: { value: "Unsaved previous Project body" },
    });
    useProjectStore.setState({
      currentProject: otherProject,
      guidelines: [otherGuideline],
    });

    expect(await screen.findByRole("heading", { name: "Other rules" })).toBeTruthy();
    expect(screen.queryByDisplayValue("Unsaved previous Project body")).toBeNull();
  });
});
