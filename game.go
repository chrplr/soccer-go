package main

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
)

type Game struct {
	teams       [2]*Team
	difficulty  Difficulty
	scoreTimer  int
	scoringTeam int

	players       []*Player
	goals         [2]*Goal
	ball          *Ball
	kickoffPlayer *Player
	cameraFocus   Vec2

	assets *Assets
	audio  *Audio
}

func NewGame(p1, p2 *Controls, difficulty int, assets *Assets, audio *Audio) *Game {
	g := &Game{
		teams:       [2]*Team{NewTeam(p1), NewTeam(p2)},
		difficulty:  difficulties[difficulty],
		scoringTeam: 1,
		assets:      assets,
		audio:       audio,
	}

	if g.teams[0].human() {
		audio.StartMatchAudio()
	} else {
		audio.StartMenuMusic()
	}

	g.reset()
	return g
}

// reset sets up players, goals and ball for a kickoff (game start or after a goal).
func (g *Game) reset() {
	randomOffset := func(x float64) float64 { return x + float64(rand.Intn(65)-32) }

	g.players = nil
	for _, pos := range playerStartPos {
		g.players = append(g.players, NewPlayer(randomOffset(pos[0]), randomOffset(pos[1]), 0))
		g.players = append(g.players, NewPlayer(randomOffset(LevelW-pos[0]), randomOffset(LevelH-pos[1]), 1))
	}

	// Peers are opposing players at mirrored ends of the list (0&13, 1&12, ...).
	n := len(g.players)
	for i, p := range g.players {
		p.peer = g.players[n-1-i]
	}

	g.goals = [2]*Goal{NewGoal(0), NewGoal(1)}

	g.teams[0].activeControlPlayer = g.players[0]
	g.teams[1].activeControlPlayer = g.players[1]

	// The team that conceded (or team 0 at game start) kicks off.
	otherTeam := 1 - g.scoringTeam
	g.kickoffPlayer = g.players[otherTeam]
	g.kickoffPlayer.vpos = Vec2{HalfLevelW - 30 + float64(otherTeam*60), HalfLevelH}

	g.ball = NewBall()
	g.cameraFocus = g.ball.vpos
}

func (g *Game) update() {
	g.scoreTimer--

	if g.scoreTimer == 0 {
		g.reset()
	} else if g.scoreTimer < 0 && math.Abs(g.ball.vpos.Y-HalfLevelH) > HalfPitchH {
		g.PlaySound("goal", 2)
		if g.ball.vpos.Y < HalfLevelH {
			g.scoringTeam = 0
		} else {
			g.scoringTeam = 1
		}
		g.teams[g.scoringTeam].score++
		g.scoreTimer = 60
	}

	// Reset marking each frame.
	for _, p := range g.players {
		p.mark = p.peer
		p.hasLead = false
		p.lead = 0
	}

	if g.ball.owner != nil {
		o := g.ball.owner
		pos, team := o.vpos, o.team
		ownersTargetGoal := g.goals[team]
		otherTeam := 1 - team

		if g.difficulty.goalieEnabled {
			// Nearest opposing player becomes the goalie, marking the goal.
			var nearest *Player
			bestD := math.Inf(1)
			for _, p := range g.players {
				if p.team != team {
					if d := p.vpos.Sub(ownersTargetGoal.vpos).Length(); d < bestD {
						bestD, nearest = d, p
					}
				}
			}
			o.peer.mark = nearest.mark
			nearest.mark = ownersTargetGoal
		}

		// Candidate pursuers: opponents able to take the ball and not the goalie.
		var l []*Player
		for _, p := range g.players {
			if p.team == team || p.timer > 0 {
				continue
			}
			if g.teams[otherTeam].human() && p == g.teams[otherTeam].activeControlPlayer {
				continue
			}
			if _, isGoal := p.mark.(*Goal); isGoal {
				continue
			}
			l = append(l, p)
		}
		sort.SliceStable(l, func(i, j int) bool {
			return l[i].vpos.Sub(pos).Length() < l[j].vpos.Sub(pos).Length()
		})

		// Split into up-field (a) and down-field (b) of the owner, then interleave.
		var a, b []*Player
		for _, p := range l {
			upfield := p.vpos.Y > pos.Y
			if team != 0 {
				upfield = p.vpos.Y < pos.Y
			}
			if upfield {
				a = append(a, p)
			} else {
				b = append(b, p)
			}
		}
		zipped := zipInterleave(a, b)
		if len(zipped) > 0 {
			zipped[0].hasLead = true
			zipped[0].lead = LeadDistance1
		}
		if g.difficulty.secondLeadEnabled && len(zipped) > 1 {
			zipped[1].hasLead = true
			zipped[1].lead = LeadDistance2
		}

		g.kickoffPlayer = nil
	}

	for _, p := range g.players {
		p.update()
	}
	g.ball.update()

	owner := g.ball.owner
	for teamNum := 0; teamNum < 2; teamNum++ {
		teamObj := g.teams[teamNum]
		if teamObj.human() && teamObj.controls.shoot() {
			// Manual switch to the nearest team-mate to the ball, weighting
			// up-field players when an opponent owns the ball.
			var best *Player
			bestKey := math.Inf(1)
			for _, p := range g.players {
				if p.team != teamNum {
					continue
				}
				distToBall := p.vpos.Sub(g.ball.vpos).Length()
				goalDir := float64(2*teamNum - 1)
				key := distToBall
				if owner != nil && (p.vpos.Y-g.ball.vpos.Y)*goalDir < 0 {
					key = distToBall / 2
				}
				if key < bestKey {
					bestKey, best = key, p
				}
			}
			g.teams[teamNum].activeControlPlayer = best
		}
	}

	// Ease the camera towards the ball, at most 8 px/frame.
	cameraBallVec, distance := safeNormalise(g.cameraFocus.Sub(g.ball.vpos))
	if distance > 0 {
		g.cameraFocus = g.cameraFocus.Sub(cameraBallVec.Mul(math.Min(distance, 8)))
	}
}

