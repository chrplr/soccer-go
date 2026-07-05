package main

// Glue between the game and the pgzgo harness. Only images are embedded here;
// sounds and music stay with the game's own audio.go, which keeps the menu-theme
// and in-match crowd orchestration (StartMenuMusic / StartMatchAudio).

import (
	"embed"

	"github.com/chrplr/pgzgo"
)

// Assets is the game's name for the harness drawing surface.
type Assets = pgzgo.Screen

//go:embed images
var imagesFS embed.FS

// app is the running harness; input.go reads its keyboard snapshot.
var app *pgzgo.App
