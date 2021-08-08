package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func dbInit() error {
	var err error
	db, err = sql.Open("sqlite3", "./flashpoint.sqlite")
	if err != nil {
		return err
	}
	_, err = db.Exec("PRAGMA foreign_keys=off")
	if err != nil {
		return err
	}
	_, err = db.Exec(`PRAGMA journal_mode=WAL`)
	if err != nil {
		return err
	}
	return nil
}

func populateEs() (int, error) {
	rows, err := db.Query("SELECT * FROM game")
	if err != nil {
		return 0, err
	}
	var totalRows = 0
	var games []Game = []Game{}
	for rows.Next() {
		totalRows += 1
		var id string
		var parentGameId, title, alternateTitles, series, developer, publisher sql.NullString
		var dateAdded, dateModified string
		var platform string
		var broken, extreme bool
		var playMode, status, notes, source string
		var applicationPath, launchCommand, releaseDate, version string
		var originalDescription, language, library, orderTitle string
		var activeDataId, activeDataOnDisk sql.NullString
		var tagsStr string
		err = rows.Scan(&id, &parentGameId, &title, &alternateTitles, &series, &developer, &publisher,
			&dateAdded, &dateModified,
			&platform,
			&broken, &extreme,
			&playMode, &status, &notes, &source,
			&applicationPath, &launchCommand, &releaseDate, &version,
			&originalDescription, &language, &library, &orderTitle,
			&activeDataId, &activeDataOnDisk,
			&tagsStr)
		if err != nil {
			defer rows.Close()
			return 0, err
		}
		game := Game{
			Id:                  id,
			Title:               nullStringToVal(title),
			AlternateTitles:     nullStringToVal(alternateTitles),
			Developer:           nullStringToVal(developer),
			Publisher:           nullStringToVal(publisher),
			Series:              nullStringToVal(series),
			DateAdded:           dateAdded,
			DateModified:        dateModified,
			Platform:            platform,
			PlayMode:            playMode,
			Status:              status,
			Notes:               notes,
			Source:              source,
			ApplicationPath:     applicationPath,
			LaunchCommand:       launchCommand,
			ReleaseDate:         releaseDate,
			Version:             version,
			OriginalDescription: originalDescription,
			Language:            language,
			TagsStr:             tagsStr,
		}
		games = append(games, game)
		if err != nil {
			defer rows.Close()
			return 0, err
		}
	}
	defer rows.Close()
	indexGames(games)
	return totalRows, nil
}

func nullStringToVal(s sql.NullString) string {
	if s.Valid {
		return s.String
	} else {
		return ""
	}
}

func _getTagAlias(name string) (*TagAliasModel, *DbResultError) {
	rows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.name = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &DbResultError{status: 404, message: "Failed to find Tag Alias with name " + name, err: nil}
		} else {
			return nil, &DbResultError{status: 500, message: "", err: err}
		}
	}
	defer rows.Close()
	rowPresent := rows.Next()
	if rowPresent == false {
		return nil, &DbResultError{status: 404, message: "Failed to find Tag Alias with name " + name, err: nil}
	}
	var alias TagAliasModel
	err = rows.Scan(&alias.id, &alias.tagId, &alias.name)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	return &alias, nil
}

func _createTagAlias(name string, tagId int, tx *sql.Tx) (int, *DbResultError) {
	res, err := tx.Exec("INSERT INTO tag_alias (tagId, name) VALUES (?, ?)", tagId, name)
	if err != nil {
		return 0, &DbResultError{status: 500, message: "", err: err}
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, &DbResultError{status: 500, message: "", err: err}
	}
	log.Println(id)
	return int(id), nil
}

func _getTagCategoryByName(name string) (*TagCategoryModel, *DbResultError) {
	rows, err := db.Query("SELECT * FROM tag_category WHERE tag_category.name = ?", name)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	defer rows.Close()
	if rows.Next() == false {
		return nil, &DbResultError{status: 400, message: "Tag Category Not Found", err: err}
	}
	var category TagCategoryModel
	err = rows.Scan(&category.id, &category.name, &category.color, &category.description)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	return &category, nil
}

