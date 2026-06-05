package main

import (
	"bubble/cmd"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
