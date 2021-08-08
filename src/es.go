package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/mitchellh/mapstructure"
)

var client *elasticsearch.Client = nil

func esInit() error {
	cfg := elasticsearch.Config{
		Addresses: []string{
			"http://10.10.0.6:9200",
		},
	}
	var err error
	client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}
	log.Println(elasticsearch.Version)
	info, err := client.Info()
	if err != nil {
		return err
	}
	log.Println(info)
	return nil
}

func _indexGames(games []Game) error {
	ctx := context.Background()
	cfg := esutil.BulkIndexerConfig{
		Client: client,
		Index:  "gameinfo",
	}
	indexer, _ := esutil.NewBulkIndexer(cfg)
	for _, workedGame := range games {
		indexer.Add(ctx, esutil.BulkIndexerItem{
			Index:  "gameinfo",
			Action: "index",
			Body:   esutil.NewJSONReader(workedGame),
		})
	}
	err := indexer.Close(ctx)
	if err != nil {
		return err
	}
	return nil
}

func indexGames(games []Game) error {
	indicies := []string{"gameinfo"}
	_, err := client.Indices.Delete(indicies)
	if err != nil {
		return err
	}
	client.Indices.Create("gameinfo")
	var workingGames []Game
	for _, game := range games {
		workingGames = append(workingGames, game)
		if len(workingGames) >= 1000 {
			err := _indexGames(workingGames)
			if err != nil {
				return err
			}
			workingGames = []Game{}
		}
	}
	if len(workingGames) > 0 {
		err := _indexGames(workingGames)
		if err != nil {
			return err
		}
	}
	return nil
}

func _findGameById(id string) (*Game, error) {
	ctx := context.Background()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"game_id": id,
			},
		},
	}
	reader := esutil.NewJSONReader(query)
	res, err := client.Search(
		client.Search.WithIndex("gameinfo"),
		client.Search.WithContext(ctx),
		client.Search.WithBody(reader),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var body map[string]interface{}
	json.NewDecoder(res.Body).Decode(&body)
	if res.IsError() {
		return nil, fmt.Errorf("Response Error: %v", body)
	}
	for _, hit := range body["hits"].(map[string]interface{})["hits"].([]interface{}) {
		game := mapHitToGame(hit)
		return &game, nil
	}
	return nil, nil
}

func search(gameSearch *GameSearch) ([]Game, float64, error) {
	ctx := context.Background()
	query := map[string]interface{}{
		"size": gameSearch.Limit,
		"from": (gameSearch.Page - 1) * gameSearch.Limit,
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": fmt.Sprintf("%s~%d", gameSearch.Query, gameSearch.Fuzz),
			},
		},
		"sort": [1]map[string]interface{}{
			{gameSearch.Sort: gameSearch.Order},
		},
	}
	reader := esutil.NewJSONReader(query)
	res, err := client.Search(
		client.Search.WithIndex("gameinfo"),
		client.Search.WithBody(reader),
		client.Search.WithPretty(),
		client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, 0.0, err
	}
	defer res.Body.Close()
	var body map[string]interface{}
	json.NewDecoder(res.Body).Decode(&body)
	if res.IsError() {
		return nil, 0.0, fmt.Errorf("Response Error: %v", body)
	}
	var games []Game
	for _, hit := range body["hits"].(map[string]interface{})["hits"].([]interface{}) {
		game := mapHitToGame(hit)
		games = append(games, game)
	}
	total := body["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)
	return games, total, nil
}

func mapHitToGame(hit interface{}) Game {
	var game Game
	source := hit.(map[string]interface{})["_source"]
	mapstructure.Decode(source, &game)
	game.Id = fmt.Sprintf("%v", source.(map[string]interface{})["game_id"])
	return game
}
