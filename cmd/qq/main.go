package main

import (
	"os"

	"github.com/ArturUshakov/qq-go/internal/app"
	"github.com/ArturUshakov/qq-go/internal/output"
)

func main() {
	application, err := app.New()
	if err != nil {
		output.Error("Ошибка инициализации: %s", err.Error())
		os.Exit(1)
	}
	os.Exit(application.Run(os.Args[1:]))
}
