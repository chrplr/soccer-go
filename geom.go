package main

import "math"

// Window and level geometry.
const (
	Width       = 800
	Height      = 480
	HalfWindowW = Width / 2

	LevelW     = 1000
	LevelH     = 1400
	HalfLevelW = LevelW / 2
	HalfLevelH = LevelH / 2

	HalfPitchW = 442
	HalfPitchH = 622

	GoalWidth = 186
	GoalDepth = 20
	HalfGoalW = GoalWidth / 2

	AIMinX = 78
	AIMaxX = LevelW - 78
	AIMinY = 98
	AIMaxY = LevelH - 98

	LeadDistance1 = 10
	LeadDistance2 = 50

	DribbleDistX = 18
	DribbleDistY = 16

	// Player speeds. Those with "Base" can be boosted for CPU teams.
	PlayerDefaultSpeed          = 2.0
	CPUPlayerWithBallBaseSpeed  = 2.6
	PlayerInterceptBallSpeed    = 2.75
	LeadPlayerBaseSpeed         = 2.9
	HumanPlayerWithBallSpeed    = 3.0
	HumanPlayerWithoutBallSpeed = 3.3

	KickStrength = 11.5
	Drag         = 0.98
)

// Bounds (min, max) for the various regions.
var (
	pitchBoundsX = [2]float64{HalfLevelW - HalfPitchW, HalfLevelW + HalfPitchW}
	pitchBoundsY = [2]float64{HalfLevelH - HalfPitchH, HalfLevelH + HalfPitchH}
	goalBoundsX  = [2]float64{HalfLevelW - HalfGoalW, HalfLevelW + HalfGoalW}
	goalBoundsY  = [2]float64{HalfLevelH - HalfPitchH - GoalDepth, HalfLevelH + HalfPitchH + GoalDepth}

	pitchRect = rect{pitchBoundsX[0], pitchBoundsY[0], HalfPitchW * 2, HalfPitchH * 2}
	goal0Rect = rect{goalBoundsX[0], goalBoundsY[0], GoalWidth, GoalDepth}
	goal1Rect = rect{goalBoundsX[0], goalBoundsY[1] - GoalDepth, GoalWidth, GoalDepth}
)

var playerStartPos = [7][2]float64{
	{350, 550}, {650, 450}, {200, 850}, {500, 750}, {800, 950}, {350, 1250}, {650, 1150},
}

// rect is a minimal axis-aligned rectangle with pygame-style collidepoint.
type rect struct {
	x, y, w, h float64
}

func (r rect) collidePoint(x, y float64) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

// Difficulty settings, indexed 0 (easy) to 2 (hard).
type Difficulty struct {
	goalieEnabled     bool
	secondLeadEnabled bool
	speedBoost        float64
	holdoffTimer      int
}

var difficulties = [3]Difficulty{
	{false, false, 0, 120},
	{false, true, 0.1, 90},
	{true, true, 0.2, 60},
}

// sin/cos for angles 0..7, where 0 is up, 1 is up+right, 2 is right, etc.
func sin(x float64) float64 { return math.Sin(x * math.Pi / 4) }
func cos(x float64) float64 { return sin(x + 2) }

// vecToAngle converts a vector to an angle in the range 0..7.
func vecToAngle(v Vec2) int {
	return int(4*math.Atan2(v.X, -v.Y)/math.Pi+8.5) % 8
}

// angleToVec converts an angle 0..7 to a direction vector.
func angleToVec(angle int) Vec2 {
	a := float64(angle)
	return Vec2{sin(a), -cos(a)}
}

// avg returns b when a and b are within 1 of each other, else their mean.
func avg(a, b float64) float64 {
	if math.Abs(b-a) < 1 {
		return b
	}
	return (a + b) / 2
}

// onPitch reports whether a dribbled ball position is on the pitch or in a goal.
func onPitch(x, y float64) bool {
	return pitchRect.collidePoint(x, y) || goal0Rect.collidePoint(x, y) || goal1Rect.collidePoint(x, y)
}

// ballPhysics advances one axis of the ball, bouncing at bounds and applying drag.
func ballPhysics(pos, vel float64, bounds [2]float64) (float64, float64) {
	pos += vel
	if pos < bounds[0] || pos > bounds[1] {
		pos, vel = pos-vel, -vel
	}
	return pos, vel * Drag
}

// steps returns how many physics frames the ball takes to travel distance.
func steps(distance float64) int {
	n, vel := 0, KickStrength
	for distance > 0 && vel > 0.25 {
		distance -= vel
		n++
		vel *= Drag
	}
	return n
}

// allowMovement reports whether (x, y) is a legal player position (inside the
// level, and not inside or behind a goal).
func allowMovement(x, y float64) bool {
	switch {
	case math.Abs(x-HalfLevelW) > HalfLevelW:
		return false
	case math.Abs(x-HalfLevelW) < HalfGoalW+20:
		return math.Abs(y-HalfLevelH) < HalfPitchH
	default:
		return math.Abs(y-HalfLevelH) < HalfLevelH
	}
}

// costAt scores a position for a CPU player with the ball; lower is better.
func costAt(pos Vec2, team int, handicap float64) float64 {
	ownGoalY := LevelH - 78.0
	if team == 1 {
		ownGoalY = 78
	}
	ownGoalPos := Vec2{HalfLevelW, ownGoalY}
	inverseOwnGoalDistance := 3500 / pos.Sub(ownGoalPos).Length()

	result := inverseOwnGoalDistance
	for _, p := range game.players {
		if p.team != team {
			result += 4000 / math.Max(24, p.vpos.Sub(pos).Length())
		}
	}
	result += (pos.X-HalfLevelW)*(pos.X-HalfLevelW)/200 - pos.Y*float64(4*team-2)
	result += handicap
	return result
}
