package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func printError(err *DbResultError) {
	log.Printf("STATUS: %d, MSG: %s, ERR: %v", err.status, err.message, err.err)
}

func apiCategoriesGet(w http.ResponseWriter, r *http.Request) {
	categories, err := getTagCategories()
	if err != nil {
		printError(err)
		sendApiResult(w, err.status, err.message, nil)
	} else {
		sendApiResult(w, 200, fmt.Sprintf("Found %d categories", len(categories)), categories)
	}
}

func apiTagsGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name, present := query["name"]
	after_date, datePresent := query["after_date"]
	cleanName := ""
	if present {
		cleanName = name[0]
	}
	cleanDate := "2000-01-01"
	if datePresent {
		cleanDate = after_date[0]
	}
	tags, err := findTags(cleanName, cleanDate)
	if err != nil {
		printError(err)
		sendApiResult(w, err.status, err.message, nil)
	} else {
		sendApiResult(w, 200, fmt.Sprintf("Found %d Tags", len(tags)), tags)
	}
}

func apiTagGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	searchToken := vars["id"]
	tagId, err := strconv.Atoi(searchToken)
	if err != nil {
		// Must be a name
		res, err := getTagByName(searchToken)
		if err != nil {
			printError(err)
			sendApiResult(w, err.status, err.message, nil)
			return
		} else {
			sendApiResult(w, 200, "Found Tag", res)
			return
		}
	} else {
		// Must be an id
		res, err := getTagById(tagId)
		if err != nil {
			printError(err)
			sendApiResult(w, err.status, err.message, nil)
			return
		} else {
			sendApiResult(w, 200, "Found Tag", res)
			return
		}
	}
}

func apiTagNewPost(w http.ResponseWriter, r *http.Request) {
	var apiRequest Api_TagModel
	err := json.NewDecoder(r.Body).Decode(&apiRequest)
	if err != nil {
		sendApiResult(w, 400, "Failed to parse JSON to Tag Model", err)
		return
	}
	if apiRequest.PrimaryAlias == nil {
		sendApiResult(w, 400, "Must provide at minimum a primaryAlias", nil)
		return
	}
	_, findErr := _getTagAlias(*apiRequest.PrimaryAlias)
	if findErr != nil {
		sqlErr := createTag(apiRequest)
		if sqlErr != nil {
			printError(sqlErr)
			sendApiResult(w, sqlErr.status, sqlErr.message, nil)
			return
		} else {
			res, sqlErr := getTagByName(*apiRequest.PrimaryAlias)
			if sqlErr != nil {
				printError(sqlErr)
				sendApiResult(w, sqlErr.status, sqlErr.message, nil)
				return
			} else {
				sendApiResult(w, 200, "Created Tag", res)
				return
			}
		}
	} else {
		sendApiResult(w, 409, fmt.Sprintf("Tag with alias '%s' already exists", *apiRequest.PrimaryAlias), nil)
		return
	}
}

func apiTagPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["id"]
	var apiRequest Api_TagModel
	err := json.NewDecoder(r.Body).Decode(&apiRequest)
	if err != nil {
		sendApiResult(w, 400, "Failed to parse JSON to Tag Model", err)
		return
	}
	alias, findErr := _getTagAlias(name)
	if findErr != nil {
		sendApiResult(w, findErr.status, findErr.message, nil)
		return
	} else {
		sqlErr := updateTag(alias.tagId, apiRequest, nil)
		if sqlErr != nil {
			printError(sqlErr)
			sendApiResult(w, sqlErr.status, sqlErr.message, nil)
			return
		} else {
			res, sqlErr := getTagById(alias.tagId)
			if sqlErr != nil {
				printError(sqlErr)
				sendApiResult(w, sqlErr.status, sqlErr.message, nil)
				return
			} else {
				sendApiResult(w, 200, "Updated Tag", res)
				return
			}
		}
	}
}
