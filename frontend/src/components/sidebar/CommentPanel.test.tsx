// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../../api/client";
import { useAnnotationStore } from "../../stores/annotationStore";
import type { QAComment } from "../../types";
import CommentPanel from "./CommentPanel";

vi.mock("../../api/client", () => ({
  createComment: vi.fn(),
  listComments: vi.fn(),
  setCommentResolved: vi.fn(),
  deleteComment: vi.fn(),
}));

const imageComment: QAComment = {
  id: "comment-image",
  image_id: "image-1",
  annotation_id: null,
  author: "reviewer",
  body: "Check the whole image",
  type: "question",
  resolved: false,
  target_deleted: false,
  created_at: "2026-07-24T00:00:00Z",
  updated_at: "2026-07-24T00:00:00Z",
};

const annotationComment: QAComment = {
  ...imageComment,
  id: "comment-annotation",
  annotation_id: "annotation-1",
  author: "annotator",
  body: "Fix this box\n\n<script>alert(1)</script>\n\n![pixel](https://example.com/pixel.png)",
  type: "issue",
};

describe("CommentPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.listComments).mockResolvedValue({
      items: [imageComment, annotationComment],
    });
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations: [{
        id: "annotation-1",
        image_id: "image-1",
        type: "bbox",
        coordinates: { x: 0, y: 0, width: 0.5, height: 0.5 },
        label_id: null,
        created_at: "2026-07-24T00:00:00Z",
      }],
      selectedId: "annotation-1",
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("filters by the selected Annotation and creates a linked Comment", async () => {
    const created: QAComment = {
      ...annotationComment,
      id: "comment-created",
      body: "Needs a tighter box",
      author: "qa-user",
    };
    vi.mocked(api.createComment).mockResolvedValue(created);
    const { container } = render(<CommentPanel imageId="image-1" />);

    expect(await screen.findByText("Fix this box")).toBeTruthy();
    expect(screen.queryByText("Check the whole image")).toBeNull();
    const scope = screen.getByRole("group", { name: "Comment scope" });
    fireEvent.click(within(scope).getByRole("button", { name: "All" }));
    expect(screen.getByText("Check the whole image")).toBeTruthy();
    fireEvent.click(within(scope).getByRole("button", { name: "Selected" }));

    expect(screen.queryByText("Check the whole image")).toBeNull();
    expect(screen.getByText("Fix this box")).toBeTruthy();
    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("img")).toBeNull();

    fireEvent.change(screen.getByLabelText("Author"), { target: { value: "qa-user" } });
    fireEvent.change(screen.getByLabelText("Comment type"), { target: { value: "issue" } });
    fireEvent.click(screen.getByRole("radio", { name: /Selected #001/ }));
    fireEvent.change(screen.getByLabelText("Comment body"), {
      target: { value: "Needs a tighter box" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add comment" }));

    await waitFor(() => {
      expect(api.createComment).toHaveBeenCalledWith("image-1", {
        annotation_id: "annotation-1",
        author: "qa-user",
        body: "Needs a tighter box",
        type: "issue",
      });
    });
    expect(await screen.findByText("Needs a tighter box")).toBeTruthy();
  });

  it("resolves, reopens, and deletes a Comment", async () => {
    vi.mocked(api.setCommentResolved)
      .mockResolvedValueOnce({ ...imageComment, resolved: true })
      .mockResolvedValueOnce(imageComment);
    vi.mocked(api.deleteComment).mockResolvedValue(undefined);
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<CommentPanel imageId="image-1" />);

    const scope = screen.getByRole("group", { name: "Comment scope" });
    fireEvent.click(within(scope).getByRole("button", { name: "All" }));
    const resolveToggle = await screen.findByRole("checkbox", {
      name: "Mark comment by reviewer as resolved",
    });
    fireEvent.click(resolveToggle);
    await waitFor(() => {
      expect(api.setCommentResolved).toHaveBeenCalledWith("image-1", imageComment.id, true);
    });

    fireEvent.click(resolveToggle);
    await waitFor(() => {
      expect(api.setCommentResolved).toHaveBeenCalledWith("image-1", imageComment.id, false);
    });

    fireEvent.click(screen.getByRole("button", { name: "Delete comment by reviewer" }));
    await waitFor(() => {
      expect(api.deleteComment).toHaveBeenCalledWith("image-1", imageComment.id);
    });
    expect(screen.queryByText(imageComment.body)).toBeNull();
  });
});
