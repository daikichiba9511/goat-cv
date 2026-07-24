import { create } from "zustand";
import type {
  Annotation,
  BBoxCoordinates,
  Edge,
  EdgeType,
  LabelCategory,
  LabelDefinition,
  NormalizedPoint,
  PolygonCoordinates,
} from "../types";
import { categoryLabel, EDGE_RELATIONS } from "../edgeRelations";
import * as api from "../api/client";

type AnnotationStore = {
  loadedImageId: string | null;
  annotations: Annotation[];
  edges: Edge[];
  selectedId: string | null;
  selectedEdgeId: string | null;
  edgeSourceId: string | null;
  edgeType: EdgeType;
  edgeDraftError: string | null;
  polygonDraftPoints: NormalizedPoint[];
  dirty: boolean;
  saving: boolean;
  saveError: string | null;
  revision: number;

  loadAnnotations: (imageId: string) => Promise<void>;
  addBBox: (imageId: string, coords: BBoxCoordinates, labelId: string | null) => void;
  updateBBoxCoordinates: (id: string, coords: BBoxCoordinates) => void;
  setLabel: (id: string, labelId: string | null) => void;
  select: (id: string | null) => void;
  selectEdge: (id: string | null) => void;
  setEdgeType: (edgeType: EdgeType) => void;
  connectEdge: (imageId: string, annotationId: string, labels: LabelDefinition[]) => void;
  cancelEdgeDraft: () => void;
  addPolygonDraftPoint: (point: NormalizedPoint) => void;
  undoPolygonDraftPoint: () => void;
  cancelPolygonDraft: () => void;
  finishPolygon: (imageId: string, labelId: string | null) => boolean;
  updatePolygonPoint: (
    annotationId: string,
    pointIndex: number,
    point: NormalizedPoint,
  ) => void;
  remove: (id: string) => void;
  removeEdge: (id: string) => void;
  save: (imageId: string) => Promise<boolean>;
  clear: () => void;
};

let nextTempId = 0;
let nextTempEdgeId = 0;

function isNormalizedPoint(point: NormalizedPoint): boolean {
  return Number.isFinite(point.x) && Number.isFinite(point.y)
    && point.x >= 0 && point.x <= 1
    && point.y >= 0 && point.y <= 1;
}

function annotationCategory(
  annotation: Annotation,
  labels: LabelDefinition[],
): LabelCategory | null {
  if (!annotation.label_id) return null;
  return labels.find((label) => label.id === annotation.label_id)?.category ?? null;
}

function categoryError(
  endpoint: "Source" | "Target",
  expectedCategory: LabelCategory | undefined,
  annotation: Annotation,
  labels: LabelDefinition[],
): string | null {
  if (!expectedCategory || annotationCategory(annotation, labels) === expectedCategory) {
    return null;
  }
  return `${endpoint} must use a ${categoryLabel(expectedCategory)} label.`;
}

function createsReadingOrderCycle(
  edges: Edge[],
  sourceAnnotationId: string,
  targetAnnotationId: string,
): boolean {
  const nextByAnnotationId = new Map<string, string[]>();
  for (const edge of edges) {
    if (edge.type !== "reading_order") continue;
    const nextIds = nextByAnnotationId.get(edge.source_annotation_id) ?? [];
    nextIds.push(edge.target_annotation_id);
    nextByAnnotationId.set(edge.source_annotation_id, nextIds);
  }

  // Why: targetからsourceへ既存経路がある場合だけ、新しいsource→targetが閉路を完成させる。
  const pendingAnnotationIds = [targetAnnotationId];
  const visitedAnnotationIds = new Set<string>();
  while (pendingAnnotationIds.length > 0) {
    const annotationId = pendingAnnotationIds.pop();
    if (!annotationId || visitedAnnotationIds.has(annotationId)) continue;
    if (annotationId === sourceAnnotationId) return true;
    visitedAnnotationIds.add(annotationId);
    pendingAnnotationIds.push(...(nextByAnnotationId.get(annotationId) ?? []));
  }
  return false;
}

