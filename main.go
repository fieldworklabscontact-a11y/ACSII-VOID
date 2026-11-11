package main

import (
	"image/color"
	"log"
	"time"

	"github.com/fieldworklabscontact-a11y/ACSII-VOID/physics"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth = 560
	screenHeight = 240
	spriteSize = 16
)

type Game struct{
	sprite *ebiten.Image
	engine *physics.ShipEngine
	lastFrame time.Time
}
func NewGame() *Game {
	paddedSize := spriteSize + 2
	img := ebiten.NewImage(paddedSize, paddedSize)
	innerSquare := ebiten.NewImage(spriteSize, spriteSize)
	innerSquare.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(1,1)
	img.DrawImage(innerSquare, op)
	engine := physics.NewShipEngine(230.0)
	engine.X = float64(screenWidth)/2 - float64(paddedSize)/2
	engine.Y = float64(screenHeight)/2 - float64(paddedSize)/2
	return &Game{
		sprite: img,
		engine: engine,
		lastFrame: time.Now(),
	}
}
func (g *Game) Update() error {
	now := time.Now()
	dt := now.Sub(g.lastFrame).Seconds()
	g.lastFrame = now
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA){
		g.engine.Left.HandlePress()
	}else{
		g.engine.Left.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD){
		g.engine.Right.HandlePress()
	}else{
		g.engine.Right.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW){
		g.engine.Up.HandlePress()
	}else{
		g.engine.Up.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS){
		g.engine.Down.HandlePress()
	}else{
		g.engine.Down.HandleRelease()
	}
	g.engine.UpdateSystem(dt)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20,24,40,225})
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(-float64(spriteSize)/2,-float64(spriteSize)/2)
	op.GeoM.Rotate(g.engine.Heading)
	op.GeoM.Translate(float64(spriteSize)/2,float64(spriteSize)/2)
	op.GeoM.Translate(g.engine.X,g.engine.Y)
	screen.DrawImage(g.sprite, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int){
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("ASCII-VOID")

	game := NewGame()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
// DISPLAY=:1 go run main.go



