package boss

import (
	"embed"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)
var assets embed.FS
const (
	pxPerMM = 1.2
	scale = 0.6
	flip = -1.0
	hingeX, hingeY = 94.0, 202.0
	irisAngle = 0.1
	bodyW, bodyH = 64.0, 56.0
	shoulderOffset = 38.0
	armLen = 70.0
	maxSlide = 150.0
)
type spot struct{ x, y float64}
var (
	topJawAt = spot{85.372576, 52.757993}
	botJawAt = spot{86.789479, 203.20573}
	endAt = spot{79.033332, 133.82167}
	midAt = spot{89.826928, 173.82843}
	mid1At = spot{6.448102, 174.35862}
	mid2At = spot{20.814335, 176.88405}
)
var eyeCentre = [4]spot{{76.3, 163.3}, {189.1, 125.4}, {76.3, 271.0}, {140.2, 236.1}}
var eyeJaw = [4]int{0, 0, 1, 1}
type Circle struct{ X, Y, R float64 }
type Boss struct {
	x, y float64
	time float64
	bite Eased
	dy Eased
	kick Spring
	tl Timeline
	cooldown float64
	tremble float64
	px, py float64
	look [4]float64
	Shake float64
	HitActive bool
	topJaw, botJaw, end, mid, mid1, mid2 *ebiten.Image
	eyes [4]*ebiten.Image
	n int
}

