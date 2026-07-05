package main

import "github.com/Zyko0/go-sdl3/sdl"

// keyDown / keyJustPressed read the harness keyboard snapshot (held and
// rising-edge). Pressed is edge-detected against the previous frame, so unlike
// the original it is safe to call more than once per frame.
func keyDown(sc sdl.Scancode) bool        { return app.Keyboard.Held(sc) }
func keyJustPressed(sc sdl.Scancode) bool { return app.Keyboard.Pressed(sc) }

// Controls maps one player's keys and reports movement/shoot intents.
type Controls struct {
	up, down, left, right, shootKey sdl.Scancode
}

func NewControls(playerNum int) *Controls {
	if playerNum == 0 {
		return &Controls{
			up: sdl.SCANCODE_UP, down: sdl.SCANCODE_DOWN,
			left: sdl.SCANCODE_LEFT, right: sdl.SCANCODE_RIGHT,
			shootKey: sdl.SCANCODE_SPACE,
		}
	}
	return &Controls{
		up: sdl.SCANCODE_W, down: sdl.SCANCODE_S,
		left: sdl.SCANCODE_A, right: sdl.SCANCODE_D,
		shootKey: sdl.SCANCODE_LSHIFT,
	}
}

// move returns the movement vector for the held direction keys, scaled by speed.
func (c *Controls) move(speed float64) Vec2 {
	dx, dy := 0.0, 0.0
	if keyDown(c.left) {
		dx = -1
	} else if keyDown(c.right) {
		dx = 1
	}
	if keyDown(c.up) {
		dy = -1
	} else if keyDown(c.down) {
		dy = 1
	}
	return Vec2{dx, dy}.Mul(speed)
}

func (c *Controls) shoot() bool {
	return keyJustPressed(c.shootKey)
}
