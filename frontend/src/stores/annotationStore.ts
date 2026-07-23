import { create } from "zustand";
import type { Annotation, BBoxCoordinates } from "../types";
import * as api from "../api/client";

type AnnotationStore = {
  annotations: Annotation[];
  selectedId: string | null;
  dirty: boolean;

  loadAnnotations: (imageId: string) => Promise<void>;
  addBBox: (imageId: string, coords: BBoxCoordinates, labelId: string | null) => void;
  updateCoordinates: (id: string, coords: BBoxCoordinates) => void;
  setLabel: (id: string, labelId: string | null) => void;
  select: (id: string | null) => void;
  remove: (id: string) => void;
  save: (imageId: string) => Promise<void>;
  clear: () => void;
};

let nextTempId = 0;

export const useAnnotationStore = create<AnnotationStore>((set, get) => ({
  annotations: [],
  selectedId: null,
  dirty: false,

  loadAnnotations: async (imageId) => {
    const res = await api.listAnnotations(imageId);
    set({ annotations: res.items, selectedId: null, dirty: false });
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

  select: (id) => set({ selectedId: id }),

  remove: (id) => {
    set((s) => ({
      annotations: s.annotations.filter((a) => a.id !== id),
      selectedId: s.selectedId === id ? null : s.selectedId,
      dirty: true,
    }));
  },

  save: async (imageId) => {
    const { annotations } = get();
    // Why: サーバーは空IDを新規Annotationとして扱う。temp IDを送らず永続IDの責任をBackendに寄せる。
    const payload = annotations.map((a) => ({
      id: a.id.startsWith("temp-") ? "" : a.id,
      type: a.type,
      coordinates: a.coordinates,
      label_id: a.label_id,
    }));
    const res = await api.bulkReplaceAnnotations(imageId, payload);
    set({ annotations: res.items, dirty: false });
  },

  clear: () => set({ annotations: [], selectedId: null, dirty: false }),
}));
