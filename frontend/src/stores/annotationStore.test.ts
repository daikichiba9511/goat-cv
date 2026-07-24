import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Annotation, Edge, LabelDefinition } from "../types";
import * as api from "../api/client";
import { useAnnotationStore } from "./annotationStore";

vi.mock("../api/client", () => ({
  listAnnotations: vi.fn(),
  listEdges: vi.fn(),
  saveImageGraph: vi.fn(),
}));

const initialAnnotations: Annotation[] = [
  {
    id: "temp-a",
    image_id: "image-1",
    type: "bbox",
    coordinates: { x: 0, y: 0, width: 0.5, height: 1 },
    label_id: null,
    created_at: "2026-07-24T00:00:00Z",
  },
  {
    id: "temp-b",
    image_id: "image-1",
    type: "bbox",
    coordinates: { x: 0.5, y: 0, width: 0.5, height: 1 },
    label_id: null,
    created_at: "2026-07-24T00:00:00Z",
  },
];

const initialEdges: Edge[] = [
  {
    id: "temp-edge-a",
    image_id: "image-1",
    source_annotation_id: "temp-a",
    target_annotation_id: "temp-b",
    type: "reading_order",
  },
];

const savedGraph = {
  annotations: [
    {
      client_id: "temp-b",
      annotation: { ...initialAnnotations[1], id: "server-b" },
    },
    {
      client_id: "temp-a",
      annotation: { ...initialAnnotations[0], id: "server-a" },
    },
  ],
  edges: [
    {
      client_id: "temp-edge-a",
      edge: {
        ...initialEdges[0],
        id: "server-edge-a",
        source_annotation_id: "server-a",
        target_annotation_id: "server-b",
      },
    },
  ],
};

describe("annotationStore save", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations: initialAnnotations,
      edges: initialEdges,
      selectedId: "temp-a",
      selectedEdgeId: null,
      edgeSourceId: "temp-a",
      polygonDraftPoints: [],
      dirty: true,
      saving: false,
      saveError: null,
      revision: 0,
    });
  });

  it("saves the graph once and resolves IDs through client_id", async () => {
    vi.mocked(api.saveImageGraph).mockResolvedValue(savedGraph);

    await useAnnotationStore.getState().save("image-1");

    expect(api.saveImageGraph).toHaveBeenCalledTimes(1);
    expect(api.saveImageGraph).toHaveBeenCalledWith("image-1", {
      annotations: [
        {
          client_id: "temp-a",
          id: "",
          type: "bbox",
          coordinates: initialAnnotations[0].coordinates,
          label_id: null,
        },
        {
          client_id: "temp-b",
          id: "",
          type: "bbox",
          coordinates: initialAnnotations[1].coordinates,
          label_id: null,
        },
      ],
      edges: [
        {
          client_id: "temp-edge-a",
          id: "",
          source_annotation_client_id: "temp-a",
          target_annotation_client_id: "temp-b",
          type: "reading_order",
        },
      ],
    });

    const state = useAnnotationStore.getState();
    expect(state.annotations.map((annotation) => annotation.id)).toEqual(["server-a", "server-b"]);
    expect(state.edges).toEqual([savedGraph.edges[0].edge]);
    expect(state.selectedId).toBe("server-a");
    expect(state.edgeSourceId).toBe("server-a");
    expect(state.dirty).toBe(false);
    expect(state.saving).toBe(false);
    expect(state.saveError).toBeNull();
  });

  it("keeps edits dirty after failure and allows a successful retry", async () => {
    vi.mocked(api.saveImageGraph)
      .mockRejectedValueOnce(new Error("edge cardinality rule violation"))
      .mockResolvedValueOnce(savedGraph);

    await useAnnotationStore.getState().save("image-1");

    const failedState = useAnnotationStore.getState();
    expect(failedState.annotations).toEqual(initialAnnotations);
    expect(failedState.edges).toEqual(initialEdges);
    expect(failedState.dirty).toBe(true);
    expect(failedState.saving).toBe(false);
    expect(failedState.saveError).toBe("edge cardinality rule violation");

    await failedState.save("image-1");

    const retriedState = useAnnotationStore.getState();
    expect(api.saveImageGraph).toHaveBeenCalledTimes(2);
    expect(retriedState.dirty).toBe(false);
    expect(retriedState.saveError).toBeNull();
  });

  it("does not overwrite edits made while a save is in flight", async () => {
    let resolveSave: (value: typeof savedGraph) => void = () => undefined;
    vi.mocked(api.saveImageGraph).mockReturnValue(new Promise((resolve) => {
      resolveSave = resolve;
    }));

    const savePromise = useAnnotationStore.getState().save("image-1");
    const editedCoordinates = { x: 0.1, y: 0, width: 0.4, height: 1 };
    useAnnotationStore.getState().updateBBoxCoordinates("temp-a", editedCoordinates);
    resolveSave(savedGraph);
    await savePromise;

    const state = useAnnotationStore.getState();
    expect(state.annotations[0].id).toBe("temp-a");
    expect(state.annotations[0].coordinates).toEqual(editedCoordinates);
    expect(state.dirty).toBe(true);
    expect(state.saving).toBe(false);
  });

  it("does not apply a save response after another image is loaded", async () => {
    let resolveSave: (value: typeof savedGraph) => void = () => undefined;
    vi.mocked(api.saveImageGraph).mockReturnValue(new Promise((resolve) => {
      resolveSave = resolve;
    }));
    const otherImageAnnotation = {
      ...initialAnnotations[0],
      id: "server-other",
      image_id: "image-2",
    };
    vi.mocked(api.listAnnotations).mockResolvedValue({ items: [otherImageAnnotation] });
    vi.mocked(api.listEdges).mockResolvedValue({ items: [] });

    const savePromise = useAnnotationStore.getState().save("image-1");
    await useAnnotationStore.getState().loadAnnotations("image-2");
    resolveSave(savedGraph);
    await savePromise;

    const state = useAnnotationStore.getState();
    expect(state.loadedImageId).toBe("image-2");
    expect(state.annotations).toEqual([otherImageAnnotation]);
    expect(state.dirty).toBe(false);
    expect(state.saving).toBe(false);
  });
});

