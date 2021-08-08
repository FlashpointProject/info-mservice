package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/tkanos/gonfig"
)

var master_key string = ""
var generic_keys []string = []string{}
var write_keys []string = []string{}

type ConfigFile struct {
	Port   string `json:"Port"`
	EsUri  string `json:"EsUri"`
	DbPath string `json:"DbPath"`
}

type ApiResult struct {
	Message string      `json:"message"`
	Result  interface{} `json:"result,omitempty"`
}

type GameSearch struct {
	Query   string `json:"query"`
	Fuzz    int    `json:"fuzz,omitempty"`
	Extreme bool   `json:"extreme,omitempty"`
	Page    int    `json:"page"`
	Limit   int    `json:"limit,omitempty"`
	Sort    string `json:"sort,omitempty"`
	Order   string `json:"order,omitempty"`
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func sendCustomApiResult(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func sendApiResult(w http.ResponseWriter, status int, message string, result interface{}) {
	apiResult := ApiResult{
		Message: message,
		Result:  result,
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResult)
}

func searchApi(w http.ResponseWriter, r *http.Request) {
	searchStruct := GameSearch{
		Fuzz:    0,
		Extreme: true,
		Page:    1,
		Limit:   50,
		Sort:    "title",
		Order:   "asc",
	}
	if r.Body != nil {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			err := dec.Decode(&searchStruct)
			if err != nil {
				sendApiResult(w, http.StatusBadRequest, "Unable to parse body", err)
				return
			}
		} else {
			sendApiResult(w, http.StatusBadRequest, "Body must be JSON", nil)
			return
		}
	}
	games, total, err := search(&searchStruct)
	if err != nil {
		log.Println(err)
		sendApiResult(w, http.StatusInternalServerError, "Server Error", nil)
		return
	}
	if len(games) == 0 {
		sendCustomApiResult(w, http.StatusOK, map[string]interface{}{
			"message":   "No Games Found",
			"last_page": 1,
			"result":    []interface{}{},
		})
		return
	}
	sendCustomApiResult(w, http.StatusOK, map[string]interface{}{
		"message":   fmt.Sprintf("Found %d Games", int(total)),
		"last_page": int(math.Ceil(total / float64(searchStruct.Limit))),
		"result":    games,
		"total":     int(total),
	})
}

func findGameById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	game_id := vars["id"]
	game, err := _findGameById(game_id)
	if err != nil {
		log.Println(err)
		sendApiResult(w, http.StatusInternalServerError, "Server Error", nil)
		return
	}
	if game == nil {
		sendApiResult(w, http.StatusNotFound, "Not Found", nil)
		return
	}
	sendApiResult(w, http.StatusOK, "Found Game", game)
}

func masterAuth(cb func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			sendApiResult(w, http.StatusBadRequest, "Invalid Authorization Header", nil)
			return
		}
		auth_key := auth[len("Bearer "):]
		if auth_key == master_key {
			cb(w, r)
			return
		}
		sendApiResult(w, http.StatusForbidden, "Forbidden", nil)
	}
}

func indexKey(auth_key string, arr []string) int {
	for i := 0; i < len(arr); i++ {
		if auth_key == arr[i] {
			return i
		}
	}
	return -1
}

func writeAuth(cb func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			sendApiResult(w, http.StatusBadRequest, "Invalid Authorization Header", nil)
			return
		}
		auth_key := auth[len("Bearer "):]
		if auth_key == master_key {
			cb(w, r)
			return
		}
		if indexKey(auth_key, write_keys) != -1 {
			cb(w, r)
			return
		}
		sendApiResult(w, http.StatusForbidden, "Forbidden", nil)
	}
}

func generalAuth(cb func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			sendApiResult(w, http.StatusBadRequest, "Invalid Authorization Header", nil)
			return
		}
		auth_key := auth[len("Bearer "):]
		if auth_key == master_key {
			cb(w, r)
			return
		}
		if indexKey(auth_key, write_keys) != -1 {
			cb(w, r)
			return
		}
		if indexKey(auth_key, generic_keys) != -1 {
			cb(w, r)
			return
		}
		sendApiResult(w, http.StatusForbidden, "Forbidden", nil)
	}
}

func listApiKeys(w http.ResponseWriter, r *http.Request) {
	var keys = map[string][]string{}
	keys["generic"] = generic_keys
	keys["write"] = write_keys
	sendApiResult(w, http.StatusOK, "Success", keys)
}

func loadConfig() ConfigFile {
	configuration := ConfigFile{}
	fileName := "./config.json"
	gonfig.GetConf(fileName, &configuration)
	return configuration
}

func loadWriteKeys() error {
	raw_keys, err := os.ReadFile("./write_keys")
	if err != nil {
		_, err = os.Create("./write_keys")
		if err != nil {
			return err
		}
		write_keys = []string{}
	} else {
		unclean_keys := strings.Split(string(raw_keys), "\n")
		write_keys = []string{}
		for i := 0; i < len(unclean_keys); i++ {
			clean_key := strings.TrimSpace(unclean_keys[i])
			if clean_key != "" {
				write_keys = append(write_keys, clean_key)
			}
		}
	}
	return nil
}

