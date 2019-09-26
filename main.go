package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"platform-service-bus/internal/pkg/config"
)

func main() {
	// Аргументы командной строки
	flagLog := flag.String("log", "platform-service-bus.log", "File to put logs into")
	flag.Parse()

	// Настройка логирования
	file, err := os.OpenFile(*flagLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, file)
		log.SetOutput(mw)
	} else {
		log.Info("Не удалось открыть файл для логирования")
	}

	log.Info("Загружаем config.json")
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
