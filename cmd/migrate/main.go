package main

import (
	"fmt"
	"os"

	"github.com/Jota-Music/jota/internal/app"
)

func main() {
	if !app.HasLegacyStore() {
		fmt.Println("No legacy store found. Nothing to migrate.")
		return
	}

	fmt.Println("Legacy store detected. Migrating...")
	summary, err := app.MigrateLegacyStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(summary)
	fmt.Println("The old store has been imported and kept aside. You can now open Jota.")
}