function edgeValidationError(
  state: Pick<AnnotationStore, "annotations" | "edges" | "edgeType">,
  sourceAnnotationId: string,
  targetAnnotationId: string,
  labels: LabelDefinition[],
): string | null {
  // Why: 保存時だけでなく接続操作の時点で、Backendと同じ拒否理由を利用者へ返す。
  if (sourceAnnotationId === targetAnnotationId) {
    return "Source and target must be different annotations.";
  }

  const sourceAnnotation = state.annotations.find(
    (annotation) => annotation.id === sourceAnnotationId,
  );
  const targetAnnotation = state.annotations.find(
    (annotation) => annotation.id === targetAnnotationId,
  );
  if (!sourceAnnotation || !targetAnnotation) {
    return "The selected annotation no longer exists.";
  }

  const relation = EDGE_RELATIONS[state.edgeType];
  const sourceCategoryError = categoryError(
    "Source",
    relation.sourceCategory,
    sourceAnnotation,
    labels,
  );
  if (sourceCategoryError) return sourceCategoryError;
  const targetCategoryError = categoryError(
    "Target",
    relation.targetCategory,
    targetAnnotation,
    labels,
  );
  if (targetCategoryError) return targetCategoryError;

  const duplicateExists = state.edges.some((edge) =>
    edge.source_annotation_id === sourceAnnotationId &&
    edge.target_annotation_id === targetAnnotationId &&
    edge.type === state.edgeType,
  );
  if (duplicateExists) return "This relation already exists.";

  if (state.edgeType === "reading_order") {
    return createsReadingOrderCycle(state.edges, sourceAnnotationId, targetAnnotationId)
      ? "Reading order cannot contain a cycle."
      : null;
  }

  if (state.edgeType === "key_value") {
    if (state.edges.some((edge) =>
      edge.type === "key_value" && edge.source_annotation_id === sourceAnnotationId,
    )) {
      return "The selected Key already has a Value.";
    }
    if (state.edges.some((edge) =>
      edge.type === "key_value" && edge.target_annotation_id === targetAnnotationId,
    )) {
      return "The selected Value is already connected to a Key.";
    }
  }

  if (state.edgeType === "table_cell" && state.edges.some((edge) =>
    edge.type === "table_cell" && edge.target_annotation_id === targetAnnotationId,
  )) {
    return "The selected Cell already belongs to a Table.";
  }
  return null;
}

