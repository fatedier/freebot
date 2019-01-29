package config

type Precondition struct {
	IsAuthor            bool     `json:"is_author"`
	IsOwner             bool     `json:"is_owner"`
	IsQA                bool     `json:"is_qa"`
	RequiredLabels      []string `json:"required_labels"`
	RequiredLabelPrefix []string `json:"required_label_prefix"`
}
