import { create } from "zustand";
import type { Annotation, BBoxCoordinates, Edge } from "../types";
import * as api from "../api/client";

type AnnotationStore = {
  loadedImageId: string | null;
  annotations: Annotation[];
  edges: Edge[];
  selectedId: string | null;
  selectedEdgeId: string | null;
  edgeSourceId: string | null;
  dirty: boolean;
  saving: boolean;
  saveError: string | null;
  revision: number;

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
  loadedImageId: null,
  annotations: [],
  edges: [],
  selectedId: null,
  selectedEdgeId: null,
  edgeSourceId: null,
  dirty: false,
  saving: false,
  saveError: null,
  revision: 0,

  loadAnnotations: async (imageId) => {
    const [annotationRes, edgeRes] = await Promise.all([
      api.listAnnotations(imageId),
      api.listEdges(imageId),
    ]);
    set((state) => ({
      loadedImageId: imageId,
      annotations: annotationRes.items,
      edges: edgeRes.items,
      selectedId: null,
      selectedEdgeId: null,
      edgeSourceId: null,
      dirty: false,
      saving: false,
      saveError: null,
      revision: state.revision + 1,
    }));
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
      revision: s.revision + 1,
    }));
  },

  updateCoordinates: (id, coords) => {
    set((s) => ({
      annotations: s.annotations.map((a) =>
        a.id === id ? { ...a, coordinates: coords } : a,
      ),
      dirty: true,
      revision: s.revision + 1,
    }));
  },

  setLabel: (id, labelId) => {
    set((s) => ({
      annotations: s.annotations.map((a) =>
        a.id === id ? { ...a, label_id: labelId } : a,
      ),
      dirty: true,
      revision: s.revision + 1,
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
        revision: s.revision + 1,
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
      revision: s.revision + 1,
    }));
  },

  removeEdge: (id) => {
    set((s) => ({
      edges: s.edges.filter((edge) => edge.id !== id),
      selectedEdgeId: s.selectedEdgeId === id ? null : s.selectedEdgeId,
      dirty: true,
      revision: s.revision + 1,
    }));
  },

  save: async (imageId) => {
    if (get().saving) return;

    const {
      annotations,
      edges,
      selectedId,
      selectedEdgeId,
      edgeSourceId,
      revision,
      loadedImageId,
    } = get();
    const graph = {
      annotations: annotations.map((annotation) => ({
        client_id: annotation.id,
        id: annotation.id.startsWith("temp-") ? "" : annotation.id,
        type: annotation.type,
        coordinates: annotation.coordinates,
        label_id: annotation.label_id,
      })),
      edges: edges.map((edge) => ({
        client_id: edge.id,
        id: edge.id.startsWith("temp-edge-") ? "" : edge.id,
        source_annotation_client_id: edge.source_annotation_id,
        target_annotation_client_id: edge.target_annotation_id,
        type: edge.type,
      })),
    };

    set({ saving: true, saveError: null });
    try {
      const savedGraph = await api.saveImageGraph(imageId, graph);
      // Why: 画像切替後に届いた旧画像の保存結果を、現在表示中のグラフへ適用しない。
      if (get().loadedImageId !== loadedImageId) {
        return;
      }
      if (get().revision !== revision) {
        // Why: 保存開始後の編集を古いレスポンスで上書きせず、次の保存対象としてローカルに残す。
        set({ saving: false, saveError: null });
        return;
      }
      const savedAnnotationByClientID = new Map(
        savedGraph.annotations.map((item) => [item.client_id, item.annotation]),
      );
      const savedEdgeByClientID = new Map(
        savedGraph.edges.map((item) => [item.client_id, item.edge]),
      );
      // Why: APIの応答順は契約に含めず、client_idでIDを解決しながらローカルの描画順を保つ。
      const savedAnnotations = annotations.map((annotation) => {
        const savedAnnotation = savedAnnotationByClientID.get(annotation.id);
        if (!savedAnnotation) {
          throw new Error(`save response is missing annotation ${annotation.id}`);
        }
        return savedAnnotation;
      });
      const savedEdges = edges.map((edge) => {
        const savedEdge = savedEdgeByClientID.get(edge.id);
        if (!savedEdge) {
          throw new Error(`save response is missing edge ${edge.id}`);
        }
        return savedEdge;
      });

      set({
        annotations: savedAnnotations,
        edges: savedEdges,
        selectedId: selectedId ? savedAnnotationByClientID.get(selectedId)?.id ?? null : null,
        selectedEdgeId: selectedEdgeId ? savedEdgeByClientID.get(selectedEdgeId)?.id ?? null : null,
        edgeSourceId: edgeSourceId ? savedAnnotationByClientID.get(edgeSourceId)?.id ?? null : null,
        dirty: false,
        saving: false,
        saveError: null,
      });
    } catch (error) {
      if (get().loadedImageId !== loadedImageId) {
        return;
      }
      set({
        dirty: true,
        saving: false,
        saveError: error instanceof Error ? error.message : String(error),
      });
    }
  },

  clear: () => set((state) => ({
    loadedImageId: null,
    annotations: [],
    edges: [],
    selectedId: null,
    selectedEdgeId: null,
    edgeSourceId: null,
    dirty: false,
    saving: false,
    saveError: null,
    revision: state.revision + 1,
  })),
}));
