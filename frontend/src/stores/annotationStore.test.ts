import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Annotation, Edge } from "../types";
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
    useAnnotationStore.getState().updateCoordinates("temp-a", editedCoordinates);
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
