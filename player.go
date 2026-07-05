package main

import (
	"math"
	"strconv"
)

// playerAnchorX/Y is the sprite anchor (feet) used for players and their shadows.
const (
	playerAnchorX = 25
	playerAnchorY = 37
)

type Player struct {
	Actor
	home      Vec2
	team      int
	dir       int
	animFrame float64
	timer     int
	shadow    *Actor
	peer      *Player
	mark      Marker
	lead      float64
	hasLead   bool
}

func NewPlayer(x, y float64, team int) *Player {
	// Players start in kickoff positions (team 0 below halfway, team 1 above).
	kickoffY := y/2 + 550 - float64(team*400)
	sh := newAnchoredActor("blank", 0, 0, playerAnchorX, playerAnchorY)
	return &Player{
		Actor:     newAnchoredActor("blank", x, kickoffY, playerAnchorX, playerAnchorY),
		home:      Vec2{x, y},
		team:      team,
		animFrame: -1,
		shadow:    &sh,
	}
}

func (p *Player) Pos() Vec2   { return p.vpos }
func (p *Player) TeamID() int { return p.team }

// active reports whether the ball is close enough (on the Y axis) for this player
// to be doing something useful rather than holding its home position.
func (p *Player) active() bool {
	return math.Abs(game.ball.vpos.Y-p.home.Y) < 400
}

// targetable reports whether target is a good pass target for source.
func targetable(target posTeam, source *Player) bool {
	v0, d0 := safeNormalise(target.Pos().Sub(source.vpos))

	// For CPU teams, avoid passes likely to be intercepted.
	if !game.teams[source.team].human() {
		for _, p := range game.players {
			v1, d1 := safeNormalise(p.vpos.Sub(source.vpos))
			if p.team != target.TeamID() && d1 > 0 && d1 < d0 && v0.Dot(v1) > 0.8 {
				return false
			}
		}
	}

	// A good target is a same-team object ahead of, near, and roughly faced by us.
	return target.TeamID() == source.team && d0 > 0 && d0 < 300 &&
		v0.Dot(angleToVec(source.dir)) > 0.8
}

func (p *Player) update() {
	p.timer--

	target := p.home
	speed := PlayerDefaultSpeed

	myTeam := game.teams[p.team]
	preKickoff := game.kickoffPlayer != nil
	iAmKickoff := game.kickoffPlayer == p
	ball := game.ball

	if p == myTeam.activeControlPlayer && myTeam.human() && (!preKickoff || iAmKickoff) {
		// Human-controlled active player.
		if ball.owner == p {
			speed = HumanPlayerWithBallSpeed
		} else {
			speed = HumanPlayerWithoutBallSpeed
		}
		target = p.vpos.Add(myTeam.controls.move(speed))
	} else if ball.owner != nil {
		if ball.owner == p {
			// CPU player with the ball: try five directions and pick the cheapest.
			bestCost := math.Inf(1)
			bestPos := p.vpos
			for d := -2; d <= 2; d++ {
				pos := p.vpos.Add(angleToVec(p.dir + d).Mul(3))
				if c := costAt(pos, p.team, math.Abs(float64(d))); c < bestCost {
					bestCost, bestPos = c, pos
				}
			}
			target = bestPos
			speed = CPUPlayerWithBallBaseSpeed + game.difficulty.speedBoost
		} else if ball.owner.team == p.team {
			// Team-mate has the ball - run somewhere useful (unique per player).
			if p.active() {
				direction := 1.0
				if p.team == 0 {
					direction = -1
				}
				target.X = (ball.vpos.X + target.X) / 2
				target.Y = (ball.vpos.Y + 400*direction + target.Y) / 2
			}
		} else {
			// Opponent has the ball.
			if p.hasLead {
				target = ball.owner.vpos.Add(angleToVec(ball.owner.dir).Mul(p.lead))
				target.X = math.Max(AIMinX, math.Min(AIMaxX, target.X))
				target.Y = math.Max(AIMinY, math.Min(AIMaxY, target.Y))
				otherTeam := 1 - p.team
				speed = LeadPlayerBaseSpeed
				if game.teams[otherTeam].human() {
					speed += game.difficulty.speedBoost
				}
			} else if p.mark.active() {
				if myTeam.human() {
					target = ball.vpos
				} else {
					vec, length := safeNormalise(ball.vpos.Sub(p.mark.Pos()))
					if _, isGoal := p.mark.(*Goal); isGoal {
						length = math.Min(150, length)
					} else {
						length /= 2
					}
					target = p.mark.Pos().Add(vec.Mul(length))
				}
			}
		}
	} else {
		// No one has the ball.
		if (preKickoff && iAmKickoff) || (!preKickoff && p.active()) {
			// Intercept: simulate the ball's path and aim where we can reach it.
			target = ball.vpos
			vel := ball.vel
			frame := 0.0
			for target.Sub(p.vpos).Length() > PlayerInterceptBallSpeed*frame+DribbleDistX && vel.Length() > 0.5 {
				target = target.Add(vel)
				vel = vel.Mul(Drag)
				frame++
			}
			speed = PlayerInterceptBallSpeed
		} else if preKickoff {
			target.Y = p.vpos.Y
		}
	}

	vec, distance := safeNormalise(target.Sub(p.vpos))

	var targetDir int
	if distance > 0 {
		distance = math.Min(distance, speed)
		targetDir = vecToAngle(vec)
		// Move per axis so we can slide along the level edge.
		if allowMovement(p.vpos.X+vec.X*distance, p.vpos.Y) {
			p.vpos.X += vec.X * distance
		}
		if allowMovement(p.vpos.X, p.vpos.Y+vec.Y*distance) {
			p.vpos.Y += vec.Y * distance
		}
		p.animFrame = math.Mod(p.animFrame+math.Max(distance, 1.5), 72)
	} else {
		targetDir = vecToAngle(ball.vpos.Sub(p.vpos))
		p.animFrame = -1
	}

	// Rotate one step per frame towards the target direction.
	dirDiff := targetDir - p.dir
	rotTable := [8]int{0, 1, 1, 1, 1, 7, 7, 7}
	p.dir = pmod(p.dir+rotTable[pmod(dirDiff, 8)], 8)

	suffix := strconv.Itoa(p.dir) + strconv.Itoa(floorDiv(int(p.animFrame), 18)+1)
	p.image = "player" + strconv.Itoa(p.team) + suffix
	p.shadow.image = "players" + suffix
	p.shadow.vpos = p.vpos
}