const edgeLabels: LabelDefinition[] = [
  { id: "label-key", project_id: "project-1", name: "Key", color: "#0F766E", category: "key" },
  { id: "label-value", project_id: "project-1", name: "Value", color: "#2563EB", category: "value" },
  { id: "label-table", project_id: "project-1", name: "Table", color: "#C2410C", category: "table" },
  { id: "label-cell", project_id: "project-1", name: "Cell", color: "#9333EA", category: "cell" },
];

const edgeAnnotations: Annotation[] = [
  { ...initialAnnotations[0], id: "annotation-a", label_id: null },
  { ...initialAnnotations[1], id: "annotation-b", label_id: null },
  { ...initialAnnotations[0], id: "annotation-key", label_id: "label-key" },
  { ...initialAnnotations[1], id: "annotation-key-2", label_id: "label-key" },
  { ...initialAnnotations[0], id: "annotation-value", label_id: "label-value" },
  { ...initialAnnotations[1], id: "annotation-value-2", label_id: "label-value" },
  { ...initialAnnotations[0], id: "annotation-table", label_id: "label-table" },
  { ...initialAnnotations[1], id: "annotation-table-2", label_id: "label-table" },
  { ...initialAnnotations[0], id: "annotation-cell", label_id: "label-cell" },
  { ...initialAnnotations[1], id: "annotation-cell-2", label_id: "label-cell" },
];

