package main

import (
	"github.com/joho/godotenv"
	"github.com/pgschema/pgschema/cmd"
)

func main() {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	cmd.Execute()
}
