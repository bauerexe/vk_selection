package main

import (
	"github.com/sirupsen/logrus"
	"vk_otbor/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		logrus.Fatalf("fatal: %v", err)
	}
}
