package main

// Actor is the base for on-screen objects, replacing the game's MyActor. Position
// is stored in world space as vpos; drawing subtracts the camera offset. The
// anchor is the image centre unless an explicit pixel anchor is given.
type Actor struct {
	vpos         Vec2
	image        string
	anchorCentre bool
	ax, ay       float64
}

func newActor(image string, x, y float64) Actor {
	return Actor{vpos: Vec2{x, y}, image: image, anchorCentre: true}
}

func newAnchoredActor(image string, x, y, ax, ay float64) Actor {
	return Actor{vpos: Vec2{x, y}, image: image, ax: ax, ay: ay}
}

func (a *Actor) anchorOffset(as *Assets) (float64, float64) {
	if a.anchorCentre {
		w, h := as.Size(a.image)
		return w / 2, h / 2
	}
	return a.ax, a.ay
}

// Draw blits the actor at its world position minus the camera offset.
func (a *Actor) Draw(as *Assets, offX, offY float64) {
	ox, oy := a.anchorOffset(as)
	as.Blit(a.image, a.vpos.X-offX-ox, a.vpos.Y-offY-oy)
}

// PosY is the actor's world Y, used to order drawing back-to-front.
func (a *Actor) PosY() float64 { return a.vpos.Y }

// Drawable is anything the renderer can depth-sort and draw with an offset.
type Drawable interface {
	Draw(as *Assets, offX, offY float64)
	PosY() float64
}
