package physics

import (
	"math"
)

type AxisController struct {
	Active bool
	Time float64
	StartFloor float64
	CurrentV float64
}

func (c *AxisController) Update(dt, vf, p1, p2 float64) {
	const weight = 1
	c.Time += dt / weight
	if c.Active {
		if c.Time >=1.0 {
			c.Time = 1.0
			c.CurrentV = vf
		}else{
			c.CurrentV = cubicBezier(c.Time, c.StartFloor, vf, p1, p2)
		}
	} else {
			if c.Time >= 1.0 {
				c.Time =1.0
				c.CurrentV = 0.0
			}else{
				c.CurrentV = cubicBezier(c.Time,c.StartFloor,0.0,p1, p2)
			}
		}
	}

func (c *AxisController) HandlePress(){
	if !c.Active {
		c.Active = true
		c.StartFloor = c.CurrentV
		c.Time = 0.0
	}
}
func (c *AxisController) HandleRelease(){
	if c.Active {
		c.Active = false
		c.StartFloor = c.CurrentV
		c.Time = 0.0
	}
}

type ShipEngine struct {
	Left, Right, Up, Down AxisController
	Vf float64
	P1, P2 float64
	X, Y float64
	Heading float64
}
func NewShipEngine(maxSpeed float64) *ShipEngine {
	return &ShipEngine{
		Vf: maxSpeed,
		P1: 0.42,
		P2: 0.58,
	}
}

func (s *ShipEngine) UpdateSystem(dt float64) {
	s.Left.Update(dt, s.Vf, s.P1, s.P2)
	s.Right.Update(dt, s.Vf, s.P1, s.P2)
	s.Up.Update(dt, s.Vf, s.P1, s.P2)
	s.Down.Update(dt, s.Vf, s.P1, s.P2)
	vx := s.Right.CurrentV - s.Left.CurrentV
	vy := s.Down.CurrentV - s.Up.CurrentV
	vRaw := math.Sqrt(vx*vx + vy*vy)
	if vRaw > s.Vf {
		vx *= (s.Vf / vRaw)
		vy *= (s.Vf / vRaw)
	}
	s.X += vx * dt
	s.Y += vy * dt
	if vRaw > 0.01 {
		targetTheta := math.Atan2(vy, vx)
		const dampeningSpeed = 8.0
		diff := targetTheta - s.Heading
		diff = math.Atan2(math.Sin(diff), math.Cos(diff))
		s.Heading +=diff * dampeningSpeed * dt
	}
}

func cubicBezier(t, vi, vf, p1, p2 float64) float64 {
	v1 := vi + p1*(vf-vi)
	v2 := vi + p2*(vf-vi)
	u := 1 - t
	tt := t*t
	uu := u * u
	return (uu * u * vi) + (3* uu*t*v1) + (3*u*tt*v2) + (tt *t * vf)

}