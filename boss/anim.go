package boss

import "math"

func cubicBezier(t, vi, vf, p1, p2 float64) float64 {
	v1 := vi + p1*(vf-vi)
	v2 := vi + p2*(vf-vi)
	u := 1 - t
	return u*u*u*vi + 3*u*u*t*v1 + 3*u*t*t*v2 + t*t*t*vf
}

type Eased struct {
	Value float64
	from, to float64
	t, dur, p1, p2 float64
}

func (e *Eased) Set(v float64) { e.Value, e.from, e.to, e.t, e.dur = v, v, v, 1, 1 }

func (e *Eased) Go(to, dur, p1, p2 float64) {
	if dur <= 0 {
		e.Set(to)
		return
	}
	e.from, e.to, e.t, e.dur, e.p1, e.p2 = e.Value, to, 0, dur, p1, p2
}

func (e *Eased) Update(dt float64) {
	if e.t >= 1 || e.dur <= 0 {
		return
	}
	e.t = math.Min(1, e.t+dt/e.dur)
	e.Value = cubicBezier(e.t, e.from, e.to, e.p1, e.p2)
}
type Spring struct{ Pos, Vel, Target, K, Damp float64 }
func (s *Spring) Update(dt float64) {
	s.Vel += (s.Target - s.Pos) * s.K * dt
	s.Vel *= math.Pow(s.Damp, dt*60)
	s.Pos += s.Vel * dt
}
type Pt struct{ X, Y float64}
type Chain struct {
	Pts []Pt
	SegLen float64
	Gravity float64
}

func NewChain(n int, segLen, x, y float64) *Chain {
	c := &Chain{Pts: make([]Pt, n), SegLen: segLen, Gravity:
90}
	for i := range c.Pts {
		c.Pts[i] = Pt{x, y + float64(i)*segLen}
	}
	return c
}
func (c *Chain) Update(dt, ax, ay float64) {
	c.Pts[0] = Pt{ax, ay}
	for i := 1; i < len(c.Pts); i++{
		p, q := &c.Pts[i], c.Pts[i-1]
		p.Y += c.Gravity * dt
		dx, dy := p.X-q.X, p.Y-q.Y
		d := math.Hypot(dx, dy)
		if d < 1e-6 {
			dx, dy, d = 0, 1, 1
		}
		p.X = q.X + dx/d*c.SegLen
		p.Y = q.Y + dy/d*c.SegLen
	}
}
type Step struct {
	Dur float64
	Enter func()
	Tick func(p float64)
}
type Timeline struct {
	steps []Step
	i int
	t float64
	entered bool
}
func (t1 *Timeline) Start(steps ...Step) { *t1 = Timeline{steps: steps}}
func (t1 *Timeline) Update(dt float64) bool {
	if t1.i >= len(t1.steps) {
		return false
	}
	s := t1.steps[t1.i]
	if !t1.entered {
		t1.entered = true
		if s.Enter != nil {
			s.Enter()
		}
	}
	t1.t += dt
	if s.Tick != nil && s.Dur > 0 {
		s.Tick(math.Min(1, t1.t/s.Dur))
	}
	if t1.t >= s.Dur {
		t1.t++
		t1.t, t1.entered = 0, false
	}
	return true
}

