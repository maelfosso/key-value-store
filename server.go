package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
	"github.com/maelfosso/key-value-store/store"
)

var (
	StoragePath = "/tmp/kv"
	Host        = "localhost"
	RaftPort    = "8081"
	log         = hclog.Default()
)

func main() {
	// Get port from env variables or set to 8080
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Info(fmt.Sprintf("Starting up on http://localhost:%s", port))

	if fromEnv := os.Getenv("STORAGE_PATH"); fromEnv != "" {
		StoragePath = fromEnv
	}

	if fromEnv := os.Getenv("RAFT_ADDRESS"); fromEnv != "" {
		Host = fromEnv
	}

	if fromEnv := os.Getenv("RAFT_PORT"); fromEnv != "" {
		RaftPort = fromEnv
	}

	leader := os.Getenv("RAFT_LEADER")
	config, err := store.NewRaftSetup(StoragePath, Host, RaftPort, leader)
	if err != nil {
		log.Error("couldn't set up Raft", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Use(config.Middleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		jw := json.NewEncoder(w)
		jw.Encode(map[string]string{"hello": "world"})
	})

	r.Post("/raft/add", config.AddHandler())

	r.Get("/key/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")

		data, err := config.Get(r.Context(), key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		w.Write([]byte(data))
	})

	r.Delete("/key/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := chi.URLParam(r, "key")

		err := config.Delete(r.Context(), key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		JSON(w, map[string]string{"status": "success"})
	})

	r.Post("/key/{key}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		key := chi.URLParam(r, "key")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		err = config.Set(r.Context(), key, string(body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]string{"error": err.Error()})
			return
		}

		JSON(w, map[string]string{"status": "success"})
	})

	http.ListenAndServe(":"+port, r)
}

// JSON encodes data to json and writes it to the http response
func JSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	b, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]string{"error": err.Error()})
		return
	}

	w.Write(b)
}

func Set(ctx context.Context, key, value string) error {
	data, err := loadData(ctx)
	if err != nil {
		return err
	}

	data[key] = value
	err = saveData(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

// Get gets the value at the specified key
func Get(ctx context.Context, key string) (string, error) {
	data, err := loadData(ctx)
	if err != nil {
		return "", err
	}

	return data[key], nil
}

func Delete(ctx context.Context, key string) error {
	data, err := loadData(ctx)
	if err != nil {
		return err
	}

	delete(data, key)

	err = saveData(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func dataPath() string {
	return filepath.Join(StoragePath, "data.json")
}

func loadData(ctx context.Context) (map[string]string, error) {
	empty := map[string]string{}
	emptyData, err := encode(map[string]string{})
	if err != nil {
		return empty, err
	}

	// First check if the folder exists and create it if it is missing
	if _, err := os.Stat(StoragePath); os.IsNotExist(err) {
		err = os.MkdirAll(StoragePath, 0755)
		if err != nil {
			return empty, err
		}
	}

	// Then check if the file exists and create it if it is missing
	if _, err := os.Stat(dataPath()); os.IsNotExist(err) {
		err := os.WriteFile(dataPath(), emptyData, 0644)
		if err != nil {
			return empty, err
		}
	}

	content, err := os.ReadFile(dataPath())
	if err != nil {
		return empty, err
	}

	return decode(content)
}

func saveData(ctx context.Context, data map[string]string) error {
	// First check if the folder exists and create it if it is missing
	if _, err := os.Stat(StoragePath); os.IsNotExist(err) {
		err = os.MkdirAll(StoragePath, 0755)
		if err != nil {
			return err
		}
	}

	encodedData, err := encode(data)
	if err != nil {
		return err
	}

	return os.WriteFile(dataPath(), encodedData, 0644)
}

func encode(data map[string]string) ([]byte, error) {
	encodedData := map[string]string{}
	for k, v := range data {
		ek := base64.URLEncoding.EncodeToString([]byte(k))
		ev := base64.URLEncoding.EncodeToString([]byte(v))
		encodedData[ek] = ev
	}

	return json.Marshal(encodedData)
}

func decode(data []byte) (map[string]string, error) {
	var jsonData map[string]string

	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	returnData := map[string]string{}
	for k, v := range jsonData {
		dk, err := base64.URLEncoding.DecodeString(k)
		if err != nil {
			return nil, err
		}

		dv, err := base64.URLEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}

		returnData[string(dk)] = string(dv)
	}

	return returnData, nil
}
