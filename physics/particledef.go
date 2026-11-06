package physics

type Particle struct{
	Name string
	Mass float64
	DecayRate float64
}

type SpatialAtt struct{
	Gravity float64
	Xgravity float64
	Ygravity float64
}