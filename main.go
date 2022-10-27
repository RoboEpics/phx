package main

import (
	"embed"

	"gitlab.roboepics.com/roboepics/xerac/phoenix/phx/cmd"
)

var (
	//go:embed scripts/*
	scripts embed.FS
)

func main() {
	cmd.Execute(scripts)
}
