package boss

import (
	"embed"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*.png
var assets embed.FS

const (
	arenaW, arenaH = 560.0, 240.0

	pxPerMM   = 3.6
	branchLen = 341.0
	baseX     = 8.5
	baseY     = 224.3
	axisFix   = 0.2413
	irisAngle = 0.1

	numAttacks = 7
	only       = -1
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
	seed             uint32
	hist             [90][2]float64
	frame            int

	Shake float64
	tear  float64
	flash float64
	dark  Eased

	vstate int
	vt     float64

	branches [4]*Branch
	eyes     [4]*Eye
	amb      [2]*Branch
	watcher  *Eye

	cool, nextGlitch float64
	next             int
	wasBusy          bool

	warnDim, warnHot, beamOut, beamCore, flashPix *ebiten.Image
	black                                         [10]*ebiten.Image
	buf                                           *ebiten.Image
}

func New(_, _, _ float64) *Boss {
	b := &Boss{cool: 1.5, seed: 12345, nextGlitch: 4}
	imgs := [4]*ebiten.Image{}
	for j, p := range vine {
		imgs[j] = load(p.name)
	}
	newBranch := func(i int) *Branch {
		br := &Branch{img: imgs, seed: float64(i) * 2.3, scale: 3}
		br.kick = Spring{K: 90, Damp: 0.9}
		return br
	}
	for i := range b.branches {
		b.branches[i] = newBranch(i)
	}
	for i := range b.amb {
		br := newBranch(i + 7)
		br.ambient, br.live = true, true
		br.scale, br.reach = 3.2, 600
		b.amb[i] = br
	}
	b.amb[0].rootX, b.amb[0].rootY, b.amb[0].angle = -30, 10, 0.62
	b.amb[1].rootX, b.amb[1].rootY, b.amb[1].angle = arenaW+30, arenaH+10, math.Pi+0.5

	e2, e3, e1, e4 := load("eye2"), load("eye3"), load("eye1"), load("eye4")
	b.eyes = [4]*Eye{newEye(e2, 2.4), newEye(e3, 2.4), newEye(e1, 2.9), newEye(e4, 2.9)}
	b.watcher = newEye(e3, 6.2)
	b.watcher.live = true
	b.watcher.homeX, b.watcher.homeY, b.watcher.outY = arenaW*0.3, -30, -1
	b.watcher.ent.Set(1)
	b.watcher.lag = 0.4

	b.warnDim = pixel(color.RGBA{110, 18, 28, 110})
	b.warnHot = pixel(color.RGBA{255, 70, 80, 255})
	b.beamOut = pixel(color.RGBA{255, 40, 60, 255})
	b.beamCore = pixel(color.RGBA{255, 240, 240, 255})
	b.flashPix = pixel(color.RGBA{90, 20, 25, 90})
	for i := range b.black {
		b.black[i] = pixel(color.RGBA{0, 0, 0, uint8(255 * i / 9)})
	}
	return b
}

func (b *Boss) rnd() float64 {
	b.seed = b.seed*1664525 + 1013904223
	return float64(b.seed>>8) / float64(1<<24)
}

func (b *Boss) rr(lo, hi float64) float64 { return lo + (hi-lo)*b.rnd() }

func (b *Boss) past(sec float64) (float64, float64) {
	k := int(sec * 60)
	if k > len(b.hist)-1 {
		k = len(b.hist) - 1
	}
	i := ((b.frame-k)%len(b.hist) + len(b.hist)) % len(b.hist)
	return b.hist[i][0], b.hist[i][1]
}

func (b *Boss) jit() (float64, float64) {
	b.n++
	step := math.Floor(b.time * 8)
	return math.Sin(step*12.9898+float64(b.n)*78.233) * 1.4,
		math.Cos(step*39.346+float64(b.n)*11.135) * 1.4
}

func (b *Boss) actorsBusy() bool {
	for _, br := range b.branches {
		if br.running {
			return true
		}
	}
	for _, e := range b.eyes {
		if e.running {
			return true
		}
	}
	return false
}

func (b *Boss) busy() bool { return b.actorsBusy() || b.vstate != 0 }

func (b *Boss) Update(dt, px, py float64) {
	b.dt, b.px, b.py = dt, px, py
	b.time += dt
	b.frame++
	b.hist[b.frame%len(b.hist)] = [2]float64{px, py}
	b.Shake = math.Max(0, b.Shake-dt*30)
	b.tear = math.Max(0, b.tear-dt*2.2)
	b.flash = math.Max(0, b.flash-dt)
	b.dark.Update(dt)

	busy := b.busy()
	if busy != b.wasBusy {
		b.wasBusy = busy
		if busy {
			b.watcher.ent.Go(0.55, 0.8, 0.3, 1)
		} else {
			b.watcher.ent.Go(1, 1.4, 0.3, 1)
		}
	}
	b.watcher.think(b, dt, true)
	for _, br := range b.amb {
		br.updateAmbient(b, dt, busy)
	}
	for _, br := range b.branches {
		br.update(b, dt)
	}
	for _, e := range b.eyes {
		e.update(b, dt)
	}

	b.nextGlitch -= dt
	if b.nextGlitch <= 0 {
		b.tear = math.Max(b.tear, 0.35)
		b.nextGlitch = b.rr(5, 10)
	}

	b.updateVacant(dt)

	if busy {
		b.cool = 1.4
		return
	}
	b.cool -= dt
	if b.cool <= 0 {
		n := b.next
		if only >= 0 {
			n = only
		}
		b.launch(n)
		b.next = (b.next + 1) % numAttacks
	}
}

func (b *Boss) farSide() float64 {
	if b.px < arenaW/2 {
		return 1
	}
	return -1
}

func (b *Boss) lane() float64 {
	if b.py < arenaH/2 {
		return arenaH * 0.74
	}
	return arenaH * 0.26
}

func (b *Boss) launch(n int) {
	side := b.farSide()
	switch n {
	case 0:
		b.branches[0].strike(b, sideStrike(side, 3.4, int(b.rnd()*3), 0))
	case 1:
		edge := 1
		if side < 0 {
			edge = 0
		}
		b.eyes[0].fire(b, edge, b.lane(), eyeSpec{trackT: 1.3, lockT: 0.7, volleys: 2, beamW: 38})
	case 2:
		cy := clamp(b.py, 90, 150)
		b.branches[0].strike(b, crossStrike(1, cy, 0, 0.7, 0))
		b.branches[1].strike(b, crossStrike(-1, cy, 0.15, 1.0, 1))
	case 3:
		edge := 0
		if side < 0 {
			edge = 1
		}
		b.eyes[0].fire(b, edge, b.lane(), eyeSpec{trackT: 1.1, lockT: 0.7, volleys: 1, beamW: 38})
		b.eyes[1].fire(b, 2, arenaW*b.rr(0.3, 0.7), eyeSpec{delay: 1.2, trackT: 0.8, lockT: 0.55, volleys: 1, beamW: 28})
		b.branches[0].strike(b, sideStrike(side, 3.6, int(b.rnd()*3), 0.35))
	case 4:
		sp0 := sideStrike(1, 2.8, 0, 0)
		sp0.rootX, sp0.angle = -20, 0
		sp1 := sideStrike(-1, 2.8, 0, 0.25)
		sp1.yOff = clamp(arenaH/2-b.py, -1, 1) * 90
		sp1.retractDur = 1.2
		sp2 := sideStrike(1, 2.6, 0, 0.5)
		sp2.vertical = true
		sp2.rootX, sp2.rootY, sp2.angle, sp2.reach = clamp(b.px, 60, arenaW-60), -20, math.Pi/2, arenaH*1.25
		sp2.retractDur = 0.6
		b.branches[0].strike(b, sp0)
		b.branches[1].strike(b, sp1)
		b.branches[2].strike(b, sp2)
		b.eyes[2].fire(b, 3, arenaW*0.5, eyeSpec{delay: 1.4, trackT: 0.6, lockT: 0.5, volleys: 1, beamW: 26})
	case 5:
		b.eyes[0].fire(b, 0, 60, eyeSpec{trackT: 1.0, lockT: 0.55, volleys: 1, beamW: 34})
		b.eyes[1].fire(b, 1, 180, eyeSpec{delay: 0.5, trackT: 1.0, lockT: 0.55, volleys: 1, beamW: 34})
		b.eyes[2].fire(b, 2, 200, eyeSpec{delay: 1.0, trackT: 0.9, lockT: 0.5, volleys: 1, beamW: 24})
		b.eyes[3].fire(b, 3, 380, eyeSpec{delay: 1.5, trackT: 0.8, lockT: 0.5, volleys: 1, beamW: 24})
		b.branches[0].strike(b, sideStrike(side, 3.2, 1, 1.9))
	case 6:
		b.vstate, b.vt = 1, 0
		b.dark.Go(1, 0.4, 0, 1)
	}
}

func (b *Boss) updateVacant(dt float64) {
	switch b.vstate {
	case 1:
		b.vt += dt
		if b.vt > 1.6 {
			b.vstate = 2
			edge := 1
			if b.farSide() < 0 {
				edge = 0
			}
			b.eyes[0].fire(b, edge, b.lane(), eyeSpec{trackT: 0.8, lockT: 0.8, volleys: 1, beamW: 40})
			b.branches[0].strike(b, sideStrike(-b.farSide(), 3.4, 0, 1.6))
		}
	case 2:
		if !b.actorsBusy() {
			b.vstate = 3
			b.dark.Go(0, 1.0, 0, 1)
		}
	case 3:
		if b.dark.Value < 0.02 {
			b.vstate = 0
		}
	}
}

func (b *Boss) Hit(px, py, pr float64) bool {
	for _, br := range b.branches {
		if br.hit(px, py, pr) {
			return true
		}
	}
	for _, e := range b.eyes {
		if e.hit(px, py, pr) {
			return true
		}
	}
	return false
}

func (b *Boss) Draw(dst *ebiten.Image) {
	b.n = 0
	sx := math.Sin(b.time*97) * b.Shake
	sy := math.Cos(b.time*83) * b.Shake

	b.watcher.draw(dst, b, 0, 0)
	for _, br := range b.amb {
		br.draw(dst, b, 0, 0)
	}
	for _, br := range b.branches {
		br.draw(dst, b, sx, sy)
	}
	b.drawDark(dst)
	for _, br := range b.branches {
		br.drawWarn(dst, b, sx, sy)
	}
	for _, e := range b.eyes {
		e.drawBeam(dst, b, sx, sy)
	}
	for _, e := range b.eyes {
		e.draw(dst, b, sx, sy)
	}
	if b.flash > 0 {
		b.rect(dst, b.flashPix, 0, 0, arenaW, arenaH)
	}
	b.drawTear(dst)
}

func (b *Boss) line(dst, pix *ebiten.Image, x, y, ang, length, w float64) {
	var m ebiten.GeoM
	m.Scale(length, w)
	m.Translate(0, -w/2)
	m.Rotate(ang)
	m.Translate(x, y)
	dst.DrawImage(pix, &ebiten.DrawImageOptions{GeoM: m})
}

func (b *Boss) rect(dst, pix *ebiten.Image, x0, y0, x1, y1 float64) {
	if x1 <= x0 || y1 <= y0 {
		return
	}
	var m ebiten.GeoM
	m.Scale(x1-x0, y1-y0)
	m.Translate(x0, y0)
	dst.DrawImage(pix, &ebiten.DrawImageOptions{GeoM: m})
}

func (b *Boss) alpha(a float64) *ebiten.Image {
	return b.black[int(clamp(a, 0, 1)*9+0.5)]
}

func (b *Boss) outside(dst, pix *ebiten.Image, cx, cy, half float64) {
	x0, x1 := clamp(cx-half, 0, arenaW), clamp(cx+half, 0, arenaW)
	y0, y1 := clamp(cy-half, 0, arenaH), clamp(cy+half, 0, arenaH)
	b.rect(dst, pix, 0, 0, arenaW, y0)
	b.rect(dst, pix, 0, y1, arenaW, arenaH)
	b.rect(dst, pix, 0, y0, x0, y1)
	b.rect(dst, pix, x1, y0, arenaW, y1)
}

func (b *Boss) drawDark(dst *ebiten.Image) {
	d := b.dark.Value
	if d < 0.02 {
		return
	}
	b.rect(dst, b.alpha(0.3*d), 0, 0, arenaW, arenaH)
	b.outside(dst, b.alpha(0.35*d), b.px, b.py, 140)
	b.outside(dst, b.alpha(0.4*d), b.px, b.py, 80)
}

func (b *Boss) drawTear(dst *ebiten.Image) {
	if b.tear < 0.03 {
		return
	}
	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	if b.buf == nil || b.buf.Bounds().Dx() != w || b.buf.Bounds().Dy() != h {
		b.buf = ebiten.NewImage(w, h)
	}
	b.buf.Clear()
	b.buf.DrawImage(dst, nil)
	step := 8
	tick := math.Floor(b.time * 20)
	for y := 0; y < h; y += step {
		r := math.Abs(math.Sin(float64(y)*12.9898+tick*78.233))
		if r > 0.35 {
			continue
		}
		off := (r/0.35*2 - 1) * 24 * b.tear
		y1 := y + step
		if y1 > h {
			y1 = h
		}
		strip := b.buf.SubImage(image.Rect(0, y, w, y1)).(*ebiten.Image)
		var m ebiten.GeoM
		m.Translate(off, float64(y))
		dst.DrawImage(strip, &ebiten.DrawImageOptions{GeoM: m})
	}
}

type Branch struct {
	img                                    [4]*ebiten.Image
	running, live, danger, warn, ambient   bool
	rootX, rootY, angle, scale, reach      float64
	ext, tilt                              Eased
	kick                                   Spring
	tl                                     Timeline
	delay, t, seed                         float64
}

type strikeSpec struct {
	rootX, rootY, angle, scale, reach float64
	track, vertical                   bool
	mode                              int
	yOff                              float64
	delay, enter, trackT              float64
	windup, thrust, retractDur        float64
}

func sideStrike(side, scale float64, mode int, delay float64) strikeSpec {
	s := strikeSpec{rootY: arenaH / 2, track: true, scale: scale, mode: mode, delay: delay,
		reach: arenaW + 60, enter: 0.5, trackT: 1.4, windup: 0.6, thrust: 0.42, retractDur: 0.9}
	if side > 0 {
		s.rootX, s.angle = arenaW+20, math.Pi
	} else {
		s.rootX, s.angle = -20, 0
	}
	return s
}

func crossStrike(side, cy, delay, retract float64, mode int) strikeSpec {
	s := strikeSpec{scale: 3.0, mode: mode, reach: arenaW + 120, delay: delay, enter: 0.5, trackT: 0.15,
		windup: 0.7, thrust: 0.46, retractDur: retract}
	if side > 0 {
		s.rootX, s.rootY = -20, cy-80
	} else {
		s.rootX, s.rootY = arenaW+20, cy+80
	}
	s.angle = math.Atan2(cy-s.rootY, arenaW/2-s.rootX)
	return s
}

func (br *Branch) strike(b *Boss, s strikeSpec) {
	br.rootX, br.rootY, br.angle, br.scale, br.reach = s.rootX, s.rootY, s.angle, s.scale, s.reach
	br.ext.Set(0)
	br.tilt.Set(0)
	br.delay, br.running, br.live, br.danger, br.warn = s.delay, true, false, false, false
	dir := 1.0
	if math.Cos(s.angle) < 0 {
		dir = -1
	}
	follow := func(rate float64) {
		if s.vertical {
			br.rootX += (clamp(b.px, 40, arenaW-40) - br.rootX) * (1 - math.Exp(-rate*b.dt))
		} else {
			br.rootY += (clamp(b.py+s.yOff, 30, arenaH-30) - br.rootY) * (1 - math.Exp(-rate*b.dt))
		}
	}
	jit := func(v float64) float64 { return v * b.rr(0.85, 1.2) }
	impact := func(power float64) {
		b.Shake = power
		b.tear = 1
		br.kick.Vel += 8
	}

	steps := []Step{
		{Dur: s.enter, Enter: func() {
			br.live = true
			if s.track {
				if s.vertical {
					br.rootX = clamp(b.px, 40, arenaW-40)
				} else {
					br.rootY = clamp(b.py+s.yOff, 30, arenaH-30)
				}
			}
			br.ext.Go(0.22, s.enter, 0.1, 1)
		}},
		{Dur: jit(s.trackT), Tick: func(p float64) {
			if s.track {
				follow(3.5)
			}
		}},
		{Dur: s.windup, Enter: func() {
			br.warn = true
			br.ext.Go(0.14, s.windup, 0.3, 1)
		}, Tick: func(p float64) {
			br.tilt.Set(math.Sin(br.t*70) * 0.012 * p)
		}},
	}

	thrust := func(dur float64) Step {
		return Step{Dur: dur, Enter: func() {
			br.warn = false
			br.tilt.Set(-0.05 * dir)
			br.tilt.Go(0, dur, 0, 1)
			br.ext.Go(1.02, dur, -0.25, 1.1)
		}, Tick: func(p float64) { br.danger = p > 0.05 }}
	}
	lurch := func(to, dur float64) Step {
		return Step{Dur: dur, Enter: func() {
			br.warn = false
			br.danger = true
			br.ext.Go(to, dur, 0.1, 1)
		}}
	}
	hold := func(dur float64) Step { return Step{Dur: dur} }

	switch s.mode {
	case 1:
		steps = append(steps,
			lurch(0.45, 0.15), hold(0.13),
			lurch(0.72, 0.12), hold(0.28),
			lurch(1.02, 0.1))
	case 2:
		steps = append(steps,
			lurch(0.62, 0.2), hold(0.08),
			Step{Dur: 0.3, Enter: func() {
				br.danger = false
				br.ext.Go(0.16, 0.3, 0.2, 1)
			}, Tick: func(p float64) { follow(7) }},
			Step{Dur: 0.45, Enter: func() { br.warn = true }, Tick: func(p float64) {
				follow(7)
				br.tilt.Set(math.Sin(br.t*80) * 0.015)
			}},
			thrust(s.thrust*0.85))
	case 3:
		steps = append(steps,
			Step{Dur: 0.9, Enter: func() {
				br.warn, br.danger = false, true
				br.ext.Go(0.92, 0.25, 0, 1)
				br.tilt.Set(-0.75)
				br.tilt.Go(0.75, 0.9, 0.3, 0.7)
			}},
			Step{Dur: 0.9, Enter: func() { br.tilt.Go(-0.75, 0.9, 0.3, 0.7) }})
	default:
		steps = append(steps, thrust(s.thrust))
	}

	steps = append(steps,
		Step{Dur: 0.25, Enter: func() { impact(9) }},
		Step{Dur: s.retractDur, Enter: func() {
			br.danger = false
			br.ext.Go(0, s.retractDur, 0.35, 1)
			br.tilt.Go(0, s.retractDur, 0, 1)
		}},
		Step{Dur: 0.02, Enter: func() { br.live = false }},
	)
	br.tl.Start(steps...)
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

func (br *Branch) updateAmbient(b *Boss, dt float64, busy bool) {
	br.t += dt
	br.kick.Update(dt)
	want := 0.2 + 0.04*math.Sin(br.t*0.8+br.seed)
	if busy {
		want = 0.02
	}
	br.ext.Value += (want - br.ext.Value) * (1 - math.Exp(-2*dt))
	br.tilt.Value = 0.07*math.Sin(br.t*0.6+br.seed) + 0.0004*(b.py-arenaH/2)
}

func (br *Branch) geometry() (tx, ty, dx, dy, a, L float64) {
	a = br.angle + br.tilt.Value
	dx, dy = math.Cos(a), math.Sin(a)
	L = branchLen * br.scale
	d := br.ext.Value*br.reach - 30
	return br.rootX + dx*d, br.rootY + dy*d, dx, dy, a, L
}

func (br *Branch) hit(px, py, pr float64) bool {
	if !br.live || !br.danger {
		return false
	}
	tx, ty, dx, dy, _, L := br.geometry()
	for d := 0.0; d < math.Min(L, 900); d += 20 {
		t := 1 - d/L
		r := (10 - 5*t) * br.scale
		if math.Hypot(px-(tx-dx*d), py-(ty-dy*d)) < r+pr {
			return true
		}
	}
	return false
}

func (br *Branch) drawWarn(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !br.live || !br.warn {
		return
	}
	tx, ty, dx, dy, a, _ := br.geometry()
	b.line(dst, b.warnDim, tx+sx, ty+sy, a, 700, 6)
	_ = dx
	_ = dy
}

func (br *Branch) draw(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !br.live {
		return
	}
	tx, ty, dx, dy, a, L := br.geometry()
	bx, by := tx-dx*L+sx, ty-dy*L+sy
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
		m.Scale(br.scale, br.scale)
		m.Rotate(a)
		jx, jy := b.jit()
		m.Translate(bx+jx, by+jy)
		dst.DrawImage(br.img[i], &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
	}
}

type Eye struct {
	img                   *ebiten.Image
	scale                 float64
	running, live, danger bool

	homeX, homeY, outX, outY float64
	ent, gOff, blink, pop    Eased
	aim, look                float64
	mode                     int
	width, pulse, t          float64
	lag, beamW               float64
	nextBlink, nextGlance    float64
	glanceHold               float64
	closing                  bool
	tl                       Timeline
	delay                    float64
}

func newEye(img *ebiten.Image, scale float64) *Eye {
	return &Eye{img: img, scale: scale, lag: 0.2, nextBlink: 2, nextGlance: 1}
}

func (e *Eye) size() float64 { return float64(e.img.Bounds().Dx()) / pxPerMM * e.scale }

func (e *Eye) center() (float64, float64) {
	d := (1 - e.ent.Value) * (e.size() + 60)
	return e.homeX + e.outX*d, e.homeY + e.outY*d
}

type eyeSpec struct {
	delay, trackT, lockT float64
	volleys              int
	beamW                float64
}

func (e *Eye) think(b *Boss, dt float64, watching bool) {
	e.t += dt
	e.ent.Update(dt)
	e.gOff.Update(dt)
	e.blink.Update(dt)
	e.pop.Update(dt)

	calm := watching || e.mode == 1
	e.nextBlink -= dt
	if calm && e.nextBlink <= 0 && !e.closing {
		e.blink.Go(1, 0.06, 0, 1)
		e.closing = true
		e.nextBlink = b.rr(2.2, 5)
	}
	if e.closing && e.blink.Value >= 0.99 {
		e.blink.Go(0, 0.14, 0, 1)
		e.closing = false
	}

	e.nextGlance -= dt
	if e.glanceHold > 0 {
		e.glanceHold -= dt
		if e.glanceHold <= 0 {
			e.gOff.Go(0, 0.09, 0, 1)
		}
	}
	if calm && e.nextGlance <= 0 && e.glanceHold <= 0 {
		e.gOff.Go(b.rr(0.7, 1.4)*math.Copysign(1, b.rnd()-0.5), 0.07, 0, 1)
		e.glanceHold = b.rr(0.2, 0.4)
		e.nextGlance = b.rr(0.8, 2.2)
	}
	if !calm {
		e.gOff.Go(0, 0.1, 0, 1)
	}

	if calm {
		x, y := e.center()
		px, py := b.past(e.lag)
		want := math.Atan2(py-y, px-x) + e.gOff.Value
		e.look += wrap(want-e.look) * (1 - math.Exp(-9*dt))
	}
}

func (e *Eye) fire(b *Boss, edge int, along float64, s eyeSpec) {
	switch edge {
	case 0:
		e.homeX, e.homeY, e.outX, e.outY = 12, along, -1, 0
	case 1:
		e.homeX, e.homeY, e.outX, e.outY = arenaW-12, along, 1, 0
	case 2:
		e.homeX, e.homeY, e.outX, e.outY = along, 12, 0, -1
	default:
		e.homeX, e.homeY, e.outX, e.outY = along, arenaH-12, 0, 1
	}
	e.beamW = s.beamW
	e.ent.Set(0)
	e.mode, e.width, e.pulse = 0, 0, 0
	e.delay, e.running, e.live, e.danger = s.delay, true, false, false

	ang := func() float64 {
		x, y := e.center()
		return math.Atan2(b.py-y, b.px-x)
	}
	steps := []Step{
		{Dur: 0.7, Enter: func() {
			e.live = true
			e.aim = ang()
			e.look = e.aim
			e.ent.Go(1, 0.7, 0.1, 1)
		}},
		{Dur: s.trackT, Enter: func() { e.mode = 1 }},
	}
	volley := func(lockT float64) {
		steps = append(steps,
			Step{Dur: 0.2, Enter: func() {
				e.mode = 4
				e.look = ang()
				e.pop.Go(0.14, 0.05, 0, 1)
			}, Tick: func(p float64) {
				if p > 0.5 {
					e.pop.Go(0, 0.12, 0, 1)
				}
			}},
			Step{Dur: lockT, Enter: func() {
				e.mode = 2
				e.aim = ang()
				e.look = e.aim
				e.ent.Go(0.9, lockT, 0.3, 1)
			}, Tick: func(p float64) {
				e.pulse = math.Sin(e.t*45) * 0.03 * p
			}},
			Step{Dur: 0.55, Enter: func() {
				e.mode, e.pulse = 3, 0
				e.ent.Go(1, 0.1, 0, 1)
				b.Shake = 8
				b.tear = 1
				b.flash = 0.06
			}, Tick: func(p float64) {
				env := math.Min(p/0.12, 1)
				if p > 0.7 {
					env *= (1 - p) / 0.3
				}
				e.width = e.beamW * env
				e.danger = env > 0.4
			}},
			Step{Dur: 0.12, Enter: func() { e.mode, e.danger, e.width = 4, false, 0 }},
		)
	}
	volley(s.lockT)
	for i := 1; i < s.volleys; i++ {
		volley(s.lockT * 0.65)
	}
	steps = append(steps,
		Step{Dur: 0.3, Enter: func() { e.mode = 0 }},
		Step{Dur: 0.5, Enter: func() { e.ent.Go(0, 0.5, 0.4, 1) }},
		Step{Dur: 0.02, Enter: func() { e.live = false }},
	)
	e.tl.Start(steps...)
}

func (e *Eye) update(b *Boss, dt float64) {
	if !e.live && !e.running {
		return
	}
	e.think(b, dt, false)
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
	ex, ey := x+math.Cos(e.aim)*1400, y+math.Sin(e.aim)*1400
	return segDist(px, py, x, y, ex, ey) < e.width*0.42+pr
}

func (e *Eye) drawBeam(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !e.live || e.delay > 0 {
		return
	}
	x, y := e.center()
	x, y = x+sx, y+sy
	switch e.mode {
	case 2:
		pix := b.warnDim
		if int(e.t*14)%2 == 0 {
			pix = b.warnHot
		}
		b.line(dst, pix, x, y, e.aim, 1400, 3)
	case 3:
		_, jy := b.jit()
		w := e.width + jy
		b.line(dst, b.beamOut, x, y, e.aim, 1400, w)
		b.line(dst, b.beamCore, x, y, e.aim, 1400, w*0.4)
	}
}

func (e *Eye) draw(dst *ebiten.Image, b *Boss, sx, sy float64) {
	if !e.live || e.delay > 0 {
		return
	}
	x, y := e.center()
	y += 3 * math.Sin(e.t*1.7)
	jx, jy := b.jit()
	w, h := float64(e.img.Bounds().Dx()), float64(e.img.Bounds().Dy())
	s := e.scale / pxPerMM * (1 + e.pulse + e.pop.Value)
	var m ebiten.GeoM
	m.Translate(-w/2, -h/2)
	m.Rotate(e.look - irisAngle)
	m.Scale(s, s)
	m.Scale(1, 1-0.9*e.blink.Value)
	m.Translate(x+jx+sx, y+jy+sy)
	dst.DrawImage(e.img, &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
}
//