export const useAnnotationStore = create<AnnotationStore>((set, get) => ({
  loadedImageId: null,
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
      edgeType: "reading_order",
      edgeDraftError: null,
      polygonDraftPoints: [],
      dirty: false,
      saving: false,
      saveError: null,
      revision: state.revision + 1,
    }));
  },

  addBBox: (imageId, coords, labelId) => {
    // Why: 保存前でも選択・編集できるように、サーバーIDとは衝突しない一時IDを付ける。
    const annotation: Annotation = {
      id: `temp-${++nextTempId}`,
      image_id: imageId,
      type: "bbox",
      coordinates: coords,
      label_id: labelId,
      created_at: new Date().toISOString(),
    };
    set((state) => ({
      annotations: [...state.annotations, annotation],
      selectedId: annotation.id,
      selectedEdgeId: null,
      edgeSourceId: null,
      edgeDraftError: null,
      polygonDraftPoints: [],
      dirty: true,
      revision: state.revision + 1,
    }));
  },

  updateBBoxCoordinates: (id, coords) => {
    set((state) => ({
      annotations: state.annotations.map((annotation) =>
        annotation.id === id ? { ...annotation, coordinates: coords } : annotation,
      ),
      dirty: true,
      revision: state.revision + 1,
    }));
  },

  setLabel: (id, labelId) => {
    set((state) => ({
      annotations: state.annotations.map((annotation) =>
        annotation.id === id ? { ...annotation, label_id: labelId } : annotation,
      ),
      dirty: true,
      revision: state.revision + 1,
    }));
  },

  select: (id) => set({ selectedId: id, selectedEdgeId: null }),

  selectEdge: (id) => set({
    selectedId: null,
    selectedEdgeId: id,
    edgeSourceId: null,
    edgeDraftError: null,
  }),

  setEdgeType: (edgeType) => {
    set((state) => {
      if (state.edgeType === edgeType) return state;
      return {
        edgeType,
        edgeSourceId: null,
        edgeDraftError: null,
      };
    });
  },

  connectEdge: (imageId, annotationId, labels) => {
    set((state) => {
      const annotation = state.annotations.find((item) => item.id === annotationId);
      if (!annotation) {
        return { edgeDraftError: "The selected annotation no longer exists." };
      }

      if (!state.edgeSourceId) {
        const error = categoryError(
          "Source",
          EDGE_RELATIONS[state.edgeType].sourceCategory,
          annotation,
          labels,
        );
        if (error) {
          return {
            selectedId: annotationId,
            selectedEdgeId: null,
            edgeDraftError: error,
          };
        }
        return {
          edgeSourceId: annotationId,
          selectedId: annotationId,
          selectedEdgeId: null,
          edgeDraftError: null,
        };
      }

      const error = edgeValidationError(
        state,
        state.edgeSourceId,
        annotationId,
        labels,
      );
      if (error) {
        return {
          selectedId: annotationId,
          selectedEdgeId: null,
          edgeDraftError: error,
        };
      }

      const edge: Edge = {
        id: `temp-edge-${++nextTempEdgeId}`,
        image_id: imageId,
        source_annotation_id: state.edgeSourceId,
        target_annotation_id: annotationId,
        type: state.edgeType,
      };
      // Why: 連続入力の単位が異なるため、Orderは終点、Tableは始点を保持し、1:1のKVは解除する。
      const nextSourceId = state.edgeType === "reading_order"
        ? annotationId
        : state.edgeType === "table_cell"
          ? state.edgeSourceId
          : null;
      return {
        edges: [...state.edges, edge],
        selectedId: annotationId,
        selectedEdgeId: null,
        edgeSourceId: nextSourceId,
        edgeDraftError: null,
        dirty: true,
        revision: state.revision + 1,
      };
    });
  },

  cancelEdgeDraft: () => set({ edgeSourceId: null, edgeDraftError: null }),

  addPolygonDraftPoint: (point) => {
    if (!isNormalizedPoint(point)) return;
    set((state) => {
      const pointExists = state.polygonDraftPoints.some(
        (existingPoint) => existingPoint.x === point.x && existingPoint.y === point.y,
      );
      if (pointExists) return state;
      return { polygonDraftPoints: [...state.polygonDraftPoints, point] };
    });
  },

  undoPolygonDraftPoint: () => set((state) => ({
    polygonDraftPoints: state.polygonDraftPoints.slice(0, -1),
  })),

  cancelPolygonDraft: () => set({ polygonDraftPoints: [] }),

  finishPolygon: (imageId, labelId) => {
    const points = get().polygonDraftPoints;
    // Why not: Backendと同じく、相異なる3点を持たない輪郭はAnnotationへ昇格させない。
    if (points.length < 3) return false;

    const annotation: Annotation = {
      id: `temp-${++nextTempId}`,
      image_id: imageId,
      type: "polygon",
      coordinates: { points },
      label_id: labelId,
      created_at: new Date().toISOString(),
    };
    set((state) => ({
      annotations: [...state.annotations, annotation],
      selectedId: annotation.id,
      selectedEdgeId: null,
      edgeSourceId: null,
      edgeDraftError: null,
      polygonDraftPoints: [],
      dirty: true,
      revision: state.revision + 1,
    }));
    return true;
  },

  updatePolygonPoint: (annotationId, pointIndex, point) => {
    if (!isNormalizedPoint(point)) return;
    set((state) => {
      const annotation = state.annotations.find((item) => item.id === annotationId);
      if (!annotation || annotation.type !== "polygon") return state;
      const coordinates = annotation.coordinates as PolygonCoordinates;
      if (pointIndex < 0 || pointIndex >= coordinates.points.length) return state;
      const overlapsAnotherPoint = coordinates.points.some(
        (existingPoint, index) => index !== pointIndex
          && existingPoint.x === point.x
          && existingPoint.y === point.y,
      );
      // Why not: 重複頂点を許すと、画面上では編集できても保存時にAPIのPolygon検証で拒否される。
      if (overlapsAnotherPoint) return state;

      const points = coordinates.points.map((existingPoint, index) =>
        index === pointIndex ? point : existingPoint,
      );
      return {
        annotations: state.annotations.map((item) =>
          item.id === annotationId
            ? { ...item, coordinates: { points } }
            : item,
        ),
        dirty: true,
        revision: state.revision + 1,
      };
    });
  },

  remove: (id) => {
    set((state) => ({
      annotations: state.annotations.filter((annotation) => annotation.id !== id),
      edges: state.edges.filter((edge) =>
        edge.source_annotation_id !== id && edge.target_annotation_id !== id,
      ),
      selectedId: state.selectedId === id ? null : state.selectedId,
      edgeSourceId: state.edgeSourceId === id ? null : state.edgeSourceId,
      edgeDraftError: state.edgeSourceId === id ? null : state.edgeDraftError,
      dirty: true,
      revision: state.revision + 1,
    }));
  },

  removeEdge: (id) => {
    set((state) => ({
      edges: state.edges.filter((edge) => edge.id !== id),
      selectedEdgeId: state.selectedEdgeId === id ? null : state.selectedEdgeId,
      edgeDraftError: null,
      dirty: true,
      revision: state.revision + 1,
    }));
  },

  save: async (imageId) => {
    if (get().saving) return false;

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
        return false;
      }
      if (get().revision !== revision) {
        // Why: 保存開始後の編集を古いレスポンスで上書きせず、次の保存対象としてローカルに残す。
        set({ saving: false, saveError: null });
        return false;
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
      return true;
    } catch (error) {
      if (get().loadedImageId !== loadedImageId) {
        return false;
      }
      set({
        dirty: true,
        saving: false,
        saveError: error instanceof Error ? error.message : String(error),
      });
      return false;
    }
  },

  clear: () => set((state) => ({
    loadedImageId: null,
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
    revision: state.revision + 1,
  })),
}));
