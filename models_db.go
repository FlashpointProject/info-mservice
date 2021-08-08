package main

import "database/sql"

type TagModel struct {
	id             int
	dateModified   string
	primaryAliasId int
	categoryId     int
	description    sql.NullString
}

type TagAliasModel struct {
	id    int
	tagId int
	name  string
}

type TagCategoryModel struct {
	id          int
	name        string
	color       string
	description sql.NullString
}
