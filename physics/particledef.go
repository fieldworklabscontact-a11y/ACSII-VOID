package physics

type Particle struct{
	Name string
	Mass float64
	DecayRate float64
	X float64
	Y float64
	VelocityX float64
	VelocityY float64
}

type SpatialAtt struct{
	Gravity float64
	Xgravity float64
	Ygravity float64
}

type Force struct {
	X float64
	Y float64
}