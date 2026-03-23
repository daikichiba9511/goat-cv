package domain

type LabelCategory string

const (
	LabelCategoryObject LabelCategory = "object"
	LabelCategoryEntity LabelCategory = "entity"
	LabelCategoryKey    LabelCategory = "key"
	LabelCategoryValue  LabelCategory = "value"
	LabelCategoryTable  LabelCategory = "table"
	LabelCategoryCell   LabelCategory = "cell"
)

type LabelDefinition struct {
	ID        string
	ProjectID string
	Name      string
	Color     string
	Category  LabelCategory
}
