package boss

import (
	"embed"
	"image/color"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*.png
var assets embed.FS

const (
	arenaW, arenaH = 560.0, 240.0

	pxPerMM     = 3.6
	branchScale = 1.8
	eyeScale    = 2.1
	branchLen   = 327.0
	baseX       = 8.0
	baseY       = 178.0
	axisFix     = 0.116
	irisAngle   = 0.1

	only = -1
)

type spot struct{ x, y float64 }

type vinePiece struct {
	name       string
	at         spot
	amp, freq  float64
	phase, kck float64
}

var vine = [4]vinePiece{
	{"midbranch_1", spot{6.448102, 174.35862}, 0.05, 2.1, 1.6, 0.3},
	{"midbranch_2", spot{20.814335, 176.88405}, 0.05, 2.6, 2.4, 0.4},
	{"midbranch", spot{89.826928, 173.82843}, 0.06, 1.7, 0.8, 0.7},
	{"endbranch", spot{79.033332, 133.82167}, 0.05, 1.3, 0.0, 1.0},
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

func pixel(c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(c)
	return img
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func wrap(a float64) float64         { return math.Atan2(math.Sin(a), math.Cos(a)) }

func segDist(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = clamp(((px-ax)*dx+(py-ay)*dy)/l2, 0, 1)
	}
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}

type Boss struct {
	px, py, dt, time float64
	n                int
	Shake            float64

	branches [2]*Branch
	eye      *Eye

	cool float64
	next int

	warnDim, warnHot, beamOut, beamCore *ebiten.Image
}

func New(_, _, _ float64) *Boss {
	b := &Boss{cool: 1.5}
	for i := range b.branches {
		br := &Branch{seed: float64(i) * 2.3}
		br.kick = Spring{K: 90, Damp: 0.9}
		for j, p := range vine {
			br.img[j] = load(p.name)
		}
		b.branches[i] = br
	}
	b.eye = &Eye{img: load("eye2")}
	b.warnDim = pixel(color.RGBA{70, 12, 18, 70})
	b.warnHot = pixel(color.RGBA{255, 70, 80, 255})
	b.beamOut = pixel(color.RGBA{255, 40, 60, 255})
	b.beamCore = pixel(color.RGBA{255, 240, 240, 255})
	return b
}

func (b *Boss) jit() (float64, float64) {
	b.n++
	step := math.Floor(b.time * 8)
	return math.Sin(step*12.9898+float64(b.n)*78.233) * 1.4,
		math.Cos(step*39.346+float64(b.n)*11.135) * 1.4
}

func (b *Boss) busy() bool {
	return b.branches[0].running || b.branches[1].running || b.eye.running
}

func (b *Boss) Update(dt, px, py float64) {
	b.dt, b.px, b.py = dt, px, py
	b.time += dt
	b.Shake = math.Max(0, b.Shake-dt*30)

	for _, br := range b.branches {
		br.update(b, dt)
	}
	b.eye.update(b, dt)

	if b.busy() {
		b.cool = 1.3
		return
	}
	b.cool -= dt
	if b.cool <= 0 {
		n := b.next
		if only >= 0 {
			n = only
		}
		b.launch(n)
		b.next = (b.next + 1) % 4
	}
}

func (b *Boss) farSide() float64 {
	if b.px < arenaW/2 {
		return 1
	}
	return -1
}

func (b *Boss) launch(n int) {
	side := b.farSide()
	switch n {
	case 0:
		b.branches[0].strike(b, sideStrike(side, true, 0))
	case 1:
		b.eye.fire(b, side, 0)
	case 2:
		cy := clamp(b.py, 90, 150)
		b.branches[0].strike(b, crossStrike(1, cy, 0, 0.7))
		b.branches[1].strike(b, crossStrike(-1, cy, 0.15, 0.95))
	case 3:
		b.eye.fire(b, -side, 0)
		b.branches[0].strike(b, sideStrike(side, true, 0.5))
	}
}

func (b *Boss) Hit(px, py, pr float64) bool {
	for _, br := range b.branches {
		if br.hit(px, py, pr) {
			return true
		}
	}
	return b.eye.hit(px, py, pr)
}

func (b *Boss) Draw(dst *ebiten.Image) {
	b.n = 0
	sx := math.Sin(b.time*97) * b.Shake
	sy := math.Cos(b.time*83) * b.Shake
	b.eye.drawBeam(dst, b, sx, sy)
	b.eye.draw(dst, b, sx, sy)
	for _, br := range b.branches {
		br.draw(dst, b, sx, sy)
	}
}

func (b *Boss) line(dst, pix *ebiten.Image, x, y, ang, length, w float64) {
	var m ebiten.GeoM
	m.Scale(length, w)
	m.Translate(0, -w/2)
	m.Rotate(ang)
	m.Translate(x, y)
	dst.DrawImage(pix, &ebiten.DrawImageOptions{GeoM: m})
}

type Branch struct {
	img     [4]*ebiten.Image
	running bool
	live    bool
	danger  bool
	warn    bool

	rootX, rootY, angle float64
	ext, tilt           Eased
	kick                Spring
	tl                  Timeline
	delay, t, seed      float64
}

type strikeSpec struct {
	rootX, rootY, angle        float64
	track                      bool
	delay, enter, trackT       float64
	windup, thrust, retractDur float64
}

func sideStrike(side float64, track bool, delay float64) strikeSpec {
	s := strikeSpec{rootY: arenaH / 2, track: track, delay: delay,
		enter: 0.5, trackT: 1.4, windup: 0.55, thrust: 0.38, retractDur: 0.8}
	if side > 0 {
		s.rootX, s.angle = arenaW+20, math.Pi
	} else {
		s.rootX, s.angle = -20, 0
	}
	return s
}

func crossStrike(side, cy, delay, retract float64) strikeSpec {
	s := strikeSpec{delay: delay, enter: 0.5, trackT: 0.15,
		windup: 0.65, thrust: 0.42, retractDur: retract}
	target := spot{arenaW / 2, cy}
	if side > 0 {
		s.rootX, s.rootY = -20, cy-80
	} else {
		s.rootX, s.rootY = arenaW+20, cy+80
	}
	s.angle = math.Atan2(target.y-s.rootY, target.x-s.rootX)
	return s
}

func (br *Branch) strike(b *Boss, s strikeSpec) {
	br.rootX, br.rootY, br.angle = s.rootX, s.rootY, s.angle
	br.ext.Set(0)
	br.tilt.Set(0)
	br.delay, br.running, br.live, br.danger, br.warn = s.delay, true, false, false, false
	dir := 1.0
	if math.Cos(s.angle) < 0 {
		dir = -1
	}
	br.tl.Start(
		Step{Dur: s.enter, Enter: func() {
			br.live = true
			if s.track {
				br.rootY = clamp(b.py, 30, arenaH-30)
			}
			br.ext.Go(0.25, s.enter, 0.1, 1)
		}},
		Step{Dur: s.trackT, Tick: func(p float64) {
			if s.track {
				target := clamp(b.py, 30, arenaH-30)
				br.rootY += (target - br.rootY) * (1 - math.Exp(-3.5*b.dt))
			}
		}},
		Step{Dur: s.windup, Enter: func() {
			br.warn = true
			br.ext.Go(0.17, s.windup, 0.3, 1)
		}, Tick: func(p float64) {
			br.tilt.Set(math.Sin(br.t*70) * 0.015 * p)
		}},
		Step{Dur: s.thrust, Enter: func() {
			br.warn = false
			br.tilt.Set(-0.09 * dir)
			br.tilt.Go(0.09*dir, s.thrust, 0, 1)
			br.ext.Go(1.02, s.thrust, -0.25, 1.1)
		}, Tick: func(p float64) {
			br.danger = p > 0.05
		}},
		Step{Dur: 0.25, Enter: func() {
			b.Shake = 6
			br.kick.Vel += 8
		}},
		Step{Dur: s.retractDur, Enter: func() {
			br.danger = false
			br.ext.Go(0, s.retractDur, 0.35, 1)
			br.tilt.Go(0, s.retractDur, 0, 1)
		}},
		Step{Dur: 0.02, Enter: func() { br.live = false }},
	)
}

func (br *Branch) update(b *Boss, dt float64) {
	br.t += dt
	br.ext.Update(dt)
	br.tilt.Update(dt)
	br.kick.Update(dt)
	if !br.running {
		return
	}
	if br.delay > 0 {
		br.delay -= dt
		return
	}
	if !br.tl.Update(dt) {
		br.running, br.live, br.danger = false, false, false
	}
}

func (br *Branch) geometry() (bx, by, dx, dy, a, length float64) {
	a = br.angle + br.tilt.Value
	dx, dy = math.Cos(a), math.Sin(a)
	length = branchLen * branchScale
	back := (1 - br.ext.Value) * (length + 40)
	return br.rootX - dx*back, br.rootY - dy*back, dx, dy, a, length
}

func (br *Branch) hit(px, py, pr float64) bool {
	if !br.live || !br.danger {
		return false
	}
	bx, by, dx, dy, _, length := br.geometry()
	for i := 0; i <= 8; i++ {
		t := float64(i) / 8
		r := (11 - 6*t) * branchScale
		cx, cy := bx+dx*length*t, by+dy*length*t
		if math.Hypot(px-cx, py-cy) < r+pr {
			return true
		}
	}
	return false
}

func (br *Branch) draw(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !br.live {
		return
	}
	bx, by, dx, dy, a, length := br.geometry()
	bx, by = bx+sx, by+sy

	if br.warn {
		b.line(dst, b.warnDim, bx+dx*length, by+dy*length, a+math.Pi, 0, 0)
		reach := math.Hypot(arenaW, arenaH)
		b.line(dst, b.warnDim, bx+dx*length*0.5, by+dy*length*0.5, a, reach, 5)
	}

	mirror := math.Cos(br.angle) < 0
	for i, p := range vine {
		sway := p.amp*math.Sin(br.t*p.freq+p.phase+br.seed) + p.kck*br.kick.Pos*0.25
		var m ebiten.GeoM
		m.Scale(1/pxPerMM, 1/pxPerMM)
		m.Translate(p.at.x-baseX, p.at.y-baseY)
		m.Rotate(sway)
		m.Rotate(axisFix)
		if mirror {
			m.Scale(1, -1)
		}
		m.Scale(branchScale, branchScale)
		m.Rotate(a)
		jx, jy := b.jit()
		m.Translate(bx+jx, by+jy)
		dst.DrawImage(br.img[i], &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
	}
}

type Eye struct {
	img     *ebiten.Image
	running bool
	live    bool
	danger  bool

	homeX, homeY, dirX float64
	ent                Eased
	aim, look          float64
	mode               int
	width, pulse, t    float64
	tl                 Timeline
	delay              float64
}

func (e *Eye) center() (float64, float64) {
	w := float64(e.img.Bounds().Dx()) / pxPerMM * eyeScale
	return e.homeX + e.dirX*(1-e.ent.Value)*(w+60), e.homeY
}

func (e *Eye) fire(b *Boss, side float64, delay float64) {
	e.dirX = side
	if side > 0 {
		e.homeX = arenaW - 12
	} else {
		e.homeX = 12
	}
	if b.py < arenaH/2 {
		e.homeY = arenaH * 0.72
	} else {
		e.homeY = arenaH * 0.28
	}
	e.ent.Set(0)
	e.mode, e.width, e.pulse = 0, 0, 0
	e.delay, e.running, e.live, e.danger = delay, true, false, false
	e.tl.Start(
		Step{Dur: 0.7, Enter: func() {
			e.live = true
			x, y := e.center()
			e.aim = math.Atan2(b.py-y, b.px-x)
			e.look = e.aim
			e.ent.Go(1, 0.7, 0.1, 1)
		}},
		Step{Dur: 1.3, Enter: func() { e.mode = 1 }, Tick: func(p float64) {
			x, y := e.center()
			want := math.Atan2(b.py-y, b.px-x)
			e.aim += wrap(want-e.aim) * (1 - math.Exp(-4*b.dt))
			e.look = e.aim
		}},
		Step{Dur: 0.7, Enter: func() { e.mode = 2 }, Tick: func(p float64) {
			e.pulse = math.Sin(e.t*45) * 0.03 * p
		}},
		Step{Dur: 0.55, Enter: func() {
			e.mode, e.pulse = 3, 0
			b.Shake = 4
		}, Tick: func(p float64) {
			env := math.Min(p/0.12, 1)
			if p > 0.7 {
				env *= (1 - p) / 0.3
			}
			e.width = 34 * env
			e.danger = env > 0.4
		}},
		Step{Dur: 0.3, Enter: func() { e.mode, e.danger, e.width = 0, false, 0 }},
		Step{Dur: 0.5, Enter: func() { e.ent.Go(0, 0.5, 0.4, 1) }},
		Step{Dur: 0.02, Enter: func() { e.live = false }},
	)
}

func (e *Eye) update(b *Boss, dt float64) {
	e.t += dt
	e.ent.Update(dt)
	if !e.running {
		return
	}
	if e.delay > 0 {
		e.delay -= dt
		return
	}
	if !e.tl.Update(dt) {
		e.running, e.live, e.danger = false, false, false
	}
}

func (e *Eye) hit(px, py, pr float64) bool {
	if !e.live || !e.danger {
		return false
	}
	x, y := e.center()
	ex, ey := x+math.Cos(e.aim)*1200, y+math.Sin(e.aim)*1200
	return segDist(px, py, x, y, ex, ey) < e.width*0.42+pr
}

func (e *Eye) drawBeam(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !e.live || e.mode == 0 {
		return
	}
	x, y := e.center()
	x, y = x+sx, y+sy
	switch e.mode {
	case 1:
		b.line(dst, b.warnDim, x, y, e.aim, 1200, 2)
	case 2:
		pix := b.warnDim
		if int(e.t*14)%2 == 0 {
			pix = b.warnHot
		}
		b.line(dst, pix, x, y, e.aim, 1200, 3)
	case 3:
		_, jy := b.jit()
		w := e.width + jy
		b.line(dst, b.beamOut, x, y, e.aim, 1200, w)
		b.line(dst, b.beamCore, x, y, e.aim, 1200, w*0.4)
	}
}

func (e *Eye) draw(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !e.live {
		return
	}
	x, y := e.center()
	y += 3 * math.Sin(e.t*1.7)
	jx, jy := b.jit()
	w, h := float64(e.img.Bounds().Dx()), float64(e.img.Bounds().Dy())
	var m ebiten.GeoM
	m.Translate(-w/2, -h/2)
	m.Rotate(e.look - irisAngle)
	s := eyeScale / pxPerMM * (1 + e.pulse)
	m.Scale(s, s)
	m.Translate(x+jx+sx, y+jy+sy)
	dst.DrawImage(e.img, &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
}