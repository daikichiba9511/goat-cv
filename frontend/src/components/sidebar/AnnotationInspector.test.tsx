// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Annotation, Edge, LabelDefinition } from "../../types";
import { useAnnotationStore } from "../../stores/annotationStore";
import AnnotationInspector from "./AnnotationInspector";

const labels: LabelDefinition[] = [
  {
    id: "label-car",
    project_id: "project-1",
    name: "Car",
    color: "#2563EB",
    category: "object",
  },
];

const annotations: Annotation[] = [
  {
    id: "annotation-bbox",
    image_id: "image-1",
    type: "bbox",
    coordinates: { x: 0, y: 0, width: 0.5, height: 0.5 },
    label_id: "label-car",
    created_at: "2026-07-24T00:00:00Z",
  },
  {
    id: "annotation-polygon",
    image_id: "image-1",
    type: "polygon",
    coordinates: { points: [{ x: 0, y: 0 }, { x: 0.5, y: 0 }, { x: 0, y: 0.5 }] },
    label_id: null,
    created_at: "2026-07-24T00:00:00Z",
  },
];

const edges: Edge[] = [
  {
    id: "edge-1",
    image_id: "image-1",
    source_annotation_id: "annotation-bbox",
    target_annotation_id: "annotation-polygon",
    type: "reading_order",
  },
];

const scrollIntoView = vi.fn();
Object.defineProperty(Element.prototype, "scrollIntoView", {
  configurable: true,
  value: scrollIntoView,
});

describe("AnnotationInspector", () => {
  afterEach(cleanup);

  beforeEach(() => {
    scrollIntoView.mockClear();
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations,
      edges,
      selectedId: null,
      selectedEdgeId: null,
      edgeSourceId: null,
      polygonDraftPoints: [],
      dirty: false,
      saving: false,
      saveError: null,
      revision: 0,
    });
  });

  it("lists both shape types and filters without changing annotation data", () => {
    render(
      <AnnotationInspector
        labels={labels}
        onSelectAnnotation={useAnnotationStore.getState().select}
        graphEditable
      />,
    );

    const annotationList = screen.getByRole("list", { name: "Annotations" });
    expect(within(annotationList).getByText("Car")).toBeTruthy();
    expect(within(annotationList).getByText("No label")).toBeTruthy();
    expect(within(annotationList).getByText("Object / BBox")).toBeTruthy();
    expect(within(annotationList).getByText("Unlabeled / Polygon")).toBeTruthy();
    expect(within(annotationList).getAllByText("1 connection")).toHaveLength(2);

    fireEvent.change(screen.getByLabelText("Filter by shape"), {
      target: { value: "polygon" },
    });

    expect(within(annotationList).queryByText("Car")).toBeNull();
    expect(within(annotationList).getByText("Unlabeled / Polygon")).toBeTruthy();
    expect(useAnnotationStore.getState().annotations).toEqual(annotations);
  });

  it("synchronizes list selection and scrolls Canvas selections into view", () => {
    render(
      <AnnotationInspector
        labels={labels}
        onSelectAnnotation={useAnnotationStore.getState().select}
        graphEditable
      />,
    );

    const annotationList = screen.getByRole("list", { name: "Annotations" });
    const bboxButton = within(annotationList).getByRole("button", { name: /Select Car BBox/ });
    fireEvent.click(bboxButton);
    expect(useAnnotationStore.getState().selectedId).toBe("annotation-bbox");
    expect(bboxButton.getAttribute("aria-pressed")).toBe("true");

    act(() => useAnnotationStore.getState().select(null));
    fireEvent.change(screen.getByLabelText("Filter by shape"), {
      target: { value: "polygon" },
    });
    act(() => useAnnotationStore.getState().select("annotation-bbox"));

    const revealedBBoxButton = within(annotationList).getByRole("button", { name: /Select Car BBox/ });
    expect(revealedBBoxButton.getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByText("1 of 2 + selected")).toBeTruthy();
    expect(scrollIntoView).toHaveBeenCalled();
  });

  it("deletes an annotation with its connections and marks the graph dirty", () => {
    render(
      <AnnotationInspector
        labels={labels}
        onSelectAnnotation={useAnnotationStore.getState().select}
        graphEditable
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Delete Car annotation 1" }));

    const state = useAnnotationStore.getState();
    expect(state.annotations.map((annotation) => annotation.id)).toEqual(["annotation-polygon"]);
    expect(state.edges).toEqual([]);
    expect(state.dirty).toBe(true);
    expect(state.revision).toBe(1);
  });
});