describe("annotationStore edge editing", () => {
  beforeEach(() => {
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations: edgeAnnotations,
      edges: [],
      selectedId: null,
      selectedEdgeId: null,
      edgeSourceId: null,
      edgeType: "reading_order",
      edgeDraftError: null,
      polygonDraftPoints: [],
      dirty: false,
      saving: false,
      saveError: null,
      revision: 0,
    });
  });

  it("creates each relation in the selected direction and continues from the useful endpoint", () => {
    const store = useAnnotationStore.getState();

    store.connectEdge("image-1", "annotation-a", edgeLabels);
    store.connectEdge("image-1", "annotation-b", edgeLabels);
    expect(useAnnotationStore.getState().edgeSourceId).toBe("annotation-b");

    store.setEdgeType("key_value");
    store.connectEdge("image-1", "annotation-key", edgeLabels);
    store.connectEdge("image-1", "annotation-value", edgeLabels);
    expect(useAnnotationStore.getState().edgeSourceId).toBeNull();

    store.setEdgeType("table_cell");
    store.connectEdge("image-1", "annotation-table", edgeLabels);
    store.connectEdge("image-1", "annotation-cell", edgeLabels);

    expect(useAnnotationStore.getState().edges.map((edge) => ({
      source: edge.source_annotation_id,
      target: edge.target_annotation_id,
      type: edge.type,
    }))).toEqual([
      { source: "annotation-a", target: "annotation-b", type: "reading_order" },
      { source: "annotation-key", target: "annotation-value", type: "key_value" },
      { source: "annotation-table", target: "annotation-cell", type: "table_cell" },
    ]);
    expect(useAnnotationStore.getState().edgeSourceId).toBe("annotation-table");
  });

  it("rejects annotations with the wrong source or target category and exposes the reason", () => {
    const store = useAnnotationStore.getState();
    store.setEdgeType("key_value");

    store.connectEdge("image-1", "annotation-value", edgeLabels);
    expect(useAnnotationStore.getState().edges).toEqual([]);
    expect(useAnnotationStore.getState().edgeSourceId).toBeNull();
    expect(useAnnotationStore.getState().edgeDraftError).toMatch(/Key label/);

    store.connectEdge("image-1", "annotation-key", edgeLabels);
    store.connectEdge("image-1", "annotation-cell", edgeLabels);
    expect(useAnnotationStore.getState().edges).toEqual([]);
    expect(useAnnotationStore.getState().edgeSourceId).toBe("annotation-key");
    expect(useAnnotationStore.getState().edgeDraftError).toMatch(/Value label/);
  });

  it("rejects key-value and table-cell cardinality violations", () => {
    const store = useAnnotationStore.getState();
    store.setEdgeType("key_value");
    store.connectEdge("image-1", "annotation-key", edgeLabels);
    store.connectEdge("image-1", "annotation-value", edgeLabels);
    store.connectEdge("image-1", "annotation-key", edgeLabels);
    store.connectEdge("image-1", "annotation-value-2", edgeLabels);
    expect(useAnnotationStore.getState().edges).toHaveLength(1);
    expect(useAnnotationStore.getState().edgeDraftError).toMatch(/already has a Value/);

    store.setEdgeType("table_cell");
    store.connectEdge("image-1", "annotation-table", edgeLabels);
    store.connectEdge("image-1", "annotation-cell", edgeLabels);
    store.cancelEdgeDraft();
    store.connectEdge("image-1", "annotation-table-2", edgeLabels);
    store.connectEdge("image-1", "annotation-cell", edgeLabels);
    expect(useAnnotationStore.getState().edges).toHaveLength(2);
    expect(useAnnotationStore.getState().edgeDraftError).toMatch(/already belongs to a Table/);
  });

  it("rejects a reading-order cycle and still allows the relation to be deleted", () => {
    const store = useAnnotationStore.getState();
    store.connectEdge("image-1", "annotation-a", edgeLabels);
    store.connectEdge("image-1", "annotation-b", edgeLabels);
    store.connectEdge("image-1", "annotation-a", edgeLabels);

    expect(useAnnotationStore.getState().edges).toHaveLength(1);
    expect(useAnnotationStore.getState().edgeDraftError).toMatch(/cycle/);

    const edgeId = useAnnotationStore.getState().edges[0].id;
    store.removeEdge(edgeId);
    expect(useAnnotationStore.getState().edges).toEqual([]);
  });

  it("discards the unfinished relation when the edge type changes", () => {
    const store = useAnnotationStore.getState();
    store.connectEdge("image-1", "annotation-a", edgeLabels);
    store.setEdgeType("key_value");

    const state = useAnnotationStore.getState();
    expect(state.edges).toEqual([]);
    expect(state.edgeSourceId).toBeNull();
    expect(state.edgeDraftError).toBeNull();
    expect(state.edgeType).toBe("key_value");
    expect(state.dirty).toBe(false);
  });

  it("reloads every relation type without changing its direction", async () => {
    const reloadedEdges: Edge[] = [
      {
        id: "edge-order",
        image_id: "image-1",
        source_annotation_id: "annotation-a",
        target_annotation_id: "annotation-b",
        type: "reading_order",
      },
      {
        id: "edge-key-value",
        image_id: "image-1",
        source_annotation_id: "annotation-key",
        target_annotation_id: "annotation-value",
        type: "key_value",
      },
      {
        id: "edge-table-cell",
        image_id: "image-1",
        source_annotation_id: "annotation-table",
        target_annotation_id: "annotation-cell",
        type: "table_cell",
      },
    ];
    vi.mocked(api.listAnnotations).mockResolvedValue({ items: edgeAnnotations });
    vi.mocked(api.listEdges).mockResolvedValue({ items: reloadedEdges });

    await useAnnotationStore.getState().loadAnnotations("image-1");

    expect(useAnnotationStore.getState().edges).toEqual(reloadedEdges);
  });
});

