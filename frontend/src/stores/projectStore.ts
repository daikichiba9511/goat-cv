import { create } from "zustand";
import type {
  Project,
  LabelDefinition,
  ImageMeta,
  LabelCategory,
  Guideline,
} from "../types";
import * as api from "../api/client";

type ProjectStore = {
  projects: Project[];
  currentProject: Project | null;
  labels: LabelDefinition[];
  guidelines: Guideline[];
  images: ImageMeta[];

  fetchProjects: () => Promise<void>;
  selectProject: (id: string) => Promise<void>;
  createProject: (name: string) => Promise<void>;
  deleteProject: (id: string) => Promise<void>;

  fetchLabels: () => Promise<void>;
  createLabel: (
    name: string,
    color: string,
    category: LabelCategory,
  ) => Promise<void>;
  updateLabel: (
    id: string,
    name: string,
    color: string,
    category: LabelCategory,
  ) => Promise<void>;
  deleteLabel: (id: string) => Promise<void>;

  fetchGuidelines: () => Promise<void>;
  createGuideline: (
    title: string,
    body: string,
    displayOrder: number,
  ) => Promise<Guideline>;
  updateGuideline: (
    id: string,
    title: string,
    body: string,
    displayOrder: number,
  ) => Promise<Guideline>;
  deleteGuideline: (id: string) => Promise<void>;

  fetchImages: () => Promise<void>;
  uploadImage: (file: File) => Promise<void>;
  deleteImage: (id: string) => Promise<void>;
};

export const useProjectStore = create<ProjectStore>((set, get) => ({
  projects: [],
  currentProject: null,
  labels: [],
  guidelines: [],
  images: [],

  fetchProjects: async () => {
    const res = await api.listProjects();
    set({ projects: res.items });
  },

  selectProject: async (id) => {
    const project = await api.getProject(id);
    set({ currentProject: project });
    await Promise.all([
      get().fetchLabels(),
      get().fetchGuidelines(),
      get().fetchImages(),
    ]);
  },

  createProject: async (name) => {
    await api.createProject(name);
    await get().fetchProjects();
  },

  deleteProject: async (id) => {
    await api.deleteProject(id);
    const { currentProject } = get();
    if (currentProject?.id === id) {
      set({ currentProject: null, labels: [], guidelines: [], images: [] });
    }
    await get().fetchProjects();
  },

  fetchLabels: async () => {
    const project = get().currentProject;
    if (!project) return;
    const res = await api.listLabels(project.id);
    set({ labels: res.items });
  },

  createLabel: async (name, color, category) => {
    const project = get().currentProject;
    if (!project) return;
    await api.createLabel(project.id, name, color, category);
    await get().fetchLabels();
  },

  updateLabel: async (id, name, color, category) => {
    const project = get().currentProject;
    if (!project) return;
    await api.updateLabel(id, project.id, name, color, category);
    await get().fetchLabels();
  },

  deleteLabel: async (id) => {
    const project = get().currentProject;
    if (!project) return;
    await api.deleteLabel(id, project.id);
    await get().fetchLabels();
  },

  fetchGuidelines: async () => {
    const project = get().currentProject;
    if (!project) return;
    const response = await api.listGuidelines(project.id);
    set({ guidelines: response.items });
  },

  createGuideline: async (title, body, displayOrder) => {
    const project = get().currentProject;
    if (!project) throw new Error("project is not selected");
    const created = await api.createGuideline(project.id, title, body, displayOrder);
    await get().fetchGuidelines();
    return created;
  },

  updateGuideline: async (id, title, body, displayOrder) => {
    const project = get().currentProject;
    if (!project) throw new Error("project is not selected");
    const updated = await api.updateGuideline(project.id, id, title, body, displayOrder);
    await get().fetchGuidelines();
    return updated;
  },

  deleteGuideline: async (id) => {
    const project = get().currentProject;
    if (!project) throw new Error("project is not selected");
    await api.deleteGuideline(project.id, id);
    await get().fetchGuidelines();
  },

  fetchImages: async () => {
    const project = get().currentProject;
    if (!project) return;
    const res = await api.listImages(project.id);
    set({ images: res.items });
  },

  uploadImage: async (file) => {
    const project = get().currentProject;
    if (!project) return;
    await api.uploadImage(project.id, file);
    await get().fetchImages();
  },

  deleteImage: async (id) => {
    await api.deleteImage(id);
    await get().fetchImages();
  },
}));
