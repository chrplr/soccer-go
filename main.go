package main

import (
	"flag"
	"strconv"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/chrplr/pgzgo"
)

// pmod is Python-style modulo (result has the divisor's sign).
func pmod(a, b int) int {
	m := a % b
	if m < 0 {
		m += b
	}
	return m
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

type State int

const (
	StateMenu State = iota
	StatePlay
	StateGameOver
)

type MenuState int

const (
	MenuNumPlayers MenuState = iota
	MenuDifficulty
)

var (
	state          State
	game           *Game
	menuState      MenuState
	menuNumPlayers = 1
	menuDifficulty = 0

	assets *Assets
	audio  *Audio
)

func update() {
	switch state {
	case StateMenu:
		if startPressed() {
			if menuState == MenuNumPlayers {
				if menuNumPlayers == 1 {
					menuState = MenuDifficulty
				} else {
					state = StatePlay
					game = NewGame(NewControls(0), NewControls(1), 2, assets, audio)
				}
			} else {
				state = StatePlay
				game = NewGame(NewControls(0), nil, menuDifficulty, assets, audio)
			}
		} else {
			selectionChange := 0
			if keyJustPressed(sdl.SCANCODE_DOWN) {
				selectionChange = 1
			} else if keyJustPressed(sdl.SCANCODE_UP) {
				selectionChange = -1
			}
			if selectionChange != 0 {
				audio.Play("move")
				if menuState == MenuNumPlayers {
					if menuNumPlayers == 1 {
						menuNumPlayers = 2
					} else {
						menuNumPlayers = 1
					}
				} else {
					menuDifficulty = pmod(menuDifficulty+selectionChange, 3)
				}
			}
		}
		game.update()

	case StatePlay:
		maxScore := game.teams[0].score
		if game.teams[1].score > maxScore {
			maxScore = game.teams[1].score
		}
		if maxScore == 9 && game.scoreTimer == 1 {
			state = StateGameOver
		} else {
			game.update()
		}

	case StateGameOver:
		if startPressed() {
			state = StateMenu
			menuState = MenuNumPlayers
			game = NewGame(nil, nil, 2, assets, audio)
		}
	}
}

func draw() {
	game.draw()

	switch state {
	case StateMenu:
		var image string
		if menuState == MenuNumPlayers {
			image = "menu0" + strconv.Itoa(menuNumPlayers)
		} else {
			image = "menu1" + strconv.Itoa(menuDifficulty)
		}
		assets.Blit(image, 0, 0)

	case StatePlay:
		assets.Blit("bar", HalfWindowW-176, 0)
		for i := 0; i < 2; i++ {
			assets.Blit("s"+strconv.Itoa(game.teams[i].score), float64(HalfWindowW+7-39*i), 6)
		}
		if game.scoreTimer > 0 {
			assets.Blit("goal", HalfWindowW-300, Height/2-88)
		}

	case StateGameOver:
		winner := 0
		if game.teams[1].score > game.teams[0].score {
			winner = 1
		}
		assets.Blit("over"+strconv.Itoa(winner), 0, 0)
		for i := 0; i < 2; i++ {
			img := "l" + strconv.Itoa(i) + strconv.Itoa(game.teams[i].score)
			assets.Blit(img, float64(HalfWindowW+25-125*i), 144)
		}
	}
}

func main() {
	flag.Parse()

	a, err := pgzgo.New(pgzgo.Config{
		Title:  "Substitute Soccer",
		Width:  Width,
		Height: Height,
		Images: imagesFS,
		// Audio is nil: soccer keeps its own mixer (menu theme + crowd loop).
	})
	if err != nil {
		panic(err)
	}
	defer a.Close()

	app = a
	assets = a.Screen
	audio = NewAudio()
	defer audio.Destroy()

	state = StateMenu
	menuState = MenuNumPlayers
	game = NewGame(nil, nil, 2, assets, audio)

	a.Loop(
		func(*pgzgo.App) { update() },
		func(*pgzgo.App) { draw() },
	)
}
