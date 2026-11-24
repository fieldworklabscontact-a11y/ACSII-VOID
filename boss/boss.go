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
	branchLen = 341.0
	baseX     = 8.5
	baseY     = 224.3
	axisFix   = 0.2413
	irisAngle = 0.1

	onlyPhase = -1
)

const (
	Executioner = iota
	Stalker
	Divider
	Witness
)

type spot struct{ x, y float64 }

type vinePiece struct {
	name       string
	mm         float64
	at         spot
	amp, freq  float64
	phase, kck float64
}

var vine = [4]vinePiece{
	{"midbranch_1", 238.3, spot{6.448102, 174.35862}, 0.05, 2.1, 1.6, 0.3},
	{"midbranch_2", 118.2, spot{20.814335, 176.88405}, 0.05, 2.6, 2.4, 0.4},
	{"midbranch", 142.5, spot{89.826928, 173.82843}, 0.06, 1.7, 0.8, 0.7},
	{"endbranch", 260.8, spot{79.033332, 133.82167}, 0.05, 1.3, 0.0, 1.0},
}

type phase struct {
	mask   int
	dur    float64
	inten  float64
	rest   bool
	dark   bool
	branch float64
}

var script = []phase{
	{mask: 1, dur: 16, inten: 0.5, branch: 5.5},
	{rest: true, dur: 3.5},
	{mask: 2 | 8, dur: 18, inten: 0.7, branch: 5},
	{rest: true, dur: 3.5},
	{mask: 1 | 4, dur: 18, inten: 0.85, branch: 4.5},
	{rest: true, dur: 3.5},
	{mask: 15, dur: 24, inten: 1, dark: true, branch: 4},
	{rest: true, dur: 6},
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

func loadRim(name string, mm float64, c color.RGBA) *ebiten.Image {
	f, err := assets.Open("assets/" + name + ".png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	bd := src.Bounds()
	w, h := bd.Dx(), bd.Dy()
	r := int(math.Round(0.55 * float64(w) / mm))
	if r < 1 {
		r = 1
	}
	a := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, _, _, al := src.At(bd.Min.X+x, bd.Min.Y+y).RGBA()
			a[y*w+x] = uint8(al >> 8)
		}
	}
	tmp := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var m uint8
			for k := -r; k <= r; k++ {
				if xx := x + k; xx >= 0 && xx < w && a[y*w+xx] > m {
					m = a[y*w+xx]
				}
			}
			tmp[y*w+x] = m
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var m uint8
			for k := -r; k <= r; k++ {
				if yy := y + k; yy >= 0 && yy < h && tmp[yy*w+x] > m {
					m = tmp[yy*w+x]
				}
			}
			if m > 24 {
				out.SetRGBA(x, y, c)
			}
		}
	}
	return ebiten.NewImageFromImage(out)
}

func pixel(c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(c)
	return img
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func wrap(a float64) float64          { return math.Atan2(math.Sin(a), math.Cos(a)) }

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
	W, H             float64
	px, py, dt, time float64
	n                int
	seed             uint32
	hist             [90][2]float64
	frame            int

	Shake float64
	tear  float64
	flash float64
	dark  Eased
	heart float64
	inten float64

	ph        int
	pt        float64
	phStarted bool
	darkWake  float64

	branches [4]*Branch
	amb      [2]*Branch
	eyes     [4]*Eye

	bcool, nextGlitch float64
	nextBranch        int
	fight, wasFight   bool

	warnDim, warnHot, beamOut, beamCore, flashPix *ebiten.Image
	black                                         [10]*ebiten.Image
	buf                                           *ebiten.Image
}

