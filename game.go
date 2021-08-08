package main

import (
	"encoding/json"
	"net/http"
)

type Game struct {
	Id                  string `json:"game_id"`
	Title               string `json:"title"`
	AlternateTitles     string `json:"alternateTitles"`
	Developer           string `json:"developer"`
	Publisher           string `json:"publisher"`
	Series              string `json:"series"`
	DateAdded           string `json:"dateAdded"`
	DateModified        string `json:"dateModified"`
	Platform            string `json:"platform"`
	PlayMode            string `json:"playMode"`
	Status              string `json:"status"`
	Notes               string `json:"notes"`
	Source              string `json:"source"`
	ApplicationPath     string `json:"applicationPath"`
	LaunchCommand       string `json:"launchCommand"`
	ReleaseDate         string `json:"releaseDate"`
	Version             string `json:"version"`
	OriginalDescription string `json:"originalDescription"`
	Language            string `json:"language"`
	TagsStr             string `json:"tagsStr"`
}

func returnAllGames(w http.ResponseWriter, r *http.Request) {
	var games [2]Game
	games[0] = Game{
		Title:    "Test Game",
		Platform: "Test Platform",
	}
	games[1] = Game{
		Title:    "Test twooo",
		Platform: "Wiggly Woooooo",
	}
	json.NewEncoder(w).Encode(games)
}
