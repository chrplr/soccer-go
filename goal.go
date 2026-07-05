package main

import (
	"math"
	"strconv"
)

// posTeam is anything with a world position and a team: a Player or a Goal. Used
// as pass targets.
type posTeam interface {
	Pos() Vec2
	TeamID() int
}

// Marker is anything a player can be assigned to mark: a Player or a Goal.
type Marker interface {
	active() bool
	Pos() Vec2
}

type Goal struct {
	Actor
	team int
}

func NewGoal(team int) *Goal {
	y := 0.0
	if team != 0 {
		y = LevelH
	}
	return &Goal{Actor: newActor("goal"+strconv.Itoa(team), HalfLevelW, y), team: team}
}

// active reports whether the ball is within 500 pixels of the goal on the Y axis.
func (g *Goal) active() bool {
	return math.Abs(game.ball.vpos.Y-g.vpos.Y) < 500
}

func (g *Goal) Pos() Vec2   { return g.vpos }
func (g *Goal) TeamID() int { return g.team }

type Team struct {
	controls            *Controls
	activeControlPlayer *Player
	score               int
}

func NewTeam(controls *Controls) *Team { return &Team{controls: controls} }

func (t *Team) human() bool { return t.controls != nil }
