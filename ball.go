package main

import "math"

type Ball struct {
	Actor
	vel    Vec2
	owner  *Player
	timer  int
	shadow *Actor
}

func NewBall() *Ball {
	sh := newActor("balls", 0, 0)
	return &Ball{
		Actor:  newActor("ball", HalfLevelW, HalfLevelH),
		shadow: &sh,
	}
}

// collide reports whether player p can acquire the ball (its hold-off timer has
// expired and it is close enough).
func (b *Ball) collide(p *Player) bool {
	return p.timer < 0 && p.vpos.Sub(b.vpos).Length() <= DribbleDistX
}

func (b *Ball) update() {
	b.timer--

	if b.owner != nil {
		// Dribbling: ease towards a point just ahead of the owner. Different X/Y
		// offsets give an elliptical dribble to suit the game's perspective.
		newX := avg(b.vpos.X, b.owner.vpos.X+DribbleDistX*sin(float64(b.owner.dir)))
		newY := avg(b.vpos.Y, b.owner.vpos.Y-DribbleDistY*cos(float64(b.owner.dir)))
		if onPitch(newX, newY) {
			b.vpos = Vec2{newX, newY}
		} else {
			// Off the pitch - the owner loses the ball.
			b.owner.timer = 60
			b.vel = angleToVec(b.owner.dir).Mul(3)
			b.owner = nil
		}
	} else {
		// Free ball physics, one axis at a time, respecting goal openings.
		boundsX := pitchBoundsX
		if math.Abs(b.vpos.Y-HalfLevelH) > HalfPitchH {
			boundsX = goalBoundsX
		}
		boundsY := pitchBoundsY
		if math.Abs(b.vpos.X-HalfLevelW) < HalfGoalW {
			boundsY = goalBoundsY
		}
		b.vpos.X, b.vel.X = ballPhysics(b.vpos.X, b.vel.X, boundsX)
		b.vpos.Y, b.vel.Y = ballPhysics(b.vpos.Y, b.vel.Y, boundsY)
	}

	b.shadow.vpos = b.vpos

	// Look for a player that can take the ball.
	for _, target := range game.players {
		if (b.owner == nil || b.owner.team != target.team) && b.collide(target) {
			if b.owner != nil {
				b.owner.timer = 60
			}
			b.timer = game.difficulty.holdoffTimer
			b.owner = target
			game.teams[target.team].activeControlPlayer = target
		}
	}

	// If owned, decide whether to kick.
	if b.owner == nil {
		return
	}
	team := game.teams[b.owner.team]

	var targetablePlayers []posTeam
	for _, p := range game.players {
		if p.team == b.owner.team && targetable(p, b.owner) {
			targetablePlayers = append(targetablePlayers, p)
		}
	}
	for _, g := range game.goals {
		if g.team == b.owner.team && targetable(g, b.owner) {
			targetablePlayers = append(targetablePlayers, g)
		}
	}

	var target posTeam
	if len(targetablePlayers) > 0 {
		bestD := math.Inf(1)
		for _, tp := range targetablePlayers {
			if d := tp.Pos().Sub(b.owner.vpos).Length(); d < bestD {
				bestD, target = d, tp
			}
		}
	}

	var doShoot bool
	if team.human() {
		doShoot = team.controls.shoot()
	} else {
		doShoot = b.timer <= 0 && target != nil &&
			costAt(target.Pos(), b.owner.team, 0) < costAt(b.owner.vpos, b.owner.team, 0)
	}
	if !doShoot {
		return
	}

	game.PlaySound("kick", 4)

	var vec Vec2
	if target != nil {
		// Kick towards the target. For a human passing to a player, iterate to
		// lead the target (assuming they keep running the same way); otherwise a
		// single pass with no lead.
		r := 0.0
		iterations := 1
		if _, isPlayer := target.(*Player); team.human() && isPlayer {
			iterations = 8
		}
		for i := 0; i < iterations; i++ {
			t := target.Pos().Add(angleToVec(b.owner.dir).Mul(r))
			v, length := safeNormalise(t.Sub(b.vpos))
			vec = v
			r = HumanPlayerWithoutBallSpeed * float64(steps(length))
		}
	} else {
		// No target - kick straight ahead and guess the likely receiver.
		vec = angleToVec(b.owner.dir)
		point := b.vpos.Add(vec.Mul(250))
		target = nearestPlayerOfTeam(b.owner.team, point)
	}

	if pl, ok := target.(*Player); ok {
		game.teams[b.owner.team].activeControlPlayer = pl
	}
	b.owner.timer = 10
	b.vel = vec.Mul(KickStrength)
	b.owner = nil
}
