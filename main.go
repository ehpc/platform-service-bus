package main

import (
	"platform-service-bus/internal/pkg/config"
)

func main() {
	// Подгружаем конфигурацию
	configObject, err := config.Load("config/config.json")
	if err != nil {
		panic(err)
	}

	// Для каждого адаптера поднимаем свой сервер
	finish := make(chan bool)
	for _, adapter := range configObject.Adapters {
		currentAdapter := adapter
		go currentAdapter.StartServer()
	}
	<-finish
}