func (g *Game) draw() {
	offsetX := math.Max(0, math.Min(LevelW-Width, g.cameraFocus.X-Width/2))
	offsetY := math.Max(0, math.Min(LevelH-Height, g.cameraFocus.Y-Height/2))

	g.assets.Blit("pitch", -offsetX, -offsetY)

	// Ball and players, sorted back-to-front by world Y, drawn with their shadows.
	type drawItem struct{ main, shadow *Actor }
	items := []drawItem{{&g.ball.Actor, g.ball.shadow}}
	for _, p := range g.players {
		items = append(items, drawItem{&p.Actor, p.shadow})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].main.vpos.Y < items[j].main.vpos.Y
	})

	g.goals[0].Draw(g.assets, offsetX, offsetY)
	for _, it := range items {
		it.main.Draw(g.assets, offsetX, offsetY)
	}
	for _, it := range items {
		it.shadow.Draw(g.assets, offsetX, offsetY)
	}
	g.goals[1].Draw(g.assets, offsetX, offsetY)

	// Active-player arrows (human teams only).
	for t := 0; t < 2; t++ {
		if g.teams[t].human() {
			pos := g.teams[t].activeControlPlayer.vpos
			g.assets.Blit("arrow"+strconv.Itoa(t), pos.X-offsetX-11, pos.Y-offsetY-45)
		}
	}
}

// PlaySound plays a game sound, but never on the menu screen.
func (g *Game) PlaySound(name string, count int) {
	if state == StateMenu {
		return
	}
	g.audio.PlaySound(name, count)
}

// nearestPlayerOfTeam returns the player on team nearest to point.
func nearestPlayerOfTeam(team int, point Vec2) *Player {
	var best *Player
	bestD := math.Inf(1)
	for _, p := range game.players {
		if p.team == team {
			if d := p.vpos.Sub(point).Length(); d < bestD {
				bestD, best = d, p
			}
		}
	}
	return best
}

// zipInterleave alternates elements of a and b (a0, b0, a1, b1, ...), reproducing
// the original's zip(a+[None,None], b+[None,None]) padding behaviour.
func zipInterleave(a, b []*Player) []*Player {
	aa := append(append([]*Player{}, a...), nil, nil)
	bb := append(append([]*Player{}, b...), nil, nil)
	n := len(aa)
	if len(bb) < n {
		n = len(bb)
	}
	var out []*Player
	for i := 0; i < n; i++ {
		if aa[i] != nil {
			out = append(out, aa[i])
		}
		if bb[i] != nil {
			out = append(out, bb[i])
		}
	}
	return out
}