func loadGenericKeys() error {
	raw_keys, err := os.ReadFile("./generic_keys")
	if err != nil {
		_, err = os.Create("./generic_keys")
		if err != nil {
			return err
		}
		generic_keys = []string{}
	} else {
		unclean_keys := strings.Split(string(raw_keys), "\n")
		generic_keys = []string{}
		for i := 0; i < len(unclean_keys); i++ {
			clean_key := strings.TrimSpace(unclean_keys[i])
			if clean_key != "" {
				generic_keys = append(generic_keys, clean_key)
			}
		}
	}
	return nil
}

func deleteApiKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	api_key := vars["id"]
	key_idx := indexKey(api_key, generic_keys)
	if key_idx == -1 {
		key_idx := indexKey(api_key, write_keys)
		if key_idx == -1 {
			write_keys = append(write_keys[:key_idx], write_keys[key_idx+1:]...)
			saveWriteKeys()
			sendApiResult(w, http.StatusOK, "Deleted", nil)
		} else {
			sendApiResult(w, http.StatusNotFound, "Not Found", nil)
			return
		}
	}
	generic_keys = append(generic_keys[:key_idx], generic_keys[key_idx+1:]...)
	saveGenericKeys()
	sendApiResult(w, http.StatusOK, "Deleted", nil)
}

func genApiKey(w http.ResponseWriter, r *http.Request) {
	new_key := uuid.New().String()
	level_raw := r.URL.Query()["level"]
	present := len(level_raw) > 0
	level := "generic"
	if present {
		level = level_raw[0]
		if level == "write" {
			write_keys = append(write_keys, new_key)
			saveWriteKeys()
		} else if level == "generic" {
			generic_keys = append(generic_keys, new_key)
			saveGenericKeys()
		} else {
			sendApiResult(w, http.StatusBadRequest, "'level' param must be of ('write', 'generic')", nil)
			return
		}
		sendApiResult(w, http.StatusOK, fmt.Sprintf("New Key with '%s' auth level", level), new_key)
		return
	}
	generic_keys = append(generic_keys, new_key)
	saveGenericKeys()
	sendApiResult(w, http.StatusOK, fmt.Sprintf("New Key with '%s' auth level", level), new_key)
}

func saveWriteKeys() error {
	var content string = ""
	for i := 0; i < len(write_keys); i++ {
		content = content + "\n" + write_keys[i]
	}
	return os.WriteFile("./write_keys", []byte(content), 0644)
}

func saveGenericKeys() error {
	var content string = ""
	for i := 0; i < len(generic_keys); i++ {
		content = content + "\n" + generic_keys[i]
	}
	return os.WriteFile("./generic_keys", []byte(content), 0644)
}

func handleRequests(port string) {
	router := mux.NewRouter()
	router.HandleFunc("/", homePage)
	router.HandleFunc("/api/games", generalAuth(searchApi)).Methods("GET")
	router.HandleFunc("/api/game/{id}", generalAuth(findGameById)).Methods("GET")
	router.HandleFunc("/api/keys", masterAuth(listApiKeys)).Methods("GET")
	router.HandleFunc("/api/key/{id}", masterAuth(deleteApiKey)).Methods("DELETE")
	router.HandleFunc("/api/key", masterAuth(genApiKey)).Methods("POST")
	router.HandleFunc("/api/tag/{id}", generalAuth(apiTagGet)).Methods("GET")
	router.HandleFunc("/api/tag/{id}", writeAuth(apiTagPost)).Methods("POST", "PUT", "PATCH")
	router.HandleFunc("/api/tag", writeAuth(apiTagNewPost)).Methods("POST")
	router.HandleFunc("/api/tags", generalAuth(apiTagsGet)).Methods("GET")
	router.HandleFunc("/api/categories", generalAuth(apiCategoriesGet)).Methods("GET")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), router))
}

func main() {
	raw_master_key, err := os.ReadFile("./master_key")
	if err != nil {
		_, err := os.Create("./master_key")
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Fatal("Please fill in the master_key file")
	} else {
		master_key = string(raw_master_key)
		if master_key == "" {
			log.Fatal("Please fill in the master_key file")
		}
	}
	err = loadGenericKeys()
	if err != nil {
		log.Printf("Error loading generic keys: %v", err)
		return
	}
	err = loadWriteKeys()
	if err != nil {
		log.Printf("Error loading write keys: %v", err)
		return
	}
	config := loadConfig()
	log.Printf("%v", config)
	err = esInit(config.EsUri)
	if err != nil {
		log.Printf("esInit Error: %v", err)
		return
	}
	err = dbInit(config.DbPath)
	if err != nil {
		log.Printf("dbInit Error: %v", err)
		return
	}
	// count, err := populateEs()
	// if err != nil {
	// 	log.Printf("populateEs Error: %v", err)
	// 	return
	// }
	// log.Printf("Total Games Loaded: %d", count)
	log.Println("API Initialized!")
	handleRequests(config.Port)
}