func New(w, h int) *Boss {
	b := &Boss{W: float64(w), H: float64(h), seed: 12345, nextGlitch: 4, bcool: 3, inten: 0.3}
	if onlyPhase >= 0 {
		b.ph = onlyPhase
	}
	imgs, rims := [4]*ebiten.Image{}, [4]*ebiten.Image{}
	for j, p := range vine {
		imgs[j] = load(p.name)
		rims[j] = loadRim(p.name, p.mm, color.RGBA{190, 35, 55, 255})
	}
	newBranch := func(i int) *Branch {
		br := &Branch{img: imgs, rim: rims, seed: float64(i) * 2.3, scale: 3}
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

	b.eyes[Executioner] = newEye(b, Executioner, load("eye2"), 45.7, 3.6, 0, 0.1)
	b.eyes[Stalker] = newEye(b, Stalker, load("eye3"), 45.7, 3.6, 1, 0.04)
	b.eyes[Divider] = newEye(b, Divider, load("eye1"), 25, 5.0, 2, 0.3)
	b.eyes[Witness] = newEye(b, Witness, load("eye4"), 25, 4.4, 3, 0.55)
	b.layout()

	b.warnDim = pixel(color.RGBA{110, 18, 28, 110})
	b.warnHot = pixel(color.RGBA{255, 70, 80, 255})
	b.beamOut = pixel(color.RGBA{255, 40, 60, 255})
	b.beamCore = pixel(color.RGBA{150, 0, 28, 255})
	b.flashPix = pixel(color.RGBA{90, 20, 25, 90})
	for i := range b.black {
		b.black[i] = pixel(color.RGBA{0, 0, 0, uint8(255 * i / 9)})
	}
	return b
}

func (b *Boss) PlayerHit() {
	b.react()
	b.tear = 1
	b.flash = math.Max(b.flash, 0.04)
}

func (b *Boss) Resize(w, h int) {
	if float64(w) == b.W && float64(h) == b.H {
		return
	}
	b.W, b.H = float64(w), float64(h)
	b.layout()
}

func (b *Boss) layout() {
	b.amb[0].rootX, b.amb[0].rootY, b.amb[0].angle = -30, 10, 0.62
	b.amb[1].rootX, b.amb[1].rootY, b.amb[1].angle = b.W+30, b.H+10, math.Pi+0.5
	for _, e := range b.eyes {
		lo, hi := e.alongRange()
		if e.along.Value == 0 {
			e.along.Set((lo + hi) / 2)
		}
		e.along.Set(clamp(e.along.Value, lo, hi))
	}
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

func (b *Boss) react() {
	for _, e := range b.eyes {
		e.pop.Go(0.09, 0.04, 0, 1)
	}
}

func (b *Boss) stare() {
	for _, e := range b.eyes {
		x, y := e.center()
		e.look = math.Atan2(b.py-y, b.px-x)
		e.pop.Go(0.14, 0.05, 0, 1)
		e.blink.Set(0)
	}
	b.tear, b.flash = 1, 0.08
}

func (b *Boss) branchesRunning() bool {
	for _, br := range b.branches {
		if br.running {
			return true
		}
	}
	return false
}

func (b *Boss) startPhase(i int) {
	p := script[i]
	b.phStarted = true
	b.pt = 0
	b.darkWake = 0
	for k, e := range b.eyes {
		e.active = p.mask&(1<<k) != 0
		e.cool = []float64{0.3, 0.9, 1.5, 0.6}[k]
	}
	if p.dark {
		b.dark.Go(1, 0.4, 0, 1)
		b.darkWake = 1.8
		for _, e := range b.eyes {
			e.cool += b.darkWake
		}
	}
	if !p.rest {
		b.stare()
		b.bcool = p.branch * 0.6
	}
}

func (b *Boss) Update(dt, px, py float64) {
	b.dt, b.px, b.py = dt, px, py
	b.time += dt
	b.frame++
	b.hist[b.frame%len(b.hist)] = [2]float64{px, py}
	b.Shake = math.Max(0, b.Shake-dt*30)
	b.tear = math.Max(0, b.tear-dt*2.2)
	b.flash = math.Max(0, b.flash-dt)
	b.dark.Update(dt)

	if !b.phStarted {
		b.startPhase(b.ph)
	}
	p := script[b.ph]
	b.pt += dt
	if b.darkWake > 0 {
		b.darkWake -= dt
		if b.darkWake <= 0 {
			b.dark.Go(0.62, 0.6, 0, 1)
		}
	}
	if b.pt >= p.dur {
		if p.dark {
			b.dark.Go(0, 1.2, 0, 1)
		}
		if onlyPhase >= 0 {
			b.startPhase(onlyPhase)
			b.ph = onlyPhase
		} else {
			b.ph = (b.ph + 1) % len(script)
			b.startPhase(b.ph)
		}
		p = script[b.ph]
	}
	want := 0.25
	if !p.rest {
		want = p.inten
	}
	b.inten += (want - b.inten) * (1 - math.Exp(-0.8*dt))
	b.heart += dt * 2 * math.Pi * (0.8 + 1.2*b.inten)
	b.fight = !p.rest
	if b.fight != b.wasFight {
		b.wasFight = b.fight
	}

	for _, br := range b.amb {
		br.updateAmbient(b, dt, b.fight)
	}
	for _, br := range b.branches {
		br.update(b, dt)
	}
	for _, e := range b.eyes {
		e.update(b, dt)
	}

	b.nextGlitch -= dt
	if b.nextGlitch <= 0 {
		b.tear = math.Max(b.tear, 0.3+0.3*b.inten)
		b.nextGlitch = b.rr(3, 8) / (0.5 + b.inten)
	}

	if p.branch > 0 && b.darkWake <= 0 && !b.branchesRunning() {
		b.bcool -= dt
		if b.bcool <= 0 {
			order := [4]int{0, 2, 3, 1}
			b.launch(order[b.nextBranch%4])
			b.nextBranch++
			b.bcool = p.branch * b.rr(0.8, 1.2)
		}
	}
}

func (b *Boss) farSide() float64 {
	if b.px < b.W/2 {
		return 1
	}
	return -1
}

func (b *Boss) launch(n int) {
	side := b.farSide()
	switch n {
	case 0:
		b.branches[0].strike(b, b.sideStrike(side, 3.4, int(b.rnd()*3), 0))
	case 1:
		sp0 := b.sideStrike(-1, 2.8, 0, 0)
		sp1 := b.sideStrike(1, 2.8, 0, 0.25)
		sp1.yOff = clamp(b.H/2-b.py, -1, 1) * 0.375 * b.H
		sp1.retractDur = 1.2
		sp2 := b.sideStrike(1, 2.6, 0, 0.5)
		sp2.vertical = true
		sp2.rootX, sp2.rootY, sp2.angle, sp2.reach = clamp(b.px, 60, b.W-60), -20, math.Pi/2, b.H*1.25
		sp2.retractDur = 0.6
		b.branches[0].strike(b, sp0)
		b.branches[1].strike(b, sp1)
		b.branches[2].strike(b, sp2)
	case 2:
		cy := clamp(b.py, 0.375*b.H, 0.625*b.H)
		b.branches[0].strike(b, b.crossStrike(1, cy, 0, 0.7, 0))
		b.branches[1].strike(b, b.crossStrike(-1, cy, 0.15, 1.0, 1))
	case 3:
		sp := b.sideStrike(1, 3.0, 3, 0)
		m := math.Min(120, b.W/4)
		sp.rootX, sp.rootY, sp.angle, sp.reach = clamp(b.W-b.px, m, b.W-m), -20, math.Pi/2, b.H*1.2
		sp.track = false
		sp.windup = 0.8
		b.branches[0].strike(b, sp)
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
	b.Resize(dst.Bounds().Dx(), dst.Bounds().Dy())
	b.n = 0
	sx := math.Sin(b.time*97) * b.Shake
	sy := math.Cos(b.time*83) * b.Shake

	for _, br := range b.amb {
		br.draw(dst, b, 0, 0)
	}
	b.drawDark(dst)
	for _, br := range b.branches {
		br.draw(dst, b, sx, sy)
	}
	for _, br := range b.branches {
		br.drawWarn(dst, b, sx, sy)
	}
	for _, e := range b.eyes {
		e.drawMarks(dst, b)
	}
	for _, e := range b.eyes {
		e.drawBeam(dst, b, sx, sy)
	}
	for _, e := range b.eyes {
		e.draw(dst, b, sx, sy)
	}
	if b.flash > 0 {
		b.rect(dst, b.flashPix, 0, 0, b.W, b.H)
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

func (b *Boss) frameRect(dst, pix *ebiten.Image, cx, cy, half, th float64) {
	b.rect(dst, pix, cx-half, cy-half, cx+half, cy-half+th)
	b.rect(dst, pix, cx-half, cy+half-th, cx+half, cy+half)
	b.rect(dst, pix, cx-half, cy-half+th, cx-half+th, cy+half-th)
	b.rect(dst, pix, cx+half-th, cy-half+th, cx+half, cy+half-th)
}

func (b *Boss) alpha(a float64) *ebiten.Image {
	return b.black[int(clamp(a, 0, 1)*9+0.5)]
}

func (b *Boss) outside(dst, pix *ebiten.Image, cx, cy, half float64) {
	x0, x1 := clamp(cx-half, 0, b.W), clamp(cx+half, 0, b.W)
	y0, y1 := clamp(cy-half, 0, b.H), clamp(cy+half, 0, b.H)
	b.rect(dst, pix, 0, 0, b.W, y0)
	b.rect(dst, pix, 0, y1, b.W, b.H)
	b.rect(dst, pix, 0, y0, x0, y1)
	b.rect(dst, pix, x1, y0, b.W, y1)
}

func (b *Boss) beat() float64 {
	s := math.Max(0, math.Sin(b.heart))
	return s * s * s
}

func (b *Boss) drawDark(dst *ebiten.Image) {
	d := b.dark.Value
	if d >= 0.02 {
		b.rect(dst, b.alpha(0.3*d), 0, 0, b.W, b.H)
		b.outside(dst, b.alpha(0.35*d), b.px, b.py, 140)
		b.outside(dst, b.alpha(0.4*d), b.px, b.py, 80)
	}
	if b.fight {
		b.outside(dst, b.alpha(0.16*b.beat()*b.inten), b.W/2, b.H/2, 0.37*b.W)
	}
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
		r := math.Abs(math.Sin(float64(y)*12.9898 + tick*78.233))
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
	img, rim                             [4]*ebiten.Image
	running, live, danger, warn, ambient bool
	rootX, rootY, angle, scale, reach    float64
	ext, tilt                            Eased
	kick                                 Spring
	tl                                   Timeline
	delay, t, seed                       float64
}

type strikeSpec struct {
	rootX, rootY, angle, scale, reach float64
	track, vertical                   bool
	mode                              int
	yOff                              float64
	delay, enter, trackT              float64
	windup, thrust, retractDur        float64
}

func (b *Boss) sideStrike(side, scale float64, mode int, delay float64) strikeSpec {
	s := strikeSpec{rootY: b.H / 2, track: true, scale: scale, mode: mode, delay: delay,
		reach: b.W + 60, enter: 0.5, trackT: 1.4, windup: 0.6, thrust: 0.42, retractDur: 0.9}
	if side > 0 {
		s.rootX, s.angle = b.W+20, math.Pi
	} else {
		s.rootX, s.angle = -20, 0
	}
	return s
}

func (b *Boss) crossStrike(side, cy, delay, retract float64, mode int) strikeSpec {
	s := strikeSpec{scale: 3.0, mode: mode, reach: b.W + 120, delay: delay, enter: 0.5, trackT: 0.15,
		windup: 0.7, thrust: 0.46, retractDur: retract}
	if side > 0 {
		s.rootX, s.rootY = -20, cy-0.33*b.H
	} else {
		s.rootX, s.rootY = b.W+20, cy+0.33*b.H
	}
	s.angle = math.Atan2(cy-s.rootY, b.W/2-s.rootX)
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
			br.rootX += (clamp(b.px, 40, b.W-40) - br.rootX) * (1 - math.Exp(-rate*b.dt))
		} else {
			br.rootY += (clamp(b.py+s.yOff, 30, b.H-30) - br.rootY) * (1 - math.Exp(-rate*b.dt))
		}
	}
	jit := func(v float64) float64 { return v * b.rr(0.85, 1.2) }
	impact := func(power float64) {
		b.Shake = power
		b.tear = 1
		br.kick.Vel += 8
		b.react()
	}

	steps := []Step{
		{Dur: s.enter, Enter: func() {
			br.live = true
			if s.track {
				if s.vertical {
					br.rootX = clamp(b.px, 40, b.W-40)
				} else {
					br.rootY = clamp(b.py+s.yOff, 30, b.H-30)
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

func (br *Branch) updateAmbient(b *Boss, dt float64, fight bool) {
	br.t += dt
	br.kick.Update(dt)
	want := 0.2 + 0.04*math.Sin(br.t*0.8+br.seed)
	if fight {
		want = 0.02
	}
	br.ext.Value += (want - br.ext.Value) * (1 - math.Exp(-2*dt))
	br.tilt.Value = 0.07*math.Sin(br.t*0.6+br.seed) + 0.0004*(b.py-b.H/2)
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
	tx, ty, _, _, a, _ := br.geometry()
	b.line(dst, b.warnDim, tx+sx, ty+sy, a, 700, 6)
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
		k := p.mm / float64(br.img[i].Bounds().Dx())
		m.Scale(k, k)
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
		op := &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear}
		dst.DrawImage(br.rim[i], op)
		dst.DrawImage(br.img[i], op)
	}
}

const (
	mIdle = iota
	mTrack
	mLock
	mFire
	mSettle
	mCharge
	mWedge
	mAway
)

type mark struct {
	x, y, t float64
	on      bool
}

type Eye struct {
	b     *Boss
	kind  int
	edge  int
	img   *ebiten.Image
	mm    float64
	scale float64

	active, running, danger bool

	along, ent, gOff, blink, pop, sweep Eased
	aim, look                           float64
	mode                                int
	width, pulse, t                     float64
	lag, beamW                          float64
	nextBlink, nextGlance, glanceHold   float64
	nextDrift, cool                     float64
	closing                             bool
	a0, a1                              float64
	tl                                  Timeline
	marks                               [3]mark
}

func newEye(b *Boss, kind int, img *ebiten.Image, mm, scale float64, edge int, lag float64) *Eye {
	e := &Eye{b: b, kind: kind, edge: edge, img: img, mm: mm, scale: scale, lag: lag,
		nextBlink: 1.5 + float64(kind), nextGlance: 1 + 0.7*float64(kind), nextDrift: 3 + float64(kind)}
	e.ent.Set(0.8)
	return e
}

func (e *Eye) size() float64 { return e.mm * e.scale }

func (e *Eye) home() (float64, float64) {
	switch e.edge {
	case 0:
		return 12, e.along.Value
	case 1:
		return e.b.W - 12, e.along.Value
	case 2:
		return e.along.Value, 12
	}
	return e.along.Value, e.b.H - 12
}

func (e *Eye) out() (float64, float64) {
	switch e.edge {
	case 0:
		return -1, 0
	case 1:
		return 1, 0
	case 2:
		return 0, -1
	}
	return 0, 1
}

func (e *Eye) center() (float64, float64) {
	hx, hy := e.home()
	ox, oy := e.out()
	d := (1 - e.ent.Value) * (e.size() + 60)
	return hx + ox*d, hy + oy*d
}

func (e *Eye) ang(b *Boss) float64 {
	x, y := e.center()
	return math.Atan2(b.py-y, b.px-x)
}

func (e *Eye) alongRange() (float64, float64) {
	if e.edge < 2 {
		return 0.2 * e.b.H, 0.8 * e.b.H
	}
	return 0.16 * e.b.W, 0.84 * e.b.W
}

func (e *Eye) think(b *Boss, dt float64) {
	e.t += dt
	e.ent.Update(dt)
	e.along.Update(dt)
	e.gOff.Update(dt)
	e.blink.Update(dt)
	e.pop.Update(dt)
	e.sweep.Update(dt)
	if e.pop.t >= 1 && e.pop.Value > 0.001 {
		e.pop.Go(0, 0.25, 0, 1)
	}

	calm := e.mode == mIdle || e.mode == mTrack || e.mode == mSettle
	e.nextBlink -= dt
	if (calm || e.mode == mAway) && e.nextBlink <= 0 && !e.closing {
		e.blink.Go(1, 0.06, 0, 1)
		e.closing = true
		e.nextBlink = b.rr(1.6, 4.5)
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
	if (e.mode == mIdle || e.mode == mTrack) && e.nextGlance <= 0 && e.glanceHold <= 0 {
		e.gOff.Go(b.rr(0.7, 1.5)*math.Copysign(1, b.rnd()-0.5), 0.07, 0, 1)
		e.glanceHold = b.rr(0.2, 0.45)
		e.nextGlance = b.rr(0.8, 2.4)
	}
	if !(e.mode == mIdle || e.mode == mTrack) && e.glanceHold <= 0 && e.gOff.Value != 0 {
		e.gOff.Go(0, 0.1, 0, 1)
	}

	switch e.mode {
	case mIdle, mTrack, mSettle:
		x, y := e.center()
		px, py := b.past(e.lag)
		want := math.Atan2(py-y, px-x) + e.gOff.Value
		e.look += wrap(want-e.look) * (1 - math.Exp(-9*dt))
	case mAway:
		want := e.ang(b) + math.Pi*0.85
		e.look += wrap(want-e.look) * (1 - math.Exp(-5*dt))
	default:
		e.look = e.aim
	}
}

func (e *Eye) idleMotion(b *Boss, dt float64) {
	if e.ent.t >= 1 {
		want := 0.8 + 0.03*math.Sin(e.t*0.9+float64(e.kind))
		e.ent.Value += (want - e.ent.Value) * (1 - math.Exp(-3*dt))
	}
	e.nextDrift -= dt
	if e.nextDrift <= 0 {
		lo, hi := e.alongRange()
		e.along.Go(b.rr(lo, hi), 1.4, 0.3, 1)
		e.nextDrift = b.rr(4, 8)
	}
}

func (e *Eye) update(b *Boss, dt float64) {
	e.think(b, dt)
	e.updateMarks(b, dt)
	if e.running {
		if !e.tl.Update(dt) {
			e.running, e.danger, e.width = false, false, 0
			e.mode = mIdle
			e.cool = b.rr(1.2, 2.6) / (0.7 + 0.5*b.inten)
		}
		return
	}
	e.idleMotion(b, dt)
	if e.active {
		e.cool -= dt
		if e.cool <= 0 {
			e.start(b)
		}
	}
}

func (e *Eye) updateMarks(b *Boss, dt float64) {
	for i := range e.marks {
		m := &e.marks[i]
		if !m.on {
			continue
		}
		m.t += dt
		if m.t > 1.65 {
			m.on = false
		}
	}
}

func (e *Eye) start(b *Boss) {
	e.running = true
	var steps []Step
	switch e.kind {
	case Executioner:
		steps = e.scriptExecutioner(b)
	case Stalker:
		steps = e.scriptStalker(b)
	case Divider:
		steps = e.scriptDivider(b)
	default:
		steps = e.scriptWitness(b)
	}
	e.tl.Start(steps...)
}

func (e *Eye) emerge(b *Boss, dur, along float64) Step {
	return Step{Dur: dur, Enter: func() {
		e.mode = mTrack
		e.ent.Go(1, dur, 0.1, 1)
		e.along.Go(along, dur, 0.2, 1)
	}}
}

func (e *Eye) retreat() Step {
	return Step{Dur: 0.7, Enter: func() {
		e.mode, e.danger, e.width = mSettle, false, 0
		e.ent.Go(0.8, 0.7, 0.4, 1)
	}}
}

func (e *Eye) notice(b *Boss) Step {
	return Step{Dur: 0.2, Enter: func() {
		e.mode = mTrack
		e.look = e.ang(b)
		e.pop.Go(0.14, 0.05, 0, 1)
	}}
}

func (e *Eye) scriptExecutioner(b *Boss) []Step {
	lockT := 0.85 - 0.2*b.inten
	lock := func(t float64) Step {
		return Step{Dur: t, Enter: func() {
			e.mode = mLock
			e.aim = e.ang(b)
			e.look = e.aim
			e.ent.Go(0.92, t, 0.3, 1)
		}, Tick: func(p float64) { e.pulse = math.Sin(e.t*45) * 0.03 * p }}
	}
	fire := Step{Dur: 0.55, Enter: func() {
		e.mode, e.pulse = mFire, 0
		e.ent.Go(1, 0.1, 0, 1)
		b.Shake, b.tear, b.flash = 8, 1, 0.06
		b.react()
	}, Tick: func(p float64) {
		env := math.Min(p/0.12, 1)
		if p > 0.7 {
			env *= (1 - p) / 0.3
		}
		e.width = 40 * env
		e.danger = env > 0.4
	}}
	steps := []Step{
		e.emerge(b, 0.5, b.rr(0.25*b.H, 0.75*b.H)),
		{Dur: b.rr(0.8, 1.3), Enter: func() { e.mode = mTrack }},
		e.notice(b),
		lock(lockT),
	}
	if b.rnd() < 0.3+0.2*b.inten {
		steps = append(steps,
			Step{Dur: 0.35, Enter: func() {
				e.mode, e.pulse = mAway, 0
				e.ent.Go(0.85, 0.35, 0.2, 1)
			}},
			e.notice(b),
			lock(0.5))
	} else {
		steps = append(steps, Step{Dur: b.rr(0, 0.45) * b.inten})
	}
	return append(steps, fire, Step{Dur: 0.15, Enter: func() { e.mode, e.danger, e.width = mSettle, false, 0 }}, e.retreat())
}

func (e *Eye) scriptStalker(b *Boss) []Step {
	steps := []Step{
		e.emerge(b, 0.5, clamp(b.py, 0.2*b.H, 0.8*b.H)),
		{Dur: 0.7, Enter: func() {
			e.mode = mTrack
			e.along.Go(clamp(b.py, 0.2*b.H, 0.8*b.H), 0.7, 0.2, 1)
		}},
		{Dur: 0.45, Enter: func() { e.mode, e.aim = mCharge, e.ang(b) }, Tick: func(p float64) {
			e.aim += wrap(e.ang(b)-e.aim) * (1 - math.Exp(-6*b.dt))
		}},
	}
	for i := 0; i < 3; i++ {
		steps = append(steps,
			Step{Dur: 0.22, Enter: func() {
				e.mode = mFire
				b.Shake, b.flash = 3, 0.03
			}, Tick: func(p float64) {
				e.aim += wrap(e.ang(b)-e.aim) * (1 - math.Exp(-3.4*b.dt))
				env := math.Min(p/0.15, 1) * math.Min((1-p)/0.25, 1)
				e.width = 16 * env
				e.danger = env > 0.4
			}},
			Step{Dur: 0.15, Enter: func() { e.mode, e.danger, e.width = mCharge, false, 0 }, Tick: func(p float64) {
				e.aim += wrap(e.ang(b)-e.aim) * (1 - math.Exp(-6*b.dt))
			}})
	}
	return append(steps, e.retreat())
}

func (e *Eye) scriptDivider(b *Boss) []Step {
	along := b.rr(0.54*b.W, 0.77*b.W)
	if b.px > b.W/2 {
		along = b.rr(0.23*b.W, 0.46*b.W)
	}
	return []Step{
		e.emerge(b, 0.6, along),
		{Dur: 0.4, Enter: func() { e.mode = mTrack }},
		{Dur: 0.95, Enter: func() {
			e.mode = mWedge
			half := 0.62
			base := clamp(e.ang(b), 0.2+half, math.Pi-0.2-half)
			s := math.Copysign(1, b.rnd()-0.5)
			e.a0, e.a1 = base-s*half, base+s*half
			e.aim = e.a0
			e.ent.Go(0.95, 0.95, 0.3, 1)
		}, Tick: func(p float64) { e.pulse = math.Sin(e.t*30) * 0.02 * p }},
		{Dur: 1.9, Enter: func() {
			e.mode, e.pulse = mFire, 0
			e.ent.Go(1, 0.1, 0, 1)
			e.sweep.Set(e.a0)
			e.sweep.Go(e.a1, 1.9, 0.25, 0.75)
			b.Shake, b.tear, b.flash = 6, 1, 0.06
			b.react()
		}, Tick: func(p float64) {
			e.aim = e.sweep.Value
			env := math.Min(p/0.06, 1) * math.Min((1-p)/0.06, 1)
			e.width = 54 * env
			e.danger = env > 0.4
		}},
		{Dur: 0.15, Enter: func() { e.mode, e.danger, e.width = mSettle, false, 0 }},
		e.retreat(),
	}
}

func (e *Eye) scriptWitness(b *Boss) []Step {
	steps := []Step{
		e.emerge(b, 0.5, b.rr(0.2*b.W, 0.8*b.W)),
		{Dur: 0.5, Enter: func() { e.mode = mTrack }},
	}
	for i := 0; i < 3; i++ {
		i := i
		steps = append(steps, Step{Dur: 0.28, Enter: func() {
			e.mode = mTrack
			x, y := b.px, b.py
			switch i {
			case 1:
				ox, oy := b.past(0.3)
				x, y = b.px+(b.px-ox)*3.3, b.py+(b.py-oy)*3.3
			case 2:
				a := b.rnd() * 2 * math.Pi
				d := b.rr(70, 120)
				x, y = b.px+math.Cos(a)*d, b.py+math.Sin(a)*d
			}
			e.marks[i] = mark{x: clamp(x, 40, b.W-40), y: clamp(y, 40, b.H-40), on: true}
			e.pop.Go(0.1, 0.04, 0, 1)
		}})
	}
	return append(steps,
		Step{Dur: 1.0, Enter: func() { e.mode = mAway; e.ent.Go(0.7, 0.4, 0.3, 1) }},
		Step{Dur: 0.4, Enter: func() {
			e.mode = mTrack
			e.look = e.ang(b)
			e.pop.Go(0.18, 0.05, 0, 1)
			e.ent.Go(1, 0.3, 0.2, 1)
		}},
		e.retreat())
}

func (e *Eye) hit(px, py, pr float64) bool {
	for _, m := range e.marks {
		if m.on && m.t >= 1.3 && math.Max(math.Abs(px-m.x), math.Abs(py-m.y)) < 38+pr*0.5 {
			return true
		}
	}
	if !e.danger {
		return false
	}
	x, y := e.center()
	ex, ey := x+math.Cos(e.aim)*1400, y+math.Sin(e.aim)*1400
	return segDist(px, py, x, y, ex, ey) < e.width*0.42+pr
}

func (e *Eye) drawMarks(dst *ebiten.Image, b *Boss) {
	for _, m := range e.marks {
		if !m.on {
			continue
		}
		if m.t < 1.3 {
			rate := 6 + 20*m.t
			pix := b.warnDim
			if int(m.t*rate)%2 == 0 {
				pix = b.warnHot
			}
			b.frameRect(dst, pix, m.x, m.y, 38, 2)
			b.frameRect(dst, pix, m.x, m.y, 38*(1-m.t/1.3)+2, 2)
		} else {
			f := 1 - (m.t-1.3)/0.35
			b.rect(dst, b.beamOut, m.x-38, m.y-38, m.x+38, m.y+38)
			b.rect(dst, b.beamCore, m.x-38*f*0.6, m.y-38*f*0.6, m.x+38*f*0.6, m.y+38*f*0.6)
		}
	}
}

func (e *Eye) drawBeam(dst *ebiten.Image, b *Boss, sx, sy float64) {
	x, y := e.center()
	x, y = x+sx, y+sy
	switch e.mode {
	case mLock:
		pix := b.warnDim
		if int(e.t*14)%2 == 0 {
			pix = b.warnHot
		}
		b.line(dst, pix, x, y, e.aim, 1400, 3)
	case mCharge:
		b.line(dst, b.warnDim, x, y, e.aim, 1400, 2)
	case mWedge:
		b.line(dst, b.warnDim, x, y, e.a0, 1400, 2)
		b.line(dst, b.warnDim, x, y, e.a1, 1400, 2)
		if int(e.t*12)%2 == 0 {
			b.line(dst, b.warnHot, x, y, e.a0, 1400, 3)
		}
	case mFire:
		_, jy := b.jit()
		w := e.width + jy
		b.line(dst, b.beamOut, x, y, e.aim, 1400, w)
		b.line(dst, b.beamCore, x, y, e.aim, 1400, w*0.4)
	}
}

func (e *Eye) draw(dst *ebiten.Image, b *Boss, sx, sy float64) {
	x, y := e.center()
	ox, oy := e.out()
	bob := 3 * math.Sin(e.t*1.7+float64(e.kind))
	x, y = x+oy*bob, y+ox*bob
	jx, jy := b.jit()
	w, h := float64(e.img.Bounds().Dx()), float64(e.img.Bounds().Dy())
	heart := 0.035 * b.beat() * (0.4 + b.inten)
	s := e.size() / w * (1 + e.pulse + e.pop.Value + heart)
	var m ebiten.GeoM
	m.Translate(-w/2, -h/2)
	m.Rotate(e.look - irisAngle)
	m.Scale(s, s)
	m.Scale(1, 1-0.9*e.blink.Value)
	m.Translate(x+jx+sx, y+jy+sy)
	dst.DrawImage(e.img, &ebiten.DrawImageOptions{GeoM: m, Filter: ebiten.FilterLinear})
}
