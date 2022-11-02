package main

import (
	"embed"

	"github.com/RoboEpics/phx/cmd"
)

var (
	//go:embed scripts/*
	scripts embed.FS
)

func main() {
	cmd.Execute(scripts)
}
