
package main

import (
	"image/color"
	"log"

	"github.com/fieldworklabscontact-a11y/ACSII-VOID/boss"
	"github.com/fieldworklabscontact-a11y/ACSII-VOID/physics"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth  = 610
	screenHeight = 240
	spriteSize   = 16
	updateDelta  = 1.0 / 60.0
)

type Game struct {
	sprite *ebiten.Image
	engine *physics.ShipEngine
	boss   *boss.Boss
}

func NewGame() *Game {
	paddedSize := spriteSize + 2

	img := ebiten.NewImage(paddedSize, paddedSize)

	innerSquare := ebiten.NewImage(spriteSize, spriteSize)
	innerSquare.Fill(color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(1, 1)
	img.DrawImage(innerSquare, op)

	engine := physics.NewShipEngine(250.0)

	engine.X = float64(screenWidth)/2 - float64(paddedSize)/2
	engine.Y = float64(screenHeight)/2 - float64(paddedSize)/2

	return &Game{
		sprite: img,
		engine: engine,
		boss:   boss.New(screenWidth/2, 50, screenHeight-24),
	}
}

func (g *Game) Update() error {
	dt := updateDelta

	g.boss.Update(dt, g.engine.X, g.engine.Y)

	if g.boss.Hit(g.engine.X, g.engine.Y, 8) {
		log.Println("bitten!")
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyA) {
		g.engine.Left.HandlePress()
	} else {
		g.engine.Left.HandleRelease()
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) ||
		ebiten.IsKeyPressed(ebiten.KeyD) {
		g.engine.Right.HandlePress()
	} else {
		g.engine.Right.HandleRelease()
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) ||
		ebiten.IsKeyPressed(ebiten.KeyW) {
		g.engine.Up.HandlePress()
	} else {
		g.engine.Up.HandleRelease()
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) ||
		ebiten.IsKeyPressed(ebiten.KeyS) {
		g.engine.Down.HandlePress()
	} else {
		g.engine.Down.HandleRelease()
	}

	g.engine.UpdateSystem(dt)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{255, 255, 255, 1})

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterLinear

	op.GeoM.Translate(
		-float64(spriteSize)/2,
		-float64(spriteSize)/2,
	)

	op.GeoM.Rotate(g.engine.Heading)

	op.GeoM.Translate(
		float64(spriteSize)/2,
		float64(spriteSize)/2,
	)

	op.GeoM.Translate(g.engine.X, g.engine.Y)

	screen.DrawImage(g.sprite, op)

	g.boss.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
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