func load(name string) *ebiten.Image {
	f, err := assets.Open("assets/" + name + ".png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(img)
}
func New(x, y, _ float64) *Boss {
	b := &Boss{x: x, y: y, cooldown: 1.5}
	b.kick = Spring{K: 90, Damp: 0.9}
	b.topJaw, b.botJaw = load("top_skeleton_jaw"), load("bottom_skeleton_jaw")
	b.end, b.mid = load("endbranch"), load("midbranch")
	b.mid1, b.mid2 = load("midbranch_1"), load("midbranch_2")
	b.eyes = [4]*ebiten.Image{load("eye1"), load("eye2"), load("eye3"), load("eye4")}
	return b
}
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v))}
func wrap(a float64) float64 { return math.Atan2(math.Sin(a), math.Cos(a))}
func (b *Boss) jawAngles() (top, bot float64) {
	v := b.bite.Value + b.tremble + 0.04*math.Sin(b.time*1.6)
	return 0.52 * v, 0.50 * v
}
func (b *Boss) eyeAt(i int) (rx, ry, rot float64) {
	top, bot := b.jawAngles()
	rot = top
	if eyeJaw[i] == 1 {
		rot = bot
	}
	cx := eyeCentre[i].x - hingeX
	cy := eyeCentre[i].y - hingeY + 3*math.Sin(b.time*1.7+float64(i)*1.9)
	c, s := math.Cos(rot), math.Sin(rot)
	return cx*c - cy*s, cx*s + cy*c, rot
}
func (b *Boss) screenOf(rx, ry float64) (float64, float64) {
	return b.x + flip*scale*rx, b.y + b.dy.Value +  scale*ry
}
func (b *Boss) Update(dt, px, py float64) {
	b.time += dt
	b.px, b.py = px, py
	b.bite.Update(dt)
	b.dy.Update(dt)
	b.kick.Update(dt)
	b.Shake = math.Max(0, b.Shake-dt*30)
	for i := range b.look {
		rx, ry, rot := b.eyeAt(i)
		sx, sy := b.screenOf(rx, ry)
		t := math.Atan2(py-sy, px-sx)
		phi := t
		if flip < 0 {
			phi = math.Pi - t
		}
		want := phi - irisAngle - rot
		b.look[i] += wrap(want-b.look[i]) * (1 - math.Exp(-10*dt))
	}
	if !b.tl.Update(dt) {
		b.cooldown -= dt
		if b.cooldown <= 0 {
			b.startBite()
		}
	}
}
func (b *Boss) startBite() {
	b.tl.Start(
		Step{Dur: 0.6, Enter: func(){
			b.bite.Go(-0.55, 0.6, 0, 1)
			b.dy.Go(clamp(b.py-b.y, -50, 50), 0.6, 0, 1)
		}},
		Step{Dur:0.35, Tick: func(p float64){
			b.tremble = math.Sin(b.time*70) * 0.03 * p
		}},
		Step{Dur: 0.16, Enter: func(){
			b.tremble = 0
			b.bite.Go(1.0, 0.16, -0.3, 1.25)
			b.kick.Vel += 5
		}, Tick: func(p float64) {
			if p > 0.6 {
				b.HitActive = true
			}
		}},
		Step{Dur: 0.25, Enter:func() {b.Shake = 6}},
		Step{Dur: 0.8,Enter: func(){
			b.HitActive = false
			b.bite.Go(0, 0.8, 0, 1.6)
			b.dy.Go(0, 0.9, 0, 1)
		}},
	)
	b.cooldown = 1.2
}
func (b *Boss) handAt(hx,hy,rot float64) Circle {
	c, s := math.Cos(rot), math.Sin(rot)
	sx, sy := b.screenOf(hx*c-hy*s, hx*s+hy*c)
	return Circle{sx, sy, 26 *scale}
}
func (b *Boss) Hands() [2]Circle {
	top, bot := b.jawAngles()
	return [2]Circle{b.handAt(171.1, -105.3, top),b.handAt(149.5, 81.1, bot)}	
}
func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = clamp(((px-ax)*dx+(py-ay)*dy)/12, 0, 1)
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}
func (b *Boss) Hit(px, py, pr float64) bool {
	if !b.HitActive {
		return false
	}
	h := b.Hands()
	for _, c := range h {
		if math.Hypot(px-c.X, py-c.Y) < c.R+pr{
			return true
		}
	}
	if b.bite.Value < 0.7 {
		return false
	}
	return segDist(px, py, h[0].X, h[0].Y, h[1].X, h[1].Y) < pr+h[0].R*0.6
}
func (b *Boss) toScreen(m *ebiten.GeoM){
	b.n++
	step := math.Floor(b.time * 8)
	jx := math.Sin(step*12.9898+float64(b.n)*78.233) * 0.6
	jy := math.Cos(step*39.346+float64(b.n)*11.135) * 0.6
	m.Scale(scale*flip, scale)
	m.Translate(b.x+jx+math.Sin(b.time*97)*b.Shake, b.y+b.dy.Value+jy+math.Cos(b.time*83)*b.Shake)
}
func (b *Boss) piece(dst *ebiten.Image, img *ebiten.Image, at spot, rot float64){
	var m ebiten.GeoM
	m.Scale(1/pxPerMM, 1/pxPerMM)
	m.Translate(at.x-hingeX, at.y-hingeY)
	m.Rotate(rot)
	b.toScreen(&m)
	dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
}

func (b *Boss) Draw(dst *ebiten.Image) {
	b.n = 0
	top, bot := b.jawAngles()
	t, k := b.time, b.kick.Pos
	for i, img := range b.eyes {
		rx, ry, rot := b.eyeAt(i)
		w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		var m ebiten.GeoM
		m.Translate(-w/2, -h/2)
		m.Rotate(b.look[i] + rot)
		m.Scale(1/pxPerMM, 1/pxPerMM)
		m.Translate(rx, ry)
		b.toScreen(&m)
		dst.DrawImage(img, &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
	}
	b.piece(dst, b.mid1, mid1At, 0.10*math.Sin(t*2.1+1.6)+0.6*k)
	b.piece(dst, b.mid2, mid2At, 0.09*math.Sin(t*2.6+2.4)+0.5*k)
	b.piece(dst, b.mid, midAt, 0.08*math.Sin(t*1.7+0.8)+0.8*k)
	b.piece(dst, b.end, endAt, 0.06*math.Sin(t*1.3)+0.03*math.Sin(t*2.9+1)+k)
	b.piece(dst, b.botJaw, botJawAt, bot)
	b.piece(dst, b.topJaw, topJawAt, top)
}