func _getTagCategory(id int) (*TagCategoryModel, *DbResultError) {
	rows, err := db.Query("SELECT * FROM tag_category WHERE tag_category.id = ?", id)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	defer rows.Close()
	if rows.Next() == false {
		return nil, &DbResultError{status: 400, message: "Tag Category Not Found", err: err}
	}
	var category TagCategoryModel
	err = rows.Scan(&category.id, &category.name, &category.color, &category.description)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	return &category, nil
}

func _validateAlias(name string, parentId int) (bool, *DbResultError) {
	// If alias already exists, make sure it can be applied to the existing tag
	alias, err := _getTagAlias(name)
	if err != nil && err.status == 500 {
		return false, err
	}
	if alias != nil {
		// Alias already exists, must be on same tag to change alias
		if alias.tagId != parentId {
			// Alias is already on another tag, can't add
			return false, err
		}
	}
	return true, nil
}

func getTagCategories() ([]*Api_TagCategoryModel, *DbResultError) {
	categoryRows, err := db.Query("SELECT * FROM tag_category")
	var categories []*Api_TagCategoryModel = []*Api_TagCategoryModel{}
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	for categoryRows.Next() {
		var category TagCategoryModel
		categoryRows.Scan(&category.id, &category.name, &category.color, &category.description)
		desc := nullStringToVal(category.description)
		apiCategory := Api_TagCategoryModel{
			Name:        &category.name,
			Color:       &category.color,
			Description: &desc,
		}
		categories = append(categories, &apiCategory)
	}
	return categories, nil
}

func getTagByName(name string) (*Api_TagModel, *DbResultError) {
	aliasRows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.name = ?", name)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	if aliasRows.Next() == false {
		return nil, &DbResultError{status: 404, message: "Tag Not Found", err: nil}
	}
	var alias TagAliasModel
	err = aliasRows.Scan(&alias.id, &alias.tagId, &alias.name)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	} else {
		return getTagById(alias.tagId)
	}
}

func getTagById(id int) (*Api_TagModel, *DbResultError) {
	// Find the Tag
	rows, err := db.Query("SELECT * FROM tag WHERE tag.id = ?", id)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, &DbResultError{status: 404, message: "Tag Not Found", err: nil}
		}
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	rows.Next()
	var dbTag TagModel
	err = rows.Scan(&dbTag.id, &dbTag.dateModified, &dbTag.primaryAliasId, &dbTag.categoryId, &dbTag.description)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	aliasRows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.tagId = ?", dbTag.id)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	// Find all aliases
	var aliases []string
	var pAlias string
	for aliasRows.Next() {
		var alias TagAliasModel
		err := aliasRows.Scan(&alias.id, &alias.tagId, &alias.name)
		if err != nil {
			return nil, &DbResultError{status: 500, message: "", err: err}
		}
		if alias.id == dbTag.primaryAliasId {
			pAlias = alias.name
		} else {
			aliases = append(aliases, alias.name)
		}
	}
	// Build response
	category, dbErr := _getTagCategory(dbTag.categoryId)
	if err != nil {
		return nil, dbErr
	}
	desc := nullStringToVal(dbTag.description)
	res := Api_TagModel{
		PrimaryAlias: &pAlias,
		Aliases:      &aliases,
		Category:     &category.name,
		Description:  &desc,
	}
	return &res, nil
}

func deleteTag(name string) *DbResultError {
	alias, err := _getTagAlias(name)
	if err != nil {
		return err
	}
	ctx := context.Background()
	tx, sqlErr := db.BeginTx(ctx, nil)
	if sqlErr != nil {
		return &DbResultError{status: 500, message: "", err: sqlErr}
	}
	defer tx.Rollback()
	_, sqlErr = tx.Exec("DELETE FROM tag_alias WHERE tag_alias.tagId = ?", alias.tagId)
	if sqlErr != nil {
		return &DbResultError{status: 500, message: "", err: sqlErr}
	}
	_, sqlErr = tx.Exec("DELETE FROM tag WHERE tag.id = ?", alias.tagId)
	if sqlErr != nil {
		return &DbResultError{status: 500, message: "", err: sqlErr}
	}
	return nil
}

