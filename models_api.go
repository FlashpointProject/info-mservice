package main

type Api_TagModel struct {
	PrimaryAlias *string   `json:"name,omitempty"`
	Aliases      *[]string `json:"aliases,omitempty"`
	Category     *string   `json:"category,omitempty"`
	Description  *string   `json:"description,omitempty"`
}

type Api_TagCategoryModel struct {
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}
