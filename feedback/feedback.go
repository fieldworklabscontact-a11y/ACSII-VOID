package feedback

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type shard struct {
	x, y, vx, vy, size, life, max float64
	red                           bool
}

type ring struct {
	x, y, t, delay float64
	red            bool
}

type Feedback struct {
	Max, HP int

	inv, freeze, shake, flash, slow, hudShake float64
	dead, died, respawned                     bool
	deadT                                     float64
	pop                                       [16]float64

	shards []shard
	rings  []ring
	seed   uint32

	Edge color.RGBA

	black, red, white *ebiten.Image
	redA, blackA      [10]*ebiten.Image
	buf               *ebiten.Image
}

func New(maxHP int) *Feedback {
	if maxHP > 16 {
		maxHP = 16
	}
	f := &Feedback{Max: maxHP, HP: maxHP, seed: 777, Edge: color.RGBA{255, 255, 255, 1}}
	px := func(c color.RGBA) *ebiten.Image {
		img := ebiten.NewImage(1, 1)
		img.Fill(c)
		return img
	}
	f.black = px(color.RGBA{18, 18, 22, 255})
	f.red = px(color.RGBA{255, 40, 60, 255})
	f.white = px(color.RGBA{255, 255, 255, 255})
	for i := range f.redA {
		a := float64(i) / 9
		f.redA[i] = px(color.RGBA{uint8(255 * a), uint8(40 * a), uint8(60 * a), uint8(255 * a)})
		f.blackA[i] = px(color.RGBA{0, 0, 0, uint8(255 * a)})
	}
	return f
}

func (f *Feedback) rnd() float64 {
	f.seed = f.seed*1664525 + 1013904223
	return float64(f.seed>>8) / float64(1<<24)
}

func (f *Feedback) burst(x, y float64, n int, speed float64) {
	for i := 0; i < n; i++ {
		a := f.rnd() * 2 * math.Pi
		v := speed * (0.35 + 0.65*f.rnd())
		l := 0.45 + 0.45*f.rnd()
		f.shards = append(f.shards, shard{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v,
			size: 2 + 4*f.rnd(), life: l, max: l, red: i%2 == 0})
	}
}

func (f *Feedback) Hurt(x, y float64) bool {
	if f.inv > 0 || f.dead || f.HP <= 0 {
		return false
	}
	f.HP--
	f.pop[f.HP] = 0.001
	f.freeze, f.slow, f.shake, f.flash, f.hudShake, f.inv = 0.09, 0.3, 1, 0.35, 0.5, 1.6
	f.burst(x, y, 18, 260)
	f.rings = append(f.rings, ring{x: x, y: y}, ring{x: x, y: y, delay: 0.07, red: true})
	if f.HP <= 0 {
		f.dead, f.died, f.deadT = true, true, 0
		f.freeze, f.shake, f.flash = 0.4, 1.6, 0.35
		f.burst(x, y, 44, 420)
		f.rings = append(f.rings, ring{x: x, y: y, delay: 0.16}, ring{x: x, y: y, delay: 0.26, red: true})
	}
	return true
}

func (f *Feedback) Frozen() bool       { return f.freeze > 0 }
func (f *Feedback) Dead() bool         { return f.dead }
func (f *Feedback) Invulnerable() bool { return f.inv > 0 || f.dead }

func (f *Feedback) PlayerVisible() bool {
	if f.dead {
		return false
	}
	return f.inv <= 0 || int(f.inv*16)%2 == 0
}

func (f *Feedback) TimeScale() float64 {
	if f.dead {
		return 0.2
	}
	if f.slow > 0 {
		return 0.35 + 0.65*(1-f.slow/0.3)
	}
	return 1
}

func (f *Feedback) Respawned() bool {
	r := f.respawned
	f.respawned = false
	return r
}