func createTag(update Api_TagModel) *DbResultError {
	if update.PrimaryAlias == nil {
		return &DbResultError{status: 400, message: "Primary Alias Required", err: nil}
	}
	if update.Category == nil {
		category := "default"
		update.Category = &category
	}
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	defer tx.Rollback()
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	// Make sure tag doesn't already exist
	rows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.name = ?", update.PrimaryAlias)
	if err != nil {
		if err != sql.ErrNoRows {
			return &DbResultError{status: 500, message: "", err: err}
		}
		// Tag doesn't exist, good
	} else {
		// Tag exists
		defer rows.Close()
		return &DbResultError{status: 400, message: "Tag already exists with this primary alias", err: err}
	}
	tagRes, err := tx.Exec("INSERT INTO tag (description) VALUES (?)", update.Description)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	tagId, err := tagRes.LastInsertId()
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	aliasRes, err := tx.Exec("INSERT INTO tag_alias (tagId, name) VALUES (?, ?)", tagId, update.PrimaryAlias)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	aliasId, err := aliasRes.LastInsertId()
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	_, err = tx.Exec("UPDATE tag SET primaryAliasId = ? WHERE tag.id = ?", aliasId, tagId)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	return updateTag(int(tagId), update, tx)
}

func updateTag(tagId int, update Api_TagModel, tx *sql.Tx) *DbResultError {
	rows, err := db.Query("SELECT * FROM tag WHERE tag.id = ?", tagId)
	if err != nil {
		if err == sql.ErrNoRows {
			return &DbResultError{status: 404, message: "No Tag Found", err: nil}
		}
		return &DbResultError{status: 500, message: "", err: err}
	}
	defer rows.Close()
	rows.Next()
	var curTag TagModel
	err = rows.Scan(&curTag.id, &curTag.dateModified, &curTag.primaryAliasId, &curTag.categoryId, &curTag.description)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	rows.Close()
	// -- Update Validation --
	// Primary Alias
	if update.PrimaryAlias != nil {
		valid, err := _validateAlias(strings.TrimSpace(*update.PrimaryAlias), tagId)
		if err != nil {
			return err
		}
		if !valid {
			return nil
		}
	}
	// Aliases
	if update.Aliases != nil {
		for _, name := range *update.Aliases {
			valid, err := _validateAlias(strings.TrimSpace(name), tagId)
			if err != nil {
				return err
			}
			if !valid {
				return nil
			}
		}
	}
	// Category Id
	if update.Category != nil {
		newCategory, err := _getTagCategoryByName(*update.Category)
		if err != nil {
			return err
		}
		if newCategory == nil {
			// Tag category doesn't exist
			return nil
		}
	}
	// -- Update Application --
	if tx == nil {
		ctx := context.Background()
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return &DbResultError{status: 500, message: "", err: err}
		}
	}
	if update.PrimaryAlias != nil {
		log.Println("Setting New pAlias")
		existing, err := _getTagAlias(strings.TrimSpace(*update.PrimaryAlias))
		if err != nil && err.status == 500 {
			log.Fatal(err)
			return err
		}
		if existing != nil {
			log.Printf("New ID: %d", existing.id)
			curTag.primaryAliasId = existing.id
		} else {
			newId, err := _createTagAlias(strings.TrimSpace(*update.PrimaryAlias), tagId, tx)
			if err != nil {
				log.Fatal(err)
				return err
			}
			log.Printf("New ID: %d", newId)
			curTag.primaryAliasId = newId
		}
	}
	if update.Category != nil {
		newCategory, err := _getTagCategoryByName(*update.Category)
		if err != nil {
			return err
		}
		curTag.categoryId = newCategory.id
	}
	if update.Description != nil {
		curTag.description.String = *update.Description
		curTag.description.Valid = true
	}
	if update.Aliases != nil {
		for _, alias := range *update.Aliases {
			existing, err := _getTagAlias(strings.TrimSpace(alias))
			if err != nil && err.status == 500 {
				tx.Rollback()
				log.Fatal(err)
				return err
			}
			if existing == nil {
				_, err := _createTagAlias(strings.TrimSpace(alias), tagId, tx)
				if err != nil {
					tx.Rollback()
					log.Fatal(err)
					return err
				}
			}
		}
	}
	aliasRows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.tagId = ?", curTag.id)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	defer aliasRows.Close()
	if update.Aliases != nil {
		for aliasRows.Next() {
			var alias TagAliasModel
			err = aliasRows.Scan(&alias.id, &alias.tagId, &alias.name)
			if err != nil {
				return &DbResultError{status: 500, message: "", err: err}
			}
			var keep = false
			for _, updatedAlias := range *update.Aliases {
				if strings.TrimSpace(strings.ToLower(updatedAlias)) == strings.ToLower(alias.name) {
					keep = true
				}
			}
			if update.PrimaryAlias != nil && strings.TrimSpace(strings.ToLower(*update.PrimaryAlias)) == strings.ToLower(alias.name) {
				keep = true
			}
			// Preserve Primary Alias is a new Primary Alias wasn't given
			if update.PrimaryAlias == nil && alias.id == curTag.primaryAliasId {
				keep = true
			}
			if keep == false {
				// Delete tag alias
				_, err := tx.Exec("DELETE FROM tag_alias WHERE tag_alias.id = ?", alias.id)
				if err != nil {
					return &DbResultError{status: 500, message: "", err: err}
				}
			}
		}
	}
	aliasRows.Close()
	_, err = tx.Exec("UPDATE tag SET categoryId = ?, primaryAliasId = ?, description = ? WHERE tag.id = ?",
		curTag.categoryId,
		curTag.primaryAliasId,
		curTag.description,
		curTag.id)
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	err = tx.Commit()
	if err != nil {
		return &DbResultError{status: 500, message: "", err: err}
	}
	log.Println("DOING COMMIT")
	return nil
}

