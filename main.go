package main

import (
	"github.com/joho/godotenv"
	"github.com/pgplex/pgschema/cmd"
)

func main() {
	// Load .env file if it exists (silently ignore errors)
	_ = godotenv.Load()

	cmd.Execute()
}
