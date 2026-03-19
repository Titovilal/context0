package main

import (
	"embed"

	"github.com/Titovilal/middleman/cmd"
)

//go:embed .mdm/agents.md .mdm/templates .mdm/guides
var defaultsFS embed.FS

func main() {
	cmd.SetDefaultsFS(defaultsFS)
	cmd.Execute()
}
