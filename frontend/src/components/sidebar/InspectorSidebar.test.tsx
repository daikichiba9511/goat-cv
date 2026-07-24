// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import InspectorSidebar from "./InspectorSidebar";

beforeEach(() => {
  Object.defineProperty(Element.prototype, "scrollIntoView", {
    configurable: true,
    value: vi.fn(),
  });
  useProjectStore.setState({ guidelines: [] });
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
    polygonDraftPoints: [{ x: 0.1, y: 0.2 }],
    dirty: true,
  });
});

afterEach(() => {
  cleanup();
});

it("keeps Canvas editing state while the Guidelines tab and inspector are toggled", () => {
  render(
    <InspectorSidebar
      labels={[]}
      activeLabel={null}
      onSelectLabel={() => undefined}
      onSelectAnnotation={() => undefined}
    />,
  );

  fireEvent.click(screen.getByRole("tab", { name: "Guide" }));
  fireEvent.click(screen.getByRole("button", { name: "Close inspector" }));
  fireEvent.click(screen.getByRole("button", { name: "Open inspector" }));

  const state = useAnnotationStore.getState();
  expect(state.selectedId).toBe("annotation-1");
  expect(state.polygonDraftPoints).toEqual([{ x: 0.1, y: 0.2 }]);
  expect(state.dirty).toBe(true);
  expect(screen.getByRole("tab", { name: "Guide" }).getAttribute("aria-selected")).toBe("true");
});
