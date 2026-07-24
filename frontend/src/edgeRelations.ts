import type { EdgeType, LabelCategory } from "./types";

export type EdgeRelationDefinition = {
  label: string;
  canvasLabel: string;
  instruction: string;
  color: string;
  dash?: number[];
  sourceCategory?: LabelCategory;
  targetCategory?: LabelCategory;
};

export const EDGE_TYPE_ORDER: EdgeType[] = [
  "reading_order",
  "key_value",
  "table_cell",
];

export const EDGE_RELATIONS: Record<EdgeType, EdgeRelationDefinition> = {
  reading_order: {
    label: "Reading order",
    canvasLabel: "Order",
    instruction: "Connect any annotation to the annotation read next.",
    color: "#7C3AED",
  },
  key_value: {
    label: "Key to Value",
    canvasLabel: "KV",
    instruction: "Connect a Key annotation to one Value annotation.",
    color: "#0F766E",
    sourceCategory: "key",
    targetCategory: "value",
  },
  table_cell: {
    label: "Table to Cell",
    canvasLabel: "Cell",
    instruction: "Connect a Table annotation to each Cell annotation.",
    color: "#C2410C",
    dash: [8, 4],
    sourceCategory: "table",
    targetCategory: "cell",
  },
};

// categoryLabelはrelation制約の説明に使うLabel categoryの表示名を返す。
export function categoryLabel(category: LabelCategory): string {
  return category.charAt(0).toUpperCase() + category.slice(1);
}
