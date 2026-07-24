export type Tool = "select" | "bbox" | "polygon" | "edge" | "pan";

export type Project = {
  id: string;
  name: string;
  created_at: string;
};

export type LabelCategory =
  | "object"
  | "entity"
  | "key"
  | "value"
  | "table"
  | "cell";

export type LabelDefinition = {
  id: string;
  project_id: string;
  name: string;
  color: string;
  category: LabelCategory;
};

export type Guideline = {
  id: string;
  project_id: string;
  title: string;
  body: string;
  display_order: number;
  updated_at: string;
};

export type CommentType = "question" | "issue" | "note";

export type QAComment = {
  id: string;
  image_id: string;
  annotation_id: string | null;
  author: string;
  body: string;
  type: CommentType;
  resolved: boolean;
  target_deleted: boolean;
  created_at: string;
  updated_at: string;
};

export type CreateCommentInput = {
  annotation_id: string | null;
  author: string;
  body: string;
  type: CommentType;
};

export type ImageStatus =
  | "pending"
  | "annotated"
  | "in_review"
  | "approved"
  | "rejected";

export type ImageWorkflowEvent =
  | "annotation_completed"
  | "annotation_reopened"
  | "review_started"
  | "review_cancelled"
  | "review_approved"
  | "review_rejected"
  | "approval_reopened"
  | "escalation_started"
  | "escalation_resolved";

export type ImageListFilters = {
  status?: ImageStatus;
  escalated?: boolean;
};

export type ImageMeta = {
  id: string;
  project_id: string;
  filename: string;
  original_width: number;
  original_height: number;
  width: number;
  height: number;
  rotation: 0 | 90 | 180 | 270;
  flip_h: boolean;
  flip_v: boolean;
  status: ImageStatus;
  escalated: boolean;
  uploaded_at: string;
};

export type BBoxCoordinates = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type NormalizedPoint = {
  x: number;
  y: number;
};

export type PolygonCoordinates = {
  points: NormalizedPoint[];
};

export type Annotation = {
  id: string;
  image_id: string;
  type: "bbox" | "polygon";
  coordinates: BBoxCoordinates | PolygonCoordinates;
  label_id: string | null;
  created_at: string;
};

export type EdgeType = "reading_order" | "key_value" | "table_cell";

export type Edge = {
  id: string;
  image_id: string;
  source_annotation_id: string;
  target_annotation_id: string;
  type: EdgeType;
};

export type ImageGraphAnnotationInput = {
  client_id: string;
  id: string;
  type: Annotation["type"];
  coordinates: Annotation["coordinates"];
  label_id: string | null;
};

export type ImageGraphEdgeInput = {
  client_id: string;
  id: string;
  source_annotation_client_id: string;
  target_annotation_client_id: string;
  type: EdgeType;
};

export type ImageGraphSaveRequest = {
  annotations: ImageGraphAnnotationInput[];
  edges: ImageGraphEdgeInput[];
};

export type ImageGraphSaveResponse = {
  annotations: { client_id: string; annotation: Annotation }[];
  edges: { client_id: string; edge: Edge }[];
};

export type ListResponse<T> = {
  items: T[];
};
