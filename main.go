package main

import (
	"embed"

	"github.com/Titovilal/middleman/cmd"
)

//go:embed defaults
var defaultsFS embed.FS

func main() {
	cmd.SetDefaultsFS(defaultsFS)
	cmd.Execute()
}
