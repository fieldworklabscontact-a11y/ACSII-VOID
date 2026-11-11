package physics

import (
	
)

type AxisController struct {
	Active bool
	Time float64
	StartFloor float64
	CurrentV float64
}

func (c *AxisController) Update(dt, vf float64) {
	const weight = 0.5
	const p1 =0.42
	const p2 = 0.58
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

func cubicBezier(t, vi, vf, p1, p2 float64) float64 {
	v1 := vi + p1*(vf-vi)
	v2 := vi + p2*(vf-vi)
	u := 1 - t
	tt := t*t
	uu := u * u
	return (uu * u * vi) + (3* uu*t*v1) + (3*u*tt*v2) + (tt *t * vf)

}