package config

import (
	"encoding/json"
	"io/ioutil"
	"platform-service-bus/internal/pkg/adapter"
)

// Config описывает конфигурацию всего приложения
type Config struct {
	Adapters []adapter.Adapter
}

// fileReader описывает функцию чтения данных из файла
type fileReader func(filename string) ([]byte, error)

// Load загружает указанный файл конфигурации
// На выходе получаем map или ошибку
func Load(configPath string, opts ...interface{}) (Config, error) {
	// Можно предоставить свою функцию чтения данных
	reader := ioutil.ReadFile
	if len(opts) > 0 {
		reader = opts[0].(fileReader)
	}
	// Читаем и парсим JSON
	config := Config{}
	configData, err := reader(configPath)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return config, err
	}
	return config, nil
}
