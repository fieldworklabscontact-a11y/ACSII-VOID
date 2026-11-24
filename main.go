package main

import (
	"image/color"
	"log"

	"github.com/fieldworklabscontact-a11y/ACSII-VOID/boss"
	"github.com/fieldworklabscontact-a11y/ACSII-VOID/feedback"
	"github.com/fieldworklabscontact-a11y/ACSII-VOID/physics"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	baseHeight  = 240
	startWidth  = 610
	minWidth    = 420
	maxWidth    = 900
	spriteSize  = 16
	updateDelta = 1.0 / 60.0
	hitRadius   = 6
	playerHP    = 5
)

type Game struct {
	sprite *ebiten.Image
	engine *physics.ShipEngine
	boss   *boss.Boss
	fb     *feedback.Feedback
	w, h   int
}

func NewGame() *Game {
	paddedSize := spriteSize + 2

	img := ebiten.NewImage(paddedSize, paddedSize)

	innerSquare := ebiten.NewImage(spriteSize, spriteSize)
	innerSquare.Fill(color.Black)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(1, 1)
	img.DrawImage(innerSquare, op)

	g := &Game{
		sprite: img,
		fb:     feedback.New(playerHP),
		w:      startWidth,
		h:      baseHeight,
	}
	g.reset()
	return g
}

func (g *Game) reset() {
	g.engine = physics.NewShipEngine(250.0)
	g.engine.X = float64(g.w)/2 - spriteSize/2
	g.engine.Y = float64(g.h)/2 - spriteSize/2
	g.boss = boss.New(g.w, g.h)
}

func (g *Game) center() (float64, float64) {
	return g.engine.X + spriteSize/2, g.engine.Y + spriteSize/2
}

func (g *Game) handleInput() {
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.engine.Left.HandlePress()
	} else {
		g.engine.Left.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.engine.Right.HandlePress()
	} else {
		g.engine.Right.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.engine.Up.HandlePress()
	} else {
		g.engine.Up.HandleRelease()
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.engine.Down.HandlePress()
	} else {
		g.engine.Down.HandleRelease()
	}
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	g.fb.Update(updateDelta)
	if g.fb.Respawned() {
		g.reset()
	}
	if g.fb.Frozen() {
		return nil
	}

	dt := updateDelta * g.fb.TimeScale()
	cx, cy := g.center()
	g.boss.Update(dt, cx, cy)

	if g.fb.Dead() {
		return nil
	}

	if !g.fb.Invulnerable() && g.boss.Hit(cx, cy, hitRadius) {
		if g.fb.Hurt(cx, cy) {
			g.boss.PlayerHit()
		}
	}

	g.handleInput()
	g.engine.UpdateSystem(dt)

	g.engine.X = clamp(g.engine.X, 0, float64(g.w-spriteSize))
	g.engine.Y = clamp(g.engine.Y, 0, float64(g.h-spriteSize))

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{255, 255, 255, 1})

	if g.fb.PlayerVisible() {
		op := &ebiten.DrawImageOptions{}
		op.Filter = ebiten.FilterLinear
		op.GeoM.Translate(-float64(spriteSize)/2, -float64(spriteSize)/2)
		op.GeoM.Rotate(g.engine.Heading)
		op.GeoM.Translate(float64(spriteSize)/2, float64(spriteSize)/2)
		op.GeoM.Translate(g.engine.X, g.engine.Y)
		screen.DrawImage(g.sprite, op)
	}

	g.boss.Draw(screen)
	g.fb.Draw(screen)
	g.fb.Post(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideWidth > 0 && outsideHeight > 0 {
		w := int(float64(baseHeight)*float64(outsideWidth)/float64(outsideHeight) + 0.5)
		g.w = int(clamp(float64(w), minWidth, maxWidth))
		g.h = baseHeight
		g.boss.Resize(g.w, g.h)
	}
	return g.w, g.h
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func main() {
	ebiten.SetWindowSize(startWidth*2, baseHeight*2)
	ebiten.SetWindowSizeLimits(minWidth, baseHeight, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("ASCII-VOID")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// DISPLAY=:1 go run main.go