describe("annotationStore polygon editing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAnnotationStore.setState({
      loadedImageId: "image-1",
      annotations: [],
      edges: [],
      selectedId: null,
      selectedEdgeId: null,
      edgeSourceId: null,
      edgeType: "reading_order",
      edgeDraftError: null,
      polygonDraftPoints: [],
      dirty: false,
      saving: false,
      saveError: null,
      revision: 0,
    });
  });

  it("does not create a polygon from fewer than three points and cancels without dirtying the graph", () => {
    const store = useAnnotationStore.getState();
    store.addPolygonDraftPoint({ x: 0.1, y: 0.1 });
    store.addPolygonDraftPoint({ x: 0.8, y: 0.1 });

    expect(store.finishPolygon("image-1", "label-object")).toBe(false);
    expect(useAnnotationStore.getState().annotations).toEqual([]);
    expect(useAnnotationStore.getState().polygonDraftPoints).toHaveLength(2);

    store.cancelPolygonDraft();
    const state = useAnnotationStore.getState();
    expect(state.polygonDraftPoints).toEqual([]);
    expect(state.dirty).toBe(false);
    expect(state.revision).toBe(0);
  });

  it("creates a labeled polygon and preserves vertex order through save and reload", async () => {
    const points = [
      { x: 0.1, y: 0.2 },
      { x: 0.8, y: 0.2 },
      { x: 0.5, y: 0.9 },
    ];
    const store = useAnnotationStore.getState();
    points.forEach(store.addPolygonDraftPoint);

    expect(store.finishPolygon("image-1", "label-object")).toBe(true);
    const createdPolygon = useAnnotationStore.getState().annotations[0];
    expect(createdPolygon.type).toBe("polygon");
    expect(createdPolygon.coordinates).toEqual({ points });
    expect(createdPolygon.label_id).toBe("label-object");
    expect(useAnnotationStore.getState().polygonDraftPoints).toEqual([]);

    const savedPolygon = { ...createdPolygon, id: "server-polygon" };
    vi.mocked(api.saveImageGraph).mockResolvedValue({
      annotations: [{ client_id: createdPolygon.id, annotation: savedPolygon }],
      edges: [],
    });
    await useAnnotationStore.getState().save("image-1");
    expect(api.saveImageGraph).toHaveBeenCalledWith("image-1", {
      annotations: [{
        client_id: createdPolygon.id,
        id: "",
        type: "polygon",
        coordinates: { points },
        label_id: "label-object",
      }],
      edges: [],
    });

    vi.mocked(api.listAnnotations).mockResolvedValue({ items: [savedPolygon] });
    vi.mocked(api.listEdges).mockResolvedValue({ items: [] });
    await useAnnotationStore.getState().loadAnnotations("image-1");
    expect(useAnnotationStore.getState().annotations[0]).toEqual(savedPolygon);
  });

  it("moves one vertex without reordering the polygon and removes connected edges with it", () => {
    const polygon: Annotation = {
      id: "polygon-1",
      image_id: "image-1",
      type: "polygon",
      coordinates: {
        points: [
          { x: 0.1, y: 0.1 },
          { x: 0.8, y: 0.1 },
          { x: 0.5, y: 0.8 },
        ],
      },
      label_id: "label-object",
      created_at: "2026-07-24T00:00:00Z",
    };
    const connectedEdge: Edge = {
      id: "edge-polygon",
      image_id: "image-1",
      source_annotation_id: polygon.id,
      target_annotation_id: initialAnnotations[0].id,
      type: "reading_order",
    };
    useAnnotationStore.setState({
      annotations: [polygon, initialAnnotations[0]],
      edges: [connectedEdge],
    });

    const store = useAnnotationStore.getState();
    store.updatePolygonPoint(polygon.id, 1, { x: 0.7, y: 0.3 });
    expect(useAnnotationStore.getState().annotations[0].coordinates).toEqual({
      points: [
        { x: 0.1, y: 0.1 },
        { x: 0.7, y: 0.3 },
        { x: 0.5, y: 0.8 },
      ],
    });

    store.updatePolygonPoint(polygon.id, 1, { x: 0.1, y: 0.1 });
    expect(useAnnotationStore.getState().annotations[0].coordinates).toEqual({
      points: [
        { x: 0.1, y: 0.1 },
        { x: 0.7, y: 0.3 },
        { x: 0.5, y: 0.8 },
      ],
    });

    store.remove(polygon.id);
    expect(useAnnotationStore.getState().annotations).toEqual([initialAnnotations[0]]);
    expect(useAnnotationStore.getState().edges).toEqual([]);
  });
});
