import type {
  Project,
  LabelDefinition,
  ImageMeta,
  Annotation,
  Edge,
  ListResponse,
  LabelCategory,
  Guideline,
  ImageGraphSaveRequest,
  ImageGraphSaveResponse,
} from "../types";

const BASE = "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

function json(body: unknown): RequestInit {
  return {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  };
}

// Projects

export async function createProject(name: string): Promise<Project> {
  return request("/projects", { method: "POST", ...json({ name }) });
}

export async function listProjects(): Promise<ListResponse<Project>> {
  return request("/projects");
}

export async function getProject(id: string): Promise<Project> {
  return request(`/projects/${id}`);
}

export async function updateProject(
  id: string,
  name: string,
): Promise<Project> {
  return request(`/projects/${id}`, { method: "PATCH", ...json({ name }) });
}

export async function deleteProject(id: string): Promise<void> {
  return request(`/projects/${id}`, { method: "DELETE" });
}

// Labels

export async function createLabel(
  projectId: string,
  name: string,
  color: string,
  category: LabelCategory,
): Promise<LabelDefinition> {
  return request(`/projects/${projectId}/labels`, {
    method: "POST",
    ...json({ name, color, category }),
  });
}

export async function listLabels(
  projectId: string,
): Promise<ListResponse<LabelDefinition>> {
  return request(`/projects/${projectId}/labels`);
}

export async function updateLabel(
  labelId: string,
  projectId: string,
  name: string,
  color: string,
  category: LabelCategory,
): Promise<LabelDefinition> {
  return request(`/projects/${projectId}/labels/${labelId}`, {
    method: "PATCH",
    ...json({ name, color, category }),
  });
}

export async function deleteLabel(
  labelId: string,
  projectId: string,
): Promise<void> {
  return request(`/projects/${projectId}/labels/${labelId}`, {
    method: "DELETE",
  });
}

// Guidelines

export async function createGuideline(
  projectId: string,
  title: string,
  body: string,
  displayOrder: number,
): Promise<Guideline> {
  return request(`/projects/${projectId}/guidelines`, {
    method: "POST",
    ...json({ title, body, display_order: displayOrder }),
  });
}

export async function listGuidelines(
  projectId: string,
): Promise<ListResponse<Guideline>> {
  return request(`/projects/${projectId}/guidelines`);
}

export async function updateGuideline(
  projectId: string,
  guidelineId: string,
  title: string,
  body: string,
  displayOrder: number,
): Promise<Guideline> {
  return request(`/projects/${projectId}/guidelines/${guidelineId}`, {
    method: "PATCH",
    ...json({ title, body, display_order: displayOrder }),
  });
}

export async function deleteGuideline(
  projectId: string,
  guidelineId: string,
): Promise<void> {
  return request(`/projects/${projectId}/guidelines/${guidelineId}`, {
    method: "DELETE",
  });
}

// Images

export async function uploadImage(
  projectId: string,
  file: File,
): Promise<ImageMeta> {
  const form = new FormData();
  form.append("file", file);
  return request(`/projects/${projectId}/images`, {
    method: "POST",
    body: form,
  });
}

export async function listImages(
  projectId: string,
): Promise<ListResponse<ImageMeta>> {
  return request(`/projects/${projectId}/images`);
}

export async function getImage(imageId: string): Promise<ImageMeta> {
  return request(`/images/${imageId}`);
}

export function imageFileUrl(imageId: string): string {
  return `${BASE}/images/${imageId}/file`;
}

export async function updateImageTransform(
  imageId: string,
  rotation: number,
  flipH: boolean,
  flipV: boolean,
): Promise<ImageMeta> {
  return request(`/images/${imageId}`, {
    method: "PATCH",
    ...json({ rotation, flip_h: flipH, flip_v: flipV }),
  });
}

export async function deleteImage(imageId: string): Promise<void> {
  return request(`/images/${imageId}`, { method: "DELETE" });
}

// Annotations

export async function createAnnotation(
  imageId: string,
  type: "bbox" | "polygon",
  coordinates: unknown,
  labelId: string | null,
): Promise<Annotation> {
  return request(`/images/${imageId}/annotations`, {
    method: "POST",
    ...json({ type, coordinates, label_id: labelId }),
  });
}

export async function listAnnotations(
  imageId: string,
): Promise<ListResponse<Annotation>> {
  return request(`/images/${imageId}/annotations`);
}

export async function updateAnnotation(
  annotationId: string,
  type: "bbox" | "polygon",
  coordinates: unknown,
  labelId: string | null,
): Promise<Annotation> {
  return request(`/annotations/${annotationId}`, {
    method: "PATCH",
    ...json({ type, coordinates, label_id: labelId }),
  });
}

export async function deleteAnnotation(annotationId: string): Promise<void> {
  return request(`/annotations/${annotationId}`, { method: "DELETE" });
}

// Edges

export async function listEdges(imageId: string): Promise<ListResponse<Edge>> {
  return request(`/images/${imageId}/edges`);
}

// saveImageGraph atomically replaces all annotations and edges for one image.
export async function saveImageGraph(
  imageId: string,
  graph: ImageGraphSaveRequest,
): Promise<ImageGraphSaveResponse> {
  return request(`/images/${imageId}/graph`, {
    method: "PUT",
    ...json(graph),
  });
}
