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

func (c *AxisController) Update(dt, vf float64, p1, p2 float64) {
	const weight = 0.5
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
