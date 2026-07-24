// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import * as api from "../../api/client";
import { useProjectStore } from "../../stores/projectStore";
import type { Project } from "../../types";
import Sidebar from "./Sidebar";

vi.mock("../../api/client", () => ({
  listImages: vi.fn(),
}));

const project: Project = {
  id: "project-1",
  name: "Vision dataset",
  created_at: "2026-07-24T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(api.listImages).mockResolvedValue({ items: [] });
  useProjectStore.setState({
    currentProject: project,
    images: [],
    imageFilters: {},
  });
});

afterEach(cleanup);

it("combines lifecycle and escalation filters selected in the image list", async () => {
  render(
    <Sidebar
      images={[]}
      currentImageId={null}
      onSelectImage={vi.fn()}
    />,
  );

  fireEvent.change(screen.getByLabelText("Filter images by lifecycle"), {
    target: { value: "rejected" },
  });
  fireEvent.change(screen.getByLabelText("Filter images by escalation"), {
    target: { value: "true" },
  });

  await waitFor(() => {
    expect(api.listImages).toHaveBeenLastCalledWith(project.id, {
      status: "rejected",
      escalated: true,
    });
  });
});
