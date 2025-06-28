package helpers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type ConfigHelper struct {
	configs map[string]interface{}
	once    sync.Once
}

var helper = &ConfigHelper{}

func ensureLoaded() {
	helper.once.Do(func() {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}

		configHost := "config-service"
		if os.Getenv("NODE_ENV") == "local" {
			configHost = "localhost"
		}
		serviceName := os.Getenv("SERVICE_NAME")
		if serviceName == "" {
			serviceName = "unknown"
		}

		configURL := fmt.Sprintf("http://%s:31070/configs/%s", configHost, serviceName)

		resp, err := http.Get(configURL)
		if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			panic(fmt.Sprintf("Cannot load configurations for %s", serviceName))
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var response struct {
			Data map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			panic("Failed to parse configuration JSON: " + err.Error())
		}
		helper.configs = response.Data
	})
}

// ReadAll returns all loaded configs
func ReadAll() map[string]interface{} {
	ensureLoaded()
	return helper.configs
}

func ReadConfig[T any](path string) (T, error) {
	ensureLoaded()
	var zero T
	valI, ok := deepLookup(helper.configs, path)
	if !ok {
		return zero, fmt.Errorf("config not found: %s", path)
	}

	switch v := valI.(type) {
	case T:
		return v, nil
	case float64:
		// Attempt conversion if T is an integer type
		var converted any
		switch any(zero).(type) {
		case int:
			converted = int(v)
		case int64:
			converted = int64(v)
		default:
			return zero, fmt.Errorf("cannot convert float64 to %T", zero)
		}
		return converted.(T), nil
	default:
		return zero, fmt.Errorf("config at %s is not of expected type, got %T", path, valI)
	}
}

// ReadConfigWithDefault reads the config value or returns a default value
func ReadConfigWithDefault[T any](path string, defaultValue T) T {
	val, err := ReadConfig[T](path)
	if err != nil {
		return defaultValue
	}
	return val
}

// deepLookup navigates a nested map by dot path
func deepLookup(m map[string]interface{}, path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = m
	for _, part := range parts {
		switch typed := current.(type) {
		case map[string]interface{}:
			current = typed[part]
		default:
			return nil, false
		}
	}
	return current, true
}
