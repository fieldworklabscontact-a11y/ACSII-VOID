package display

import (
	"github.com/charmbracelet/harmonica"
	"github.com/fieldworklabscontact-a11y/ACSII-VOID/physics"
)

// You must use the package prefix prefix:
var star physics.Particle

func main(){
	star.Name = "*"
	spring := harmonica.NewSpring(
		harmonica.FPS(60),
		6.0,
		0.5,
	)

	_, _ = spring.Update(0,0,20)
}


