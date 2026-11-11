package main

import (
	"log"
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	screenWidth = 320
	screenHeight = 240
	spriteSize = 16
)

type Game struct{
	sprite *ebiten.Image
}
func NewGame() *Game {
	img := ebiten.NewImage(spriteSize, spriteSize)
	img.Fill(color.White)
	return &Game{
		sprite: img,
	}
}
func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20,24,40,225})
	x:= float64(screenWidth)/2 - float64(spriteSize)/2
	y := float64(screenHeight)/2 - float64(spriteSize)/2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x,y)
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