func (f *Feedback) Update(dt float64) {
	if f.freeze > 0 {
		f.freeze -= dt
		return
	}
	f.shake = math.Max(0, f.shake-dt*3)
	f.flash = math.Max(0, f.flash-dt)
	f.slow = math.Max(0, f.slow-dt)
	f.hudShake = math.Max(0, f.hudShake-dt)
	f.inv = math.Max(0, f.inv-dt)
	for i := range f.pop {
		if f.pop[i] > 0 {
			f.pop[i] += dt
			if f.pop[i] > 0.4 {
				f.pop[i] = 0
			}
		}
	}
	live := f.shards[:0]
	for _, s := range f.shards {
		s.life -= dt
		if s.life <= 0 {
			continue
		}
		d := math.Exp(-3.2 * dt)
		s.vx *= d
		s.vy *= d
		s.x += s.vx * dt
		s.y += s.vy * dt
		live = append(live, s)
	}
	f.shards = live
	rl := f.rings[:0]
	for _, r := range f.rings {
		if r.delay > 0 {
			r.delay -= dt
			rl = append(rl, r)
			continue
		}
		r.t += dt
		if r.t < 0.5 {
			rl = append(rl, r)
		}
	}
	f.rings = rl
	if f.dead {
		f.deadT += dt
		if f.deadT > 1.6 {
			f.dead = false
			f.HP = f.Max
			f.inv = 2
			f.respawned = true
		}
	}
}
func (f *Feedback) rect(dst, pix *ebiten.Image, x0, y0, x1, y1 float64) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	var m ebiten.GeoM
	m.Scale(x1-x0, y1-y0)
	m.Translate(x0, y0)
	dst.DrawImage(pix, &ebiten.DrawImageOptions{GeoM: m})
}
func (f *Feedback) frame(dst, pix *ebiten.Image, cx, cy, half, th, rot float64) {
	for i := 0; i < 4; i++ {
		var m ebiten.GeoM
		m.Scale(2*half, th)
		m.Translate(-half, -half)
		m.Rotate(rot + float64(i)*math.Pi/2)
		m.Translate(cx, cy)
		dst.DrawImage(pix, &ebiten.DrawImageOptions{GeoM: m})
	}
}
func (f *Feedback) Draw(dst *ebiten.Image) {
	for _, r := range f.rings {
		if r.delay > 0 {
			continue
		}
		p := r.t / 0.5
		e := 1 - (1-p)*(1-p)*(1-p)
		half := 6 + 70*e
		th := 1 + 3*(1-p)
		pix := f.black
		if r.red {
			pix = f.red
		}
		f.frame(dst, pix, r.x, r.y, half, th, 0)
		f.frame(dst, pix, r.x, r.y, half, th, math.Pi/4)
	}
	for _, s := range f.shards {
		pix := f.black
		if s.red {
			pix = f.red
		}
		k := s.size * math.Min(1, s.life/s.max*2)
		f.rect(dst, pix, s.x-k/2, s.y-k/2, s.x+k/2, s.y+k/2)
	}
	ox, oy := 0.0, 0.0
	if f.hudShake > 0 {
		ox, oy = (f.rnd()-0.5)*8*f.hudShake, (f.rnd()-0.5)*8*f.hudShake
	}
	for i := 0; i < f.Max; i++ {
		x, y := 10+float64(i)*16+ox, 10+oy
		if i < f.HP {
			f.rect(dst, f.red, x, y, x+10, y+10)
			f.frame(dst, f.black, x+5, y+5, 5, 1.5, 0)
		} else {
			f.frame(dst, f.black, x+5, y+5, 5, 1, 0)
		}
		if p := f.pop[i]; p > 0 {
			t := p / 0.4
			f.frame(dst, f.red, x+5, y+5, 5+18*t, 3*(1-t)+0.5, 0)
		}
	}
}
func (f *Feedback) Post(dst *ebiten.Image) {
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	if f.shake > 0.02 {
		if f.buf == nil || f.buf.Bounds().Dx() != w || f.buf.Bounds().Dy() != h {
			f.buf = ebiten.NewImage(w, h)
		}
		f.buf.Clear()
		f.buf.DrawImage(dst, nil)
		dst.Fill(f.Edge)
		var m ebiten.GeoM
		a := 10 * f.shake
		m.Translate(math.Round((f.rnd()-0.5)*2*a), math.Round((f.rnd()-0.5)*2*a))
		dst.DrawImage(f.buf.SubImage(image.Rect(0, 0, w, h)).(*ebiten.Image), &ebiten.DrawImageOptions{GeoM: m})
	}
	if f.flash > 0 {
		var pix *ebiten.Image
		if f.flash > 0.3 {
			pix = f.blackA[8]
		} else {
			pix = f.redA[int(5*f.flash/0.3+0.5)]
		}
		f.rect(dst, pix, 0, 0, float64(w), float64(h))
	}
}
