package helpers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Configs map[string]interface{}
}

var cfg *Config

func init() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigFile(".env")
	_ = viper.ReadInConfig()

	service := viper.GetString("SERVICE_NAME")
	if service == "" {
		log.Fatal("Missing required env SERVICE_NAME")
	}

	environment := viper.GetString("ENVIRONMENT")
	if environment == "" {
		log.Fatal("Missing required env ENVIRONMENT")
	}

	var hostname string
	switch strings.ToUpper(environment) {
	case "LOCAL":
		hostname = "localhost"
	default:
		hostname = "config-service"
	}

	remoteURL := fmt.Sprintf("http://%v:31070/configs/%s", hostname, service)

	resp, err := http.Get(remoteURL)
	if err != nil {
		log.Fatalf("cannot fetch remote config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("remote config HTTP error: %d %s", resp.StatusCode, body)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("cannot read remote response body: %v", err)
	}

	viper.SetConfigType("json")
	if err := viper.MergeConfig(bytes.NewBuffer(raw)); err != nil {
		log.Fatalf("cannot merge remote config: %v", err)
	}

	cfg = &Config{Configs: viper.AllSettings()}
}

func ReadAll() map[string]interface{} {
	return cfg.Configs
}

func ReadConfig[T any](path string) (T, error) {
	var zero T
	key := "data." + path
	if !viper.IsSet(key) {
		return zero, fmt.Errorf("config not found: %s", path)
	}

	var anyVal interface{}
	switch (any)(zero).(type) {
	case int:
		anyVal = viper.GetInt(key)
	case int64:
		anyVal = viper.GetInt64(key)
	case float64:
		anyVal = viper.GetFloat64(key)
	case string:
		anyVal = viper.GetString(key)
	default:
		anyVal = viper.Get(key)
	}

	cast, ok := anyVal.(T)
	if !ok {
		return zero, fmt.Errorf("type assertion failed for key %s: got %T", path, anyVal)
	}
	return cast, nil
}
