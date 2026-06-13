package main

import (
	"embed"
	"io/fs"

	"go-nx-admin/cmd/commands"
)

//go:embed all:frontend/dist
var frontendDist embed.FS

func main() {
	sub, err := fs.Sub(frontendDist, "frontend/dist")
	if err != nil {
		panic("embedded dist not found, run 'make build' first: " + err.Error())
	}
	commands.Execute(sub)
}
