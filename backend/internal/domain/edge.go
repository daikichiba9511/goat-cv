package domain

// EdgeType identifies the semantic relationship between two annotations.
type EdgeType string

const (
	// EdgeTypeReadingOrder represents reading order from source to target.
	EdgeTypeReadingOrder EdgeType = "reading_order"
	// EdgeTypeKeyValue represents a key-value relation.
	EdgeTypeKeyValue EdgeType = "key_value"
	// EdgeTypeTableCell represents a table-to-cell relation.
	EdgeTypeTableCell EdgeType = "table_cell"
)

// Edge represents a directed relationship between two annotations.
type Edge struct {
	ID                 string
	ImageID            string
	SourceAnnotationID string
	TargetAnnotationID string
	Type               EdgeType
}
