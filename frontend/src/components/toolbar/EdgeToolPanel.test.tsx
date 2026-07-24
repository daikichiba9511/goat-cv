// @vitest-environment jsdom

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useAnnotationStore } from "../../stores/annotationStore";
import { useProjectStore } from "../../stores/projectStore";
import type { Annotation, LabelDefinition } from "../../types";
import EdgeToolPanel from "./EdgeToolPanel";

const keyLabel: LabelDefinition = {
  id: "label-key",
  project_id: "project-1",
  name: "Invoice field",
  color: "#0F766E",
  category: "key",
};

const keyAnnotation: Annotation = {
  id: "annotation-key",
  image_id: "image-1",
  type: "bbox",
  coordinates: { x: 0, y: 0, width: 0.5, height: 0.5 },
  label_id: keyLabel.id,
  created_at: "2026-07-24T00:00:00Z",
};

describe("EdgeToolPanel", () => {
  afterEach(cleanup);

  beforeEach(() => {
    useProjectStore.setState({ labels: [keyLabel] });
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations: [keyAnnotation],
      edges: [],
      selectedId: keyAnnotation.id,
      selectedEdgeId: null,
      edgeSourceId: keyAnnotation.id,
      edgeType: "key_value",
      edgeDraftError: "Target must use a Value label.",
      polygonDraftPoints: [],
      dirty: false,
      saving: false,
      saveError: null,
      revision: 0,
    });
  });

  it("shows the active relation constraints, source category, and rejection reason", () => {
    render(<EdgeToolPanel />);

    expect(screen.getByText("Invoice field / Key")).toBeTruthy();
    expect(screen.getByText("Connect a Key annotation to one Value annotation.")).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toBe("Target must use a Value label.");
  });
});
