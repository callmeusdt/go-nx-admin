package main

import (
	"embed"
	"io/fs"
	"log"

	"go-nx-admin/cmd/commands"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	sub, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		log.Fatalf("embedded frontend not found, run 'make build-frontend' first: %v", err)
	}
	commands.Execute(sub)
}
