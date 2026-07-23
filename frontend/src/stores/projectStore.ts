import { create } from "zustand";
import type { Project, LabelDefinition, ImageMeta, LabelCategory } from "../types";
import * as api from "../api/client";

type ProjectStore = {
  projects: Project[];
  currentProject: Project | null;
  labels: LabelDefinition[];
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

  fetchImages: () => Promise<void>;
  uploadImage: (file: File) => Promise<void>;
  deleteImage: (id: string) => Promise<void>;
};

export const useProjectStore = create<ProjectStore>((set, get) => ({
  projects: [],
  currentProject: null,
  labels: [],
  images: [],

  fetchProjects: async () => {
    const res = await api.listProjects();
    set({ projects: res.items });
  },

  selectProject: async (id) => {
    const project = await api.getProject(id);
    set({ currentProject: project });
    await Promise.all([get().fetchLabels(), get().fetchImages()]);
  },

  createProject: async (name) => {
    await api.createProject(name);
    await get().fetchProjects();
  },

  deleteProject: async (id) => {
    await api.deleteProject(id);
    const { currentProject } = get();
    if (currentProject?.id === id) {
      set({ currentProject: null, labels: [], images: [] });
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
