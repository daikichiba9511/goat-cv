import { create } from "zustand";
import type { Annotation, BBoxCoordinates, Edge } from "../types";
import * as api from "../api/client";

type AnnotationStore = {
  annotations: Annotation[];
  edges: Edge[];
  selectedId: string | null;
  selectedEdgeId: string | null;
  edgeSourceId: string | null;
  dirty: boolean;

  loadAnnotations: (imageId: string) => Promise<void>;
  addBBox: (imageId: string, coords: BBoxCoordinates, labelId: string | null) => void;
  updateCoordinates: (id: string, coords: BBoxCoordinates) => void;
  setLabel: (id: string, labelId: string | null) => void;
  select: (id: string | null) => void;
  selectEdge: (id: string | null) => void;
  setEdgeSource: (id: string | null) => void;
  addReadingOrderEdge: (imageId: string, sourceAnnotationId: string, targetAnnotationId: string) => void;
  remove: (id: string) => void;
  removeEdge: (id: string) => void;
  save: (imageId: string) => Promise<void>;
  clear: () => void;
};

let nextTempId = 0;
let nextTempEdgeId = 0;

export const useAnnotationStore = create<AnnotationStore>((set, get) => ({
  annotations: [],
  edges: [],
  selectedId: null,
  selectedEdgeId: null,
  edgeSourceId: null,
  dirty: false,

  loadAnnotations: async (imageId) => {
    const [annotationRes, edgeRes] = await Promise.all([
      api.listAnnotations(imageId),
      api.listEdges(imageId),
    ]);
    set({
      annotations: annotationRes.items,
      edges: edgeRes.items,
      selectedId: null,
      selectedEdgeId: null,
      edgeSourceId: null,
      dirty: false,
    });
  },

  addBBox: (imageId, coords, labelId) => {
    // Why: 保存前でも選択・編集できるように、サーバーIDとは衝突しない一時IDを付ける。
    const ann: Annotation = {
      id: `temp-${++nextTempId}`,
      image_id: imageId,
      type: "bbox",
      coordinates: coords,
      label_id: labelId,
      created_at: new Date().toISOString(),
    };
    set((s) => ({
      annotations: [...s.annotations, ann],
      selectedId: ann.id,
      selectedEdgeId: null,
      edgeSourceId: null,
      dirty: true,
    }));
  },

  updateCoordinates: (id, coords) => {
    set((s) => ({
      annotations: s.annotations.map((a) =>
        a.id === id ? { ...a, coordinates: coords } : a,
      ),
      dirty: true,
    }));
  },

  setLabel: (id, labelId) => {
    set((s) => ({
      annotations: s.annotations.map((a) =>
        a.id === id ? { ...a, label_id: labelId } : a,
      ),
      dirty: true,
    }));
  },

  select: (id) => set({ selectedId: id, selectedEdgeId: null }),

  selectEdge: (id) => set({ selectedId: null, selectedEdgeId: id, edgeSourceId: null }),

  setEdgeSource: (id) => set({ edgeSourceId: id, selectedId: id, selectedEdgeId: null }),

  addReadingOrderEdge: (imageId, sourceAnnotationId, targetAnnotationId) => {
    if (sourceAnnotationId === targetAnnotationId) return;
    set((s) => {
      const exists = s.edges.some((edge) =>
        edge.source_annotation_id === sourceAnnotationId &&
        edge.target_annotation_id === targetAnnotationId &&
        edge.type === "reading_order",
      );
      if (exists) {
        return { edgeSourceId: targetAnnotationId, selectedId: targetAnnotationId };
      }

      const edge: Edge = {
        id: `temp-edge-${++nextTempEdgeId}`,
        image_id: imageId,
        source_annotation_id: sourceAnnotationId,
        target_annotation_id: targetAnnotationId,
        type: "reading_order",
      };
      return {
        edges: [...s.edges, edge],
        selectedId: targetAnnotationId,
        selectedEdgeId: null,
        edgeSourceId: targetAnnotationId,
        dirty: true,
      };
    });
  },

  remove: (id) => {
    set((s) => ({
      annotations: s.annotations.filter((a) => a.id !== id),
      edges: s.edges.filter((edge) =>
        edge.source_annotation_id !== id && edge.target_annotation_id !== id,
      ),
      selectedId: s.selectedId === id ? null : s.selectedId,
      edgeSourceId: s.edgeSourceId === id ? null : s.edgeSourceId,
      dirty: true,
    }));
  },

  removeEdge: (id) => {
    set((s) => ({
      edges: s.edges.filter((edge) => edge.id !== id),
      selectedEdgeId: s.selectedEdgeId === id ? null : s.selectedEdgeId,
      dirty: true,
    }));
  },

  save: async (imageId) => {
    const { annotations, edges, selectedId } = get();
    // Why: サーバーは空IDを新規Annotationとして扱う。temp IDを送らず永続IDの責任をBackendに寄せる。
    const payload = annotations.map((a) => ({
      id: a.id.startsWith("temp-") ? "" : a.id,
      type: a.type,
      coordinates: a.coordinates,
      label_id: a.label_id,
    }));
    const annotationRes = await api.bulkReplaceAnnotations(imageId, payload);

    const savedAnnotationIdByClientId = new Map<string, string>();
    annotations.forEach((annotation, index) => {
      const savedAnnotation = annotationRes.items[index];
      if (savedAnnotation) {
        savedAnnotationIdByClientId.set(annotation.id, savedAnnotation.id);
      }
    });

    const validAnnotationIds = new Set(annotationRes.items.map((annotation) => annotation.id));
    const edgePayload = edges.flatMap((edge) => {
      const sourceAnnotationId = savedAnnotationIdByClientId.get(edge.source_annotation_id);
      const targetAnnotationId = savedAnnotationIdByClientId.get(edge.target_annotation_id);
      if (
        !sourceAnnotationId ||
        !targetAnnotationId ||
        !validAnnotationIds.has(sourceAnnotationId) ||
        !validAnnotationIds.has(targetAnnotationId)
      ) {
        return [];
      }
      return [{
        id: edge.id.startsWith("temp-edge-") ? "" : edge.id,
        source_annotation_id: sourceAnnotationId,
        target_annotation_id: targetAnnotationId,
        type: edge.type,
      }];
    });
    const edgeRes = await api.bulkReplaceEdges(imageId, edgePayload);

    set({
      annotations: annotationRes.items,
      edges: edgeRes.items,
      selectedId: selectedId ? savedAnnotationIdByClientId.get(selectedId) ?? null : null,
      selectedEdgeId: null,
      edgeSourceId: null,
      dirty: false,
    });
  },

  clear: () => set({
    annotations: [],
    edges: [],
    selectedId: null,
    selectedEdgeId: null,
    edgeSourceId: null,
    dirty: false,
  }),
}));
