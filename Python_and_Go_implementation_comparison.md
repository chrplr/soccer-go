# Substitute Soccer — Python vs. Go implementation comparison

This document compares the Go port in this folder with the original `soccer.py`.
Substitute Soccer is a top-down football game with a 14-player match, a
scrolling camera, and a genuinely elaborate AI (pass targeting via dot products,
lead/marking assignment, cost-based movement, ball interception by simulation).
It is the most **vector- and AI-heavy** of the ports, which makes it the best
showcase for how Python's operator overloading and duck typing translate to Go.
The Go version is a faithful translation; the differences are mechanical
consequences of Go's static typing, the lack of class inheritance/operator
overloading, and swapping Pygame Zero for
[go-sdl3](https://github.com/Zyko0/go-sdl3).

---

## 1. File organisation

Python is a single 1,126-line module. The Go port splits it by concern:

| Python section | Go file |
|---|---|
| `update()`, `draw()`, state machine, menu, mixer setup | `main.go` |
| `Game` class (update loop, camera, draw) | `game.go` |
| `Player` (+ `targetable`) | `player.go` |
| `Ball` | `ball.go` |
| `Goal`, `Team`, and the target/marker interfaces | `goal.go` |
| `MyActor` base | `actor.go` |
| `Vector2` replacement + `safe_normalise` | `vec.go` |
| geometry, constants, `sin/cos`, `cost`, `ball_physics`, `steps` | `geom.go` |
| `Controls`, `key_just_pressed` | `input.go` |
| image cache | pgzgo `Screen` (harness) |
| `play_sound` / music / crowd | `audio.go` (bespoke, over pgzgo) |

---

## 2. `pygame.Vector2` → a `Vec2` value type with methods

Soccer leans on vectors constantly, including **dot products** for AI decisions,
so this is the most consequential difference. Python uses `Vector2` with
overloaded operators (`+ - *`), where `*` between two vectors means dot product.
Go has no operator overloading, so `vec.go` defines a value type with methods:

```go
type Vec2 struct{ X, Y float64 }
func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Length() float64    { return math.Hypot(a.X, a.Y) }
```

The AI's pass-targeting dot products translate directly:

```python
if p.team != target.team and d1 > 0 and d1 < d0 and v0*v1 > 0.8:   # dot product
return target.team == source.team and d0 > 0 and d0 < 300 and v0 * angle_to_vec(source.dir) > 0.8
```
```go
if p.team != target.TeamID() && d1 > 0 && d1 < d0 && v0.Dot(v1) > 0.8 { ... }
return target.TeamID() == source.team && d0 > 0 && d0 < 300 &&
       v0.Dot(angleToVec(source.dir)) > 0.8
```

`safe_normalise` (returning the unit vector **and** original length) maps to a
two-value return, and using **value semantics** means Python's explicit copies
like `Vector2(self.home)` are just plain assignments in Go.

---

## 3. The star translation: duck typing → interfaces

In Python, a pass target or a marked object can be **either a `Player` or a
`Goal`**, and the code just accesses `.vpos`, `.team`, `.active()` on whatever it
gets. Go is statically typed, so the port introduces small interfaces capturing
exactly the methods each role needs:

```go
// A pass target: a Player or a Goal.
type posTeam interface {
    Pos() Vec2
    TeamID() int
}
// Something a player can mark: a Player or a Goal.
type Marker interface {
    active() bool
    Pos() Vec2
}
```

Both `*Player` and `*Goal` implement these, so the mixed lists become typed
slices of the interface:

```go
var targetablePlayers []posTeam
for _, p := range game.players { if p.team == b.owner.team && targetable(p, b.owner) { targetablePlayers = append(targetablePlayers, p) } }
for _, g := range game.goals   { if g.team == b.owner.team && targetable(g, b.owner) { targetablePlayers = append(targetablePlayers, g) } }
```

And Python's `isinstance(...)` checks become **type assertions**:

| Python | Go |
|---|---|
| `isinstance(target, Player)` | `if pl, ok := target.(*Player); ok { ... }` |
| `isinstance(self.mark, Goal)` | `if _, isGoal := p.mark.(*Goal); isGoal { ... }` |

This is the cleanest example in the whole project of Go interfaces standing in
for Python duck typing.

---

## 4. `min`/`sorted` with `key=` → explicit loops and `sort.Slice`

The AI uses `min(..., key=dist_key(...))` and `sorted(..., key=...)` all over.
Python's `dist_key` is a **closure factory** returning a distance function; Go
inlines the comparison into a loop that tracks the best candidate:

```python
target = min(targetable_players, key=dist_key(self.owner.vpos))
```
```go
bestD := math.Inf(1)
for _, tp := range targetablePlayers {
    if d := tp.Pos().Sub(b.owner.vpos).Length(); d < bestD { bestD, target = d, tp }
}
```

Sorting the pursuit candidates uses `sort.SliceStable`:

```go
sort.SliceStable(l, func(i, j int) bool {
    return l[i].vpos.Sub(pos).Length() < l[j].vpos.Sub(pos).Length()
})
```

### A Python-specific workaround that Go doesn't need

`cost()` returns a **tuple `(result, pos)`** in Python purely to dodge a crash:
`min` over the five candidate directions compares tuples, and if two costs tie it
would try to compare the `Vector2` positions with `<`, which `Vector2` doesn't
support — hence the `key=lambda element: element[0]`. In Go, the cost is just a
`float64` and the min loop compares floats directly, so `costAt` drops the tuple
entirely:

```go
for d := -2; d <= 2; d++ {
    pos := p.vpos.Add(angleToVec(p.dir + d).Mul(3))
    if c := costAt(pos, p.team, math.Abs(float64(d))); c < bestCost { bestCost, bestPos = c, pos }
}
```

---

## 5. `zip(a+NONE2, b+NONE2)` interleave → a helper

The lead-player selection interleaves up-field and down-field candidate lists,
padding each with `[None, None]` so the result always has at least two entries,
then filtering the `None`s out:

```python
NONE2 = [None] * 2
zipped = [s for t in zip(a+NONE2, b+NONE2) for s in t if s]
```

Go reproduces this exactly with a small helper:

```go
func zipInterleave(a, b []*Player) []*Player {
    aa := append(append([]*Player{}, a...), nil, nil)
    bb := append(append([]*Player{}, b...), nil, nil)
    n := min(len(aa), len(bb))
    var out []*Player
    for i := 0; i < n; i++ {
        if aa[i] != nil { out = append(out, aa[i]) }
        if bb[i] != nil { out = append(out, bb[i]) }
    }
    return out
}
```

---

## 6. `None` → pointers and companion booleans

| Python | Go |
|---|---|
| `self.owner = None / Player` | `owner *Player` |
| `game.kickoff_player = None` | `kickoffPlayer *Player` |
| `active_control_player` | `activeControlPlayer *Player` |
| `self.mark` (Player or Goal) | `mark Marker` (interface, nil-able) |
| `self.lead = None / number` | `lead float64` + `hasLead bool` |
| `Team.controls = None` (AI team) | `controls *Controls` nil, `human()` checks non-nil |

The `lead` field is the notable one: because it holds a number when set, "is it
assigned?" needs a separate `hasLead` boolean, and `if self.lead is not None`
becomes `if p.hasLead`.

---

## 7. `MyActor` → `Actor` with camera-offset drawing

Python's `MyActor` stores a world-space `vpos` and, on draw, converts to screen
space using the scroll offset. Go mirrors this, adding a Pygame-style anchor
(players/shadows are anchored at the feet `(25, 37)`):

```go
type Actor struct { vpos Vec2; image string; anchorCentre bool; ax, ay float64 }
func (a *Actor) Draw(as *Assets, offX, offY float64) {
    ox, oy := a.anchorOffset(as)
    as.Blit(a.image, a.vpos.X-offX-ox, a.vpos.Y-offY-oy)
}
```

The depth-sorted draw (ball + players by Y, then all shadows, with goals at each
end) is reproduced with a `drawItem{main, shadow}` list and `sort.SliceStable`.

---

## 8. Angles, integer maths, and Python semantics

- **Custom trig** for 8-way angles (`sin`/`cos` scaled by π/4, `vec_to_angle`,
  `angle_to_vec`) is ported verbatim.
- **Floor modulo / division** are needed for the facing-direction rotation and
  the animation-frame sprite index, since `anim_frame` can be `-1`:

  ```python
  self.dir = (self.dir + [0,1,1,1,1,7,7,7][dir_diff % 8]) % 8
  suffix = str(self.dir) + str((int(self.anim_frame) // 18) + 1)
  ```
  ```go
  p.dir = pmod(p.dir+rotTable[pmod(dirDiff, 8)], 8)
  suffix := strconv.Itoa(p.dir) + strconv.Itoa(floorDiv(int(p.animFrame), 18)+1)
  ```

  `main.go` supplies `pmod`/`floorDiv` to match Python's floor behaviour, since
  Go's `%` and `/` truncate toward zero.

---

## 9. Input, sound, state

- **`key_just_pressed`** uses a `key_status` dict in Python; Go keeps a per-frame
  keyboard snapshot (`keys`/`prevKeys`) and derives the rising edge from it.
  `Controls` still maps two key sets (arrows+Space, WASD+Shift) to `move()` and
  `shoot()`.
- **Two enums** (`State`, `MenuState`) → two `const … iota` blocks, driving the
  1P/2P and difficulty menus.
- **Sound**: the match audio is richer than the other games — a looping crowd
  ambience plus a start whistle when a human match begins, and title music on the
  menu. Python calls `sounds.crowd.play(-1)`, `music.fadeout(1)` directly; the Go
  `Audio` wraps these as `StartMatchAudio`/`StartMenuMusic` with looping tracks,
  and `play_sound` variant selection is preserved. Effects are muted on the menu.

---

## 10. What is intentionally identical

- Ball physics: per-axis bounce with goal-mouth openings, drag, dribble easing
  with elliptical X/Y offsets, off-pitch loss of possession, acquisition rules.
- Kicking: targetable-player/goal selection, cost comparison for CPU shots, the
  8-iteration lead refinement for human passes, and the "kick ahead + guess
  receiver" fallback.
- Player AI: human control, CPU cost-based five-direction choice, team-mate
  spreading, lead-player pursuit, marking (including the goalie), and ball
  interception by forward simulation.
- Team-level logic: per-frame mark/lead reset, goalie assignment, up/down-field
  lead selection, manual player switching with up-field weighting.
- Camera easing, kickoff setup, scoring, first-to-9 win, and all menu/HUD sprites.

---

## 11. Summary of differences

| Category | Difference | Reason |
|---|---|---|
| Vectors | `Vector2` + operators (incl. `*` = dot) → `Vec2` methods | no operator overloading |
| Duck typing | Player/Goal used interchangeably → `posTeam`/`Marker` interfaces | static typing |
| `isinstance` | → type assertions `x.(*Player)` / `x.(*Goal)` | static typing |
| `min`/`sorted` `key=` | closure factories → explicit min loops / `sort.Slice` | no `key=` param |
| `cost` tuple | `(result, pos)` min workaround → plain `float64` | Go compares floats fine |
| `zip(+NONE2)` | None-padded interleave → `zipInterleave` helper | no None/`zip` |
| Optionals | `None` → pointers; `lead` → `float64 + hasLead` | no `None` |
| Integer maths | `//`, `%` → `floorDiv`, `pmod` | Go truncates toward zero |
| Base actor | `MyActor` → `Actor` + camera-offset draw | no classes |
| Framework | Pygame Zero → pgzgo (over go-sdl3) | library swap |

The AI, physics, and match rules are line-by-line equivalent to `soccer.py`. The
defining translation is the pair of Go interfaces (`posTeam`, `Marker`) that let
a Player and a Goal be treated uniformly — the static-typing counterpart to
Python's duck typing — together with the `Vec2` method set replacing pygame's
overloaded vector operators.
