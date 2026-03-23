package domain

type EdgeType string

const (
	EdgeTypeReadingOrder EdgeType = "reading_order"
	EdgeTypeKeyValue     EdgeType = "key_value"
	EdgeTypeTableCell    EdgeType = "table_cell"
)

type Edge struct {
	ID                 string
	ImageID            string
	SourceAnnotationID string
	TargetAnnotationID string
	Type               EdgeType
}
