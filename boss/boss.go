package boss

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)
const (
	bodyW, bodyH = 64.0, 56.0
	shoulderOffset = 38.0
	armLen = 70.0
	maxSlide = 150.0
)
var (
	cream = color.RGBA{236, 226, 196, 255}
	ink = color.RGBA{18, 14, 12, 255}
	red = color.RGBA{200, 50, 40, 255}
)

type Boss struct {
	homeX, homeY, groundY float64
	time, bob float64
	armRot [2]Eased
	armExt [2]Eased
	squash Eased
	bodyDX Eased
	headTilt Spring
	tail *Chain
	t1 Timeline
	cooldown float64
	slamCount int
	tremble float64
	trembleArm int
	prevDX float64
	px, py float64
	Shake float64
	HitActive bool
	white *ebiten.Image
	n int
}
func New(x, y, groundY float64) *Boss {
	b := &Boss{homeX: x, homeY: y, groundY: groundY, cooldown: 1.5}
	for i := range b. armExt {
		b.armExt[i].Set(1)
	}
	b.armRot[0].Set(-0.25)
	b.armRot[1].Set(0.25)
	b.squash.Set(1)
	b.headTilt = Spring{K: 220, Damp: 0.9}
	b.tail = NewChain(8,13, x, y+22)
	b.white = ebiten.NewImage(1, 1)
	b.white.Fill(color.White)
	return b
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v))}
func (b *Boss) Update(dt, px, py float64) {
	b.time += dt
	b.px, b.py = px, py
	b.bob = math.Sin(b.time*2.2) * 4
	for i := range b.armRot {
		b.armRot[i].Update(dt)
		b.armExt[i].Update(dt)
	}
	b.squash.Update(dt)
	b.bodyDX.Update(dt)
	vx := (b.bodyDX.Value - b.prevDX) / dt
	b.prevDX = b.bodyDX.Value
	b.headTilt.Target = clamp(vx*0.004, -0.5, 0.5)
	b.headTilt.Update(dt)
	b.Shake = math.Max(0,b.Shake-dt*30)
	if !b.t1.Update(dt){
		b.cooldown -= dt
		if b.cooldown <= 0 {
			b.startSlam()
		}
	}
	bx, by := b.bodyPos()
	b.tail.Update(dt, bx, by+22)
}
func (b *Boss) bodyPos() (float64, float64) {
	return b.homeX + b.bodyDX.Value, b.homeY + b.bob
}
func (b *Boss) armGeom(i int) (sx, sy, tx, ty, rot float64) {
	bx, by := b.bodyPos()
	side := float64(2*i - 1)
	sx, sy = bx+side*shoulderOffset, by-6
	rot = b.armRot[i].Value
	if b.trembleArm == i {
		rot += b.tremble
	}
	l := armLen * b.armExt[i].Value
	return sx, sy, sx + math.Sin(rot)*l, sy + math.Cos(rot)*1, rot
}
func (b *Boss) Hitbox() (x, y, r float64, active bool) {
	for i :=range b.armRot {
		_, _, tx, ty, _ := b.armGeom(i)
		if ty > b.homeY+armLen {
			return tx, ty, 13, b.HitActive
		}
	}
	return 0, 0, 0, false
}
func (b *Boss) startSlam() {
	side, i := 1.0, 1
	if b.slamCount%2 == 0 {
		side, i = -1, 0
	}
	b.slamCount++
	rest := side *0.25
	reach := (b.groundY - (b.homeY-6)) / armLen
	b.t1.Start(
		Step{Dur: 0.55, Enter: func() {
			b.trembleArm, b.tremble = i, 0
			b.armRot[i].Go(side*2.5, 0.55, 0, 1)
			b.squash.Go(0.85, 0.55, 0, 1)
			dx := clamp(b.px-side*shoulderOffset-b.homeX, -maxSlide, maxSlide)
			b.bodyDX.Go(dx, 0.55, 0, 1)
		}},
		Step{Dur: 0.14, Enter: func() {
			b.tremble = 0
			b.armRot[i].Go(0, 0.14, -0.35, 1)
			b.armExt[i].Go(reach, 0.14, 0, 1)
			b.squash.Go(1.18, 0.14, 0, 1)
		}, Tick: func(p float64) {
			if p > 0.7 {
				b.HitActive = true
			}
		}},
		Step{Dur: 0.2, Enter: func() {
			b.Shake = 6
			b.squash.Go(0.82, 0.08, 0, 1)
		}},
		Step{Dur: 0.7, Enter: func() {
			b.HitActive = false
			b.armRot[i].Go(rest, 0.7, 0, 1.7)
			b.armExt[i].Go(1, 0.5, 0, 1.4)
			b.squash.Go(1, 0.6, 0, 1.8)
			b.bodyDX.Go(0, 0.9, 0, 1)
		}}, 
	)
	b.cooldown = 1.0
}
func (b *Boss) part(dst *ebiten.Image, x, y, w, h, rot, pvx, pvy float64, c color.RGBA) {
	b.n++
	step := math.Floor(b.time * 8)
	ox := math.Sin(step*12.9898+float64(b.n)*78.233) * 0.8
	oy := math.Sin(step*39.346+float64(b.n)*11.135) * 0.8
	sx := math.Sin(b.time*97) * b.Shake
	sy := math.Cos(b.time*83) * b.Shake
	draw := func(w, h float64, c color.RGBA) {
		var m ebiten.GeoM
		m.Scale(w, h)
		m.Translate(-pvx*w, -pvy*h)
		m.Rotate(rot)
		m.Translate(x+ox+sx, y+oy+sy)
		op := &ebiten.DrawImageOptions{GeoM: m}
		op.ColorScale.ScaleWithColor(c)
		dst.DrawImage(b.white, op)
	}
	draw(w+4, h+4, ink)
	draw(w, h, c)
	
}
func (b *Boss) Draw(dst *ebiten.Image){
	b.n = 0 
	bx, by := b.bodyPos()
	for i, p := range b.tail.Pts {
		s := 12 - float64(i)*1.1
		sway := math.Sin(b.time*3+float64(i)*0.6) * float64(i) * 0.8
		b.part(dst, p.X+sway, p.Y, s, s, 0, 0.5, 0.5, cream)
	}
	sy := b.squash.Value
	b.part(dst, bx, by, bodyW/sy, bodyH*sy, 0, 0.5, 0.5, cream)
	for i := range b.armRot {
		sx, shy, tx, ty, rot := b.armGeom(i)
		l := armLen * b.armExt[i].Value
		b.part(dst, sx, shy, 12, l, -rot, 0.5, 0, cream)
		b.part(dst, tx, ty, 22, 22, 0, 0.5, 0.5, red)
	}
	hy := by - bodyH*sy/2
	b.part(dst, bx, hy, 36, 30, b.headTilt.Pos, 0.5, 1, cream)
	dx, dy := b.px-bx, b.py-hy
	d := math.Max(1, math.Hypot(dx, dy))
	lx, ly := dx/d*3, dy/d*3
	for _, e := range []float64{-8, 8} {
		b.part(dst, bx+e+lx, hy-16+ly, 5, 8, 0, 0.5, 0.5, ink)
	}
}