func findTags(partial string) ([]Api_TagModel, *DbResultError) {
	formedPartial := fmt.Sprintf("%%%s%%", partial)
	aliasRows, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.name LIKE ?", formedPartial)
	if err != nil {
		return nil, &DbResultError{status: 500, message: "", err: err}
	}
	defer aliasRows.Close()
	var checkedTagIds []int
	var tags []Api_TagModel = []Api_TagModel{}
	for aliasRows.Next() {
		var apiTag Api_TagModel
		var alias TagAliasModel
		err = aliasRows.Scan(&alias.id, &alias.tagId, &alias.name)
		if err != nil {
			return nil, &DbResultError{status: 500, message: "", err: err}
		}
		if containsId(checkedTagIds, alias.tagId) {
			// Skip already checked tags
			continue
		}

		row := db.QueryRow("SELECT * FROM tag WHERE tag.id = ?", alias.tagId)
		var dbTag TagModel
		err = row.Scan(&dbTag.id, &dbTag.dateModified, &dbTag.primaryAliasId, &dbTag.categoryId, &dbTag.description)
		if err != nil {
			return nil, &DbResultError{status: 500, message: "", err: err}
		}

		desc := nullStringToVal(dbTag.description)
		category, dbErr := _getTagCategory(dbTag.categoryId)
		if dbErr != nil {
			return nil, dbErr
		}
		apiTag.Description = &desc
		apiTag.Category = &category.name

		otherAliases, err := db.Query("SELECT * FROM tag_alias WHERE tag_alias.tagId = ?", dbTag.id)
		if err != nil {
			return nil, &DbResultError{status: 500, message: "", err: err}
		}
		var aliases []string = []string{}
		for otherAliases.Next() {
			var nextAlias TagAliasModel
			err = otherAliases.Scan(&nextAlias.id, &nextAlias.tagId, &nextAlias.name)
			if err != nil {
				return nil, &DbResultError{status: 500, message: "", err: err}
			}
			if dbTag.primaryAliasId == nextAlias.id {
				apiTag.PrimaryAlias = &nextAlias.name
			} else {
				aliases = append(aliases, nextAlias.name)
			}
		}
		apiTag.Aliases = &aliases
		tags = append(tags, apiTag)
		checkedTagIds = append(checkedTagIds, dbTag.id)
	}
	return tags, nil
}

func containsId(arr []int, id int) bool {
	for _, i := range arr {
		if i == id {
			return true
		}
	}
	return false
}
