package main

import "math"

// Vec2 is a 2D vector, replacing pygame's Vector2.
type Vec2 struct {
	X, Y float64
}

func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Length() float64    { return math.Hypot(a.X, a.Y) }

// safeNormalise returns the unit vector and original length, guarding against a
// zero-length vector (which cannot be normalised).
func safeNormalise(v Vec2) (Vec2, float64) {
	length := v.Length()
	if length == 0 {
		return Vec2{0, 0}, 0
	}
	return Vec2{v.X / length, v.Y / length}, length
}
