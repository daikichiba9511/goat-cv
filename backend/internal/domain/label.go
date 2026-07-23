package domain

// LabelCategory identifies how a label participates in annotation tasks.
type LabelCategory string

const (
	// LabelCategoryObject marks labels used for object detection.
	LabelCategoryObject LabelCategory = "object"
	// LabelCategoryEntity marks labels used for information extraction entities.
	LabelCategoryEntity LabelCategory = "entity"
	// LabelCategoryKey marks labels used as keys in key-value extraction.
	LabelCategoryKey LabelCategory = "key"
	// LabelCategoryValue marks labels used as values in key-value extraction.
	LabelCategoryValue LabelCategory = "value"
	// LabelCategoryTable marks labels used for table regions.
	LabelCategoryTable LabelCategory = "table"
	// LabelCategoryCell marks labels used for table cells.
	LabelCategoryCell LabelCategory = "cell"
)

// LabelDefinition defines an annotation label within a project.
type LabelDefinition struct {
	ID        string
	ProjectID string
	Name      string
	Color     string
	Category  LabelCategory
}
