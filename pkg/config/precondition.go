package config

type Precondition struct {
	IsAuthor            bool     `json:"is_author"`
	RequiredRoles       []string `json:"required_roles"`
	RequiredLabels      []string `json:"required_labels"`
	RequiredLabelPrefix []string `json:"required_label_prefix"`
}
