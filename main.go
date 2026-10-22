package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	screenWidth   = 1920
	screenHeight  = 1080
	sampleRate    = 44100
	highScoreFile = "highscore.txt"
)

// --- AUDIO SYSTEM ---
var audioContext *audio.Context

func initAudio() {
	audioContext = audio.NewContext(sampleRate)
}

func playSound(freq float64, durationMs int, vol float64, slide float64) {
	if audioContext == nil {
		return
	}
	// Spin a lightweight goroutine to play and safely close the player to prevent memory leaks
	go func() {
		length := sampleRate * durationMs / 1000
		buf := make([]byte, length*4)
		phase := 0.0
		for i := 0; i < length; i++ {
			progress := float64(i) / float64(length)
			envelope := math.Pow(1.0-progress, 1.4)
			curFreq := freq + slide*progress
			phase += 2 * math.Pi * curFreq / sampleRate

			v := 0.65*math.Sin(phase) + 0.35*math.Sin(phase*2.0)
			val := int16(v * 8500 * vol * envelope)
			buf[4*i] = byte(val)
			buf[4*i+1] = byte(val >> 8)
			buf[4*i+2] = byte(val)
			buf[4*i+3] = byte(val >> 8)
		}
		player := audioContext.NewPlayerFromBytes(buf)
		player.Play()
		time.Sleep(time.Duration(durationMs+50) * time.Millisecond)
		player.Close()
	}()
}

// --- ASSETS & SPRITES ---
var (
	fontSource *text.GoTextFaceSource
	fontHUD    *text.GoTextFace
	fontTitle  *text.GoTextFace
	fontSub    *text.GoTextFace
	fontPill   *text.GoTextFace

	bgImages       []*ebiten.Image
	starSprite     *ebiten.Image
	whitePixel     *ebiten.Image
	ballSprite     *ebiten.Image
	fireBallSprite *ebiten.Image
	boltBallSprite *ebiten.Image
	glowSprite     *ebiten.Image
	laserSprite    *ebiten.Image
	mislSprite     *ebiten.Image
	eBulletSprite  *ebiten.Image
	invaderSprite  *ebiten.Image
	galaxSprite    *ebiten.Image
	brickSprites   map[int]*ebiten.Image
	pillSprites    map[PowerUpType]*ebiten.Image
)

func initAssets() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		log.Fatal(err)
	}
	fontSource = s
	fontHUD = &text.GoTextFace{Source: fontSource, Size: 22}
	fontTitle = &text.GoTextFace{Source: fontSource, Size: 64}
	fontSub = &text.GoTextFace{Source: fontSource, Size: 26}
	fontPill = &text.GoTextFace{Source: fontSource, Size: 17}

	// Tiny optimized sprite for high-volume particles and stars
	starSprite = ebiten.NewImage(2, 2)
	starSprite.Fill(color.White)
	
	whitePixel = ebiten.NewImage(1, 1)
	whitePixel.Fill(color.White)

	// Generate heavy backgrounds using ultra-fast byte slices
	bgImages = []*ebiten.Image{
		createNebulaBackground(screenWidth, screenHeight, 0),
		createNebulaBackground(screenWidth, screenHeight, 1),
		createNebulaBackground(screenWidth, screenHeight, 2),
		createNebulaBackground(screenWidth, screenHeight, 3),
	}

	glowSprite = createRadialGlow(128)
	ballSprite = createChromeOrb(24, color.RGBA{180, 215, 255, 255})
	fireBallSprite = createFireOrb(24)
	boltBallSprite = createChromeOrb(24, color.RGBA{120, 240, 255, 255})
	laserSprite = createLaserSprite(16, 40, color.RGBA{0, 230, 255, 255})
	mislSprite = createMissileSprite(12, 28)
	eBulletSprite = createLaserSprite(14, 30, color.RGBA{255, 60, 90, 255})

	invaderSprite = createInvaderShip(84, 58)
	galaxSprite = createGalaxianShip(76, 62)

	brickSprites = make(map[int]*ebiten.Image)
	brickSprites[1] = createCrystalBrick(104, 38, color.RGBA{0, 200, 255, 255})
	brickSprites[2] = createCrystalBrick(104, 38, color.RGBA{255, 140, 20, 255})
	brickSprites[3] = createCrystalBrick(104, 38, color.RGBA{255, 40, 100, 255})
	brickSprites[4] = createCrystalBrick(104, 38, color.RGBA{175, 80, 255, 255})
	brickSprites[99] = createGoldBrick(104, 38)
	brickSprites[100] = createBombBrick(104, 38)

	pillSprites = make(map[PowerUpType]*ebiten.Image)
	pillSprites[PowerMultiball] = createPillSprite(68, 30, color.RGBA{190, 60, 255, 255}, "MULTI")
	pillSprites[PowerEnlarge] = createPillSprite(68, 30, color.RGBA{40, 215, 110, 255}, "WIDE")
	pillSprites[PowerLaser] = createPillSprite(68, 30, color.RGBA{30, 150, 255, 255}, "LASER")
	pillSprites[PowerSticky] = createPillSprite(68, 30, color.RGBA{0, 230, 180, 255}, "CATCH")
	pillSprites[Power1UP] = createPillSprite(68, 30, color.RGBA{255, 205, 30, 255}, "1-UP")
	pillSprites[PowerShrink] = createPillSprite(68, 30, color.RGBA{255, 45, 55, 255}, "SHRINK")
	pillSprites[PowerSlow] = createPillSprite(68, 30, color.RGBA{70, 210, 255, 255}, "SLOW")
	pillSprites[PowerFast] = createPillSprite(68, 30, color.RGBA{255, 115, 0, 255}, "FAST")
	pillSprites[PowerFireball] = createPillSprite(68, 30, color.RGBA{255, 65, 15, 255}, "FIRE")
	pillSprites[PowerShield] = createPillSprite(68, 30, color.RGBA{0, 140, 255, 255}, "SHLD")
	pillSprites[PowerMissile] = createPillSprite(68, 30, color.RGBA{255, 220, 0, 255}, "MISL")
	pillSprites[PowerBolt] = createPillSprite(68, 30, color.RGBA{100, 255, 255, 255}, "BOLT")
	pillSprites[PowerWarp] = createPillSprite(68, 30, color.RGBA{220, 50, 255, 255}, "WARP")
}

// Memory-optimized procedural texture generator using byte slices
func createNebulaBackground(w, h, style int) *ebiten.Image {
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		for x := 0; x < w; x++ {
			fx := float64(x) / float64(w)
			var r, g, b uint8
			switch style {
			case 0:
				n := math.Sin(fx*4.5+t*3.0) * math.Cos(t*5.0-fx*2.0)
				r = uint8(math.Min(255, 8+t*12+n*25))
				g = uint8(math.Min(255, 12+t*18+n*12))
				b = uint8(math.Min(255, 30+t*35+n*50))
			case 1:
				dx := fx - 0.5
				dy := t - 0.4
				dist := math.Sqrt(dx*dx + dy*dy)
				core := math.Max(0, 1.0-dist*1.8)
				r = uint8(math.Min(255, 15+core*65+t*10))
				g = uint8(math.Min(255, 5+core*25+t*10))
				b = uint8(math.Min(255, 35+core*95+t*30))
			case 2:
				n := math.Cos(fx*6.0 - t*4.0)
				r = uint8(math.Min(255, 5+n*10))
				g = uint8(math.Min(255, 18+t*25+n*30))
				b = uint8(math.Min(255, 28+t*40+n*45))
			default:
				r = uint8(math.Min(255, 5+t*8))
				g = uint8(math.Min(255, 8+t*12))
				b = uint8(math.Min(255, 18+t*28))
			}
			idx := (y*w + x) * 4
			pixels[idx] = r
			pixels[idx+1] = g
			pixels[idx+2] = b
			pixels[idx+3] = 255
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}

func createRadialGlow(size int) *ebiten.Image {
	pixels := make([]byte, size*size*4)
	r := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := (float64(x) + 0.5 - r) / r
			dy := (float64(y) + 0.5 - r) / r
			d := math.Sqrt(dx*dx + dy*dy)
			if d < 1.0 {
				v := uint8(255 * math.Pow(1.0-d, 2.2))
				idx := (y*size + x) * 4
				pixels[idx] = v
				pixels[idx+1] = v
				pixels[idx+2] = v
				pixels[idx+3] = v
			}
		}
	}
	img := ebiten.NewImage(size, size)
	img.WritePixels(pixels)
	return img
}

func createChromeOrb(size int, tint color.RGBA) *ebiten.Image {
	pixels := make([]byte, size*size*4)
	r := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - r
			dy := float64(y) + 0.5 - r
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				edgeAlpha := math.Min(1.0, r-dist+0.5)
				nx, ny := dx/r, dy/r
				nz := math.Sqrt(math.Max(0, 1.0-nx*nx-ny*ny))

				dot := math.Max(0.15, -0.45*nx-0.55*ny+0.70*nz)
				spec := math.Pow(math.Max(0, -0.35*nx-0.45*ny+0.82*nz), 16) * 230
				rim := math.Pow(1.0-nz, 3.0) * 130

				idx := (y*size + x) * 4
				pixels[idx] = uint8(math.Min(255, float64(tint.R)*dot+spec+rim*0.3) * edgeAlpha)
				pixels[idx+1] = uint8(math.Min(255, float64(tint.G)*dot+spec+rim*0.7) * edgeAlpha)
				pixels[idx+2] = uint8(math.Min(255, float64(tint.B)*dot+spec+rim) * edgeAlpha)
				pixels[idx+3] = uint8(255 * edgeAlpha)
			}
		}
	}
	img := ebiten.NewImage(size, size)
	img.WritePixels(pixels)
	return img
}

func createFireOrb(size int) *ebiten.Image {
	pixels := make([]byte, size*size*4)
	r := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - r
			dy := float64(y) + 0.5 - r
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				edgeAlpha := math.Min(1.0, r-dist+0.5)
				core := math.Pow(math.Max(0, 1.0-(dist/r)), 1.4)
				
				idx := (y*size + x) * 4
				pixels[idx] = uint8(255 * edgeAlpha)
				pixels[idx+1] = uint8(math.Min(255, 65+core*190) * edgeAlpha)
				pixels[idx+2] = uint8(math.Min(255, core*150) * edgeAlpha)
				pixels[idx+3] = uint8(255 * edgeAlpha)
			}
		}
	}
	img := ebiten.NewImage(size, size)
	img.WritePixels(pixels)
	return img
}

func createCrystalBrick(w, h int, base color.RGBA) *ebiten.Image {
	pixels := make([]byte, w*h*4)
	bevel := 7
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+y < 4 || (w-1-x)+y < 4 || x+(h-1-y) < 4 || (w-1-x)+(h-1-y) < 4 {
				continue
			}
			dTop, dBot, dLeft, dRight := y, h-1-y, x, w-1-x
			minD := dTop
			if dBot < minD { minD = dBot }
			if dLeft < minD { minD = dLeft }
			if dRight < minD { minD = dRight }

			mult := 1.0
			spec := 0.0
			if minD < bevel {
				if minD == dTop { mult = 1.38; spec = 45 } else if minD == dLeft { mult = 1.18; spec = 15 } else if minD == dRight { mult = 0.72 } else { mult = 0.52 }
			} else {
				cx := (float64(x) - float64(w)*0.5) / (float64(w) * 0.5)
				cy := (float64(y) - float64(h)*0.5) / (float64(h) * 0.5)
				coreDist := math.Sqrt(cx*cx + cy*cy)
				mult = 0.85 + 0.40*math.Max(0, 1.0-coreDist)
				if y < h/2 { spec = 22 * (1.0 - float64(y)/float64(h/2)) }
			}
			if minD == 0 || minD == bevel { mult *= 1.25; spec += 30 }

			idx := (y*w + x) * 4
			pixels[idx] = uint8(math.Min(255, float64(base.R)*mult+spec))
			pixels[idx+1] = uint8(math.Min(255, float64(base.G)*mult+spec))
			pixels[idx+2] = uint8(math.Min(255, float64(base.B)*mult+spec))
			pixels[idx+3] = 255
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}

func createGoldBrick(w, h int) *ebiten.Image {
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+y < 4 || (w-1-x)+y < 4 || x+(h-1-y) < 4 || (w-1-x)+(h-1-y) < 4 { continue }
			mult := 1.0
			if y < h/2 { mult = 1.3 } else { mult = 0.75 }
			
			idx := (y*w + x) * 4
			pixels[idx] = uint8(math.Min(255, 230.0*mult))
			pixels[idx+1] = uint8(math.Min(255, 185.0*mult))
			pixels[idx+2] = uint8(math.Min(255, 40.0*mult))
			pixels[idx+3] = 255
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}

func createBombBrick(w, h int) *ebiten.Image {
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+y < 4 || (w-1-x)+y < 4 || x+(h-1-y) < 4 || (w-1-x)+(h-1-y) < 4 { continue }
			mult := 0.8 + 0.3*math.Sin(float64(x)*0.2)
			idx := (y*w + x) * 4
			pixels[idx] = uint8(math.Min(255, 255.0*mult))
			pixels[idx+1] = uint8(math.Min(255, 30.0*mult))
			pixels[idx+2] = uint8(math.Min(255, 50.0*mult))
			pixels[idx+3] = 255
		}
	}
	img := ebiten.NewImage(w, h)
	img.WritePixels(pixels)
	return img
}

// Vector-based single-draw assets are fine for startup
func createInvaderShip(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	cx := float32(w) / 2
	vector.DrawFilledRect(img, 6, 18, float32(w-12), 24, color.RGBA{45, 210, 120, 255}, true)
	vector.DrawFilledRect(img, 16, 8, float32(w-32), 34, color.RGBA{65, 240, 145, 255}, true)
	vector.DrawFilledCircle(img, cx, 22, 22, color.RGBA{90, 255, 175, 255}, true)
	vector.DrawFilledCircle(img, cx, 18, 14, color.RGBA{190, 255, 225, 255}, true)
	vector.DrawFilledCircle(img, cx-12, 26, 5, color.RGBA{255, 40, 70, 255}, true)
	vector.DrawFilledCircle(img, cx+12, 26, 5, color.RGBA{255, 40, 70, 255}, true)
	return img
}

func createGalaxianShip(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	cx := float32(w) / 2
	var path vector.Path
	path.MoveTo(cx, 4); path.LineTo(float32(w-4), 38); path.LineTo(float32(w-14), 56)
	path.LineTo(cx, 42); path.LineTo(14, 56); path.LineTo(4, 38); path.Close()
	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = 1.0; vs[i].ColorG = 0.22; vs[i].ColorB = 0.45; vs[i].ColorA = 1.0
	}
	img.DrawTriangles(vs, is, whitePixel, &ebiten.DrawTrianglesOptions{AntiAlias: true})
	vector.DrawFilledRect(img, cx-10, 14, 20, 34, color.RGBA{255, 200, 40, 255}, true)
	vector.DrawFilledCircle(img, cx, 28, 8, color.RGBA{0, 240, 255, 255}, true)
	return img
}

func createPillSprite(w, h int, base color.RGBA, label string) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	r := float32(h) / 2.0
	rim := color.RGBA{220, 235, 255, 255}
	vector.DrawFilledCircle(img, r, r, r, rim, true)
	vector.DrawFilledCircle(img, float32(w)-r, r, r, rim, true)
	vector.DrawFilledRect(img, r, 0, float32(w)-2*r, float32(h), rim, true)
	inR := r - 2
	vector.DrawFilledCircle(img, r, r, inR, base, true)
	vector.DrawFilledCircle(img, float32(w)-r, r, inR, base, true)
	vector.DrawFilledRect(img, r, 2, float32(w)-2*r, float32(h)-4, base, true)
	highlight := color.RGBA{255, 255, 255, 110}
	vector.DrawFilledRect(img, r, 4, float32(w)-2*r, r*0.55, highlight, true)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(w)/2, float64(h)/2)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(img, label, fontPill, op)
	return img
}

func createLaserSprite(w, h int, c color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	vector.DrawFilledRect(img, 2, 2, float32(w-4), float32(h-4), color.RGBA{c.R, c.G, c.B, 120}, true)
	vector.DrawFilledRect(img, 5, 0, float32(w-10), float32(h), color.RGBA{240, 255, 255, 255}, true)
	return img
}

func createMissileSprite(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	vector.DrawFilledRect(img, 2, 0, float32(w-4), float32(h), color.RGBA{255, 220, 0, 255}, true)
	vector.DrawFilledRect(img, 0, float32(h-8), float32(w), 8, color.RGBA{255, 80, 0, 255}, true)
	return img
}

// Optimized Rectangle drawer replacing heavy vector rects where appropriate
func drawRect(dst *ebiten.Image, x, y, w, h float32, clr color.Color) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(whitePixel, op)
}

// --- STRUCTS & GAME STATE ---
type GameState int

const (
	StateStart GameState = iota
	StateServe
	StatePlay
	StatePaused
	StateGameOver
)

type PowerUpType int

const (
	PowerMultiball PowerUpType = iota
	PowerEnlarge
	PowerLaser
	PowerSticky
	Power1UP
	PowerShrink
	PowerSlow
	PowerFast
	PowerFireball
	PowerShield
	PowerMissile
	PowerBolt
	PowerWarp
)

type EnemyType int

const (
	EnemyInvader EnemyType = iota
	EnemyGalaxian
	EnemyMothership
)

type Enemy struct {
	x, y, w, h     float32
	homeX, homeY   float32
	vx, vy         float32
	hp, maxHP      int
	eType          EnemyType
	diving         bool
	divePhase      float64
	active         bool
	flash          int
	invulnCooldown int
}

type EnemyBullet struct {
	x, y, w, h, vx, vy float32
	active             bool
}

type Paddle struct {
	x, y, w, h   float32
	vx           float32
	speed        float32
	gunLevel     int
	sticky       bool
	dashCooldown int
}

type Ball struct {
	x, y, dx, dy, size float32
	active             bool
	stuck              bool
	stuckOffset        float32
	fireball           bool
	bolt               bool
}

type Brick struct {
	x, y, w, h float32
	active     bool
	hp, maxHP  int
	flash      int
	isGold     bool
	isBomb     bool
	vx         float32
	minX, maxX float32
}

type PowerUp struct {
	x, y, w, h, dy float32
	pType          PowerUpType
	active         bool
}

type Laser struct {
	x, y, w, h, dy float32
	active         bool
}

type Missile struct {
	x, y, w, h, dx, dy float32
	active             bool
}

type Particle struct {
	x, y, dx, dy, size float32
	life, max          int
	color              color.RGBA
}

type ScorePopup struct {
	x, y  float32
	text  string
	life  int
	color color.RGBA
}

type Star struct {
	x, y, speed, size float32
	color             color.RGBA
}

type Portal struct {
	y, h     float32
	active   bool
	isLeft   bool
}

type Game struct {
	state          GameState
	level          int
	score          int
	lives          int
	highScore      int
	shake          int
	tick           int
	lastMouseX     int
	combo          int
	comboTimer     int
	ballSpeedMult  float32
	fireballActive bool
	boltActive     bool
	shieldActive   bool
	missileActive  bool
	missileTimer   int
	offscreen      *ebiten.Image
	paddle         Paddle
	balls          []Ball
	bricks         []Brick
	enemies        []Enemy
	enemyBullets   []EnemyBullet
	enemyWaveSpawn bool
	invaderDir     float32
	waveBanner     string
	waveBannerTime int
	powerUps       []PowerUp
	lasers         []Laser
	missiles       []Missile
	particles      []Particle
	popups         []ScorePopup
	stars          []Star
	leftPortal     Portal
	rightPortal    Portal
	laserCooldown  int
}

var brickParticleColors = map[int]color.RGBA{
	1:   {0, 200, 255, 255},
	2:   {255, 140, 20, 255},
	3:   {255, 40, 100, 255},
	4:   {175, 80, 255, 255},
	99:  {230, 185, 40, 255},
	100: {255, 50, 50, 255},
}

func (g *Game) loadHighScore() {
	if data, err := os.ReadFile(highScoreFile); err == nil {
		if hs, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			g.highScore = hs
		}
	}
}

func (g *Game) saveHighScore() {
	if g.score > g.highScore {
		g.highScore = g.score
		_ = os.WriteFile(highScoreFile, []byte(fmt.Sprintf("%d", g.highScore)), 0644)
	}
}

func (g *Game) initGame() {
	g.score = 0
	g.lives = 3
	g.ballSpeedMult = 1.0
	g.fireballActive = false
	g.boltActive = false
	g.shieldActive = false
	g.missileActive = false
	g.combo = 0
	g.comboTimer = 0
	if g.offscreen == nil {
		g.offscreen = ebiten.NewImage(screenWidth, screenHeight)
	}
	mx, _ := ebiten.CursorPosition()
	g.lastMouseX = mx

	g.stars = make([]Star, 140)
	starTints := []color.RGBA{
		{255, 255, 255, 220},
		{150, 220, 255, 200},
		{255, 210, 170, 180},
		{200, 160, 255, 190},
	}
	for i := range g.stars {
		depth := rand.Float32()
		c := starTints[rand.Intn(len(starTints))]
		c.A = uint8(60 + depth*170)
		g.stars[i] = Star{
			x:     rand.Float32() * screenWidth,
			y:     rand.Float32() * screenHeight,
			speed: 0.3 + depth*2.2,
			size:  0.8 + depth*1.4,
			color: c,
		}
	}

	g.paddle.gunLevel = 0
	g.paddle.sticky = false
	g.generateLevel()
	g.resetServe()
	g.state = StateStart
}

func (g *Game) resetServe() {
	g.ballSpeedMult = 1.0
	g.paddle = Paddle{
		x:        screenWidth/2 - 110,
		y:        screenHeight - 82,
		w:        220,
		h:        26,
		speed:    17,
		gunLevel: g.paddle.gunLevel,
		sticky:   g.paddle.sticky,
	}
	g.balls = []Ball{{
		x:           screenWidth/2 - 12,
		y:           screenHeight - 110,
		dx:          4.5,
		dy:          -10.5,
		size:        24,
		active:      true,
		stuck:       true,
		stuckOffset: 0,
		fireball:    g.fireballActive,
		bolt:        g.boltActive,
	}}
	g.powerUps = nil
	g.lasers = nil
	g.missiles = nil
	g.enemyBullets = nil
	g.state = StateServe
}

func (g *Game) applyBallSpeedMultiplier() {
	targetSpeed := float32(12.0) * g.ballSpeedMult
	for i := range g.balls {
		b := &g.balls[i]
		if !b.active || b.stuck {
			continue
		}
		cur := float32(math.Sqrt(float64(b.dx*b.dx + b.dy*b.dy)))
		if cur > 0.01 {
			b.dx = (b.dx / cur) * targetSpeed
			b.dy = (b.dy / cur) * targetSpeed
		}
	}
}

func (g *Game) generateLevel() {
	g.bricks = nil
	g.enemies = nil
	g.enemyBullets = nil
	g.enemyWaveSpawn = false
	g.leftPortal.active = false
	g.rightPortal.active = false

	cols := 15
	bWidth := float32(104)
	bHeight := float32(38)
	padX := float32(10)
	padY := float32(10)
	startX := (screenWidth - (float32(cols)*(bWidth+padX) - padX)) / 2
	startY := float32(115)

	rows := 4 + (g.level / 2)
	if rows > 8 { rows = 8 }

	pattern := g.level % 3

	for row := 0; row < rows; row++ {
		for col := 0; col <= cols/2; col++ {
			skip := false
			if pattern == 1 && (row+col)%3 == 0 { skip = true } else if pattern == 2 && col == row%4 { skip = true } else if rand.Float32() < 0.12 { skip = true }
			if skip { continue }

			hp := 1
			isGold, isBomb := false, false

			if row < 2 && g.level >= 2 { hp = 2 }
			if row == 0 && g.level >= 3 { hp = 3 }
			if g.level >= 2 && rand.Float32() < 0.12 {
				hp = 99; isGold = true
			} else if g.level >= 3 && rand.Float32() < 0.15 {
				hp = 1; isBomb = true
			}

			xLeft := startX + float32(col)*(bWidth+padX)
			yPos := startY + float32(row)*(bHeight+padY)

			var vx float32 = 0
			if g.level >= 4 && row == 1 && rand.Float32() < 0.5 { vx = 0.8 }

			g.bricks = append(g.bricks, Brick{
				x: xLeft, y: yPos, w: bWidth, h: bHeight, active: true, hp: hp, maxHP: hp,
				isGold: isGold, isBomb: isBomb, vx: vx, minX: xLeft - 60, maxX: xLeft + 60,
			})

			mirrorCol := cols - 1 - col
			if mirrorCol != col {
				xRight := startX + float32(mirrorCol)*(bWidth+padX)
				g.bricks = append(g.bricks, Brick{
					x: xRight, y: yPos, w: bWidth, h: bHeight, active: true, hp: hp, maxHP: hp,
					isGold: isGold, isBomb: isBomb, vx: -vx, minX: xRight - 60, maxX: xRight + 60,
				})
			}
		}
	}

	if g.level >= 2 {
		g.leftPortal = Portal{y: 400, h: 180, active: true, isLeft: true}
		g.rightPortal = Portal{y: 400, h: 180, active: true, isLeft: false}
	}

	if g.level == 5 {
		g.waveBanner = "WARNING: GIANT DOH MOTHERSHIP BOSS APPROACHING!"
		g.waveBannerTime = 200
	} else if g.level == 3 || (g.level > 3 && g.level%2 == 1) {
		g.waveBanner = fmt.Sprintf("SECTOR %d  —  ALIEN SQUADRON BEHIND BRICKS!", g.level)
		g.waveBannerTime = 160
	} else {
		g.waveBanner = fmt.Sprintf("SECTOR %d", g.level)
		g.waveBannerTime = 110
	}
}

func (g *Game) spawnAlienWave() {
	g.enemyWaveSpawn = true
	g.enemies = nil
	g.invaderDir = 1.0

	if g.level == 5 {
		g.waveBanner = "BOSS FIGHT: MOTHERSHIP DOH ARMED & ACTIVE!"
		g.waveBannerTime = 240
		playSound(140, 600, 0.9, 150)

		bossX := float32(screenWidth/2 - 220)
		bossY := float32(110)
		g.enemies = append(g.enemies, Enemy{
			x: bossX, y: bossY, w: 440, h: 180,
			homeX: bossX, homeY: bossY,
			hp: 35, maxHP: 35, eType: EnemyMothership, active: true,
		})
	} else if g.level%2 == 1 {
		g.waveBanner = "WARNING: SPACE INVADER ARMADA ATTACKING!"
		g.waveBannerTime = 210
		playSound(180, 500, 0.8, 120)

		cols := 6
		rows := 2
		startX := float32(screenWidth-cols*150) / 2
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				hx := startX + float32(c*150)
				hy := float32(130 + r*95)
				g.enemies = append(g.enemies, Enemy{
					x: hx, y: -80 - float32(r*70), w: 84, h: 58,
					homeX: hx, homeY: hy,
					hp: 5, maxHP: 5, eType: EnemyInvader, active: true,
				})
			}
		}
	} else {
		g.waveBanner = "DANGER: GALAXIAN DIVE-BOMBER SWARM ENGAGED!"
		g.waveBannerTime = 210
		playSound(240, 550, 0.8, 260)

		count := 8
		startX := float32(screenWidth-count*140) / 2
		for i := 0; i < count; i++ {
			hx := startX + float32(i*140)
			hy := float32(150 + (i%2)*80)
			g.enemies = append(g.enemies, Enemy{
				x: hx, y: -90, w: 76, h: 62,
				homeX: hx, homeY: hy,
				hp: 5, maxHP: 5, eType: EnemyGalaxian,
				divePhase: float64(i) * 0.9, active: true,
			})
		}
	}
}

// --- UPDATE LOGIC ---
func (g *Game) Update() error {
	g.tick++

	if inpututil.IsKeyJustPressed(ebiten.KeyP) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.state == StatePlay { g.state = StatePaused; return nil
		} else if g.state == StatePaused { g.state = StatePlay; return nil }
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF) { ebiten.SetFullscreen(!ebiten.IsFullscreen()) }

	if g.state == StateStart {
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && g.level > 1 { g.level-- }
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) && g.level < 10 { g.level++ }
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyN) && (g.state == StatePlay || g.state == StateServe) {
		for i := range g.bricks { g.bricks[i].active = false }
	}

	for i := range g.stars {
		g.stars[i].y += g.stars[i].speed
		if g.stars[i].y > screenHeight { g.stars[i].y = 0; g.stars[i].x = rand.Float32() * screenWidth }
	}

	if g.shake > 0 { g.shake-- }
	if g.waveBannerTime > 0 { g.waveBannerTime-- }
	if g.comboTimer > 0 {
		g.comboTimer--
		if g.comboTimer == 0 { g.combo = 0 }
	}
	if g.missileTimer > 0 {
		g.missileTimer--
		if g.missileTimer == 0 { g.missileActive = false }
	}

	for i := range g.bricks {
		if g.bricks[i].flash > 0 { g.bricks[i].flash-- }
		if g.bricks[i].vx != 0 && g.bricks[i].active {
			g.bricks[i].x += g.bricks[i].vx
			if g.bricks[i].x < g.bricks[i].minX || g.bricks[i].x > g.bricks[i].maxX {
				g.bricks[i].vx *= -1
			}
		}
	}

	switch g.state {
	case StateStart, StateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if g.state == StateGameOver { g.initGame() }
			g.state = StateServe
		}
	case StateServe:
		g.movePaddle()
		g.balls[0].x = g.paddle.x + g.paddle.w/2 - g.balls[0].size/2
		g.balls[0].y = g.paddle.y - g.balls[0].size - 4
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.balls[0].stuck = false
			spd := float32(12.0) * g.ballSpeedMult
			g.balls[0].dx = spd * 0.35
			if rand.Intn(2) == 0 { g.balls[0].dx = -g.balls[0].dx }
			g.balls[0].dy = -spd * 0.93
			g.state = StatePlay
			playSound(520, 70, 0.5, 180)
		}
	case StatePaused:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) { g.state = StatePlay }
	case StatePlay:
		g.updatePlay()
	}

	for i := len(g.particles) - 1; i >= 0; i-- {
		p := &g.particles[i]
		p.x += p.dx; p.y += p.dy; p.dy += 0.18; p.life--
		if p.life <= 0 { g.particles = append(g.particles[:i], g.particles[i+1:]...) }
	}

	for i := len(g.popups) - 1; i >= 0; i-- {
		po := &g.popups[i]
		po.y -= 1.2; po.life--
		if po.life <= 0 { g.popups = append(g.popups[:i], g.popups[i+1:]...) }
	}

	return nil
}

func (g *Game) movePaddle() {
	mx, _ := ebiten.CursorPosition()
	dashPressed := inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || inpututil.IsKeyJustPressed(ebiten.KeyShiftRight) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if dashPressed && g.paddle.dashCooldown == 0 {
		g.paddle.dashCooldown = 45
		if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			g.paddle.x -= 140
			playSound(750, 80, 0.6, -200)
		} else {
			g.paddle.x += 140
			playSound(750, 80, 0.6, 200)
		}
	}
	if g.paddle.dashCooldown > 0 { g.paddle.dashCooldown-- }

	if mx != g.lastMouseX {
		targetX := float32(mx) - g.paddle.w/2
		g.paddle.vx = (targetX - g.paddle.x) * 0.45
		g.paddle.x += g.paddle.vx
		g.lastMouseX = mx
	} else {
		accel := float32(2.6)
		if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			g.paddle.vx -= accel
		} else if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
			g.paddle.vx += accel
		} else {
			g.paddle.vx *= 0.78
		}
		if g.paddle.vx > g.paddle.speed { g.paddle.vx = g.paddle.speed }
		if g.paddle.vx < -g.paddle.speed { g.paddle.vx = -g.paddle.speed }
		g.paddle.x += g.paddle.vx
	}

	if g.paddle.x < 24 { g.paddle.x = 24; g.paddle.vx = 0 }
	if g.paddle.x+g.paddle.w > screenWidth-24 { g.paddle.x = screenWidth - 24 - g.paddle.w; g.paddle.vx = 0 }
}

func (g *Game) getGunPositions() []float32 {
	px, pw := g.paddle.x, g.paddle.w
	switch g.paddle.gunLevel {
	case 1: return []float32{px + pw/2}
	case 2: return []float32{px + 20, px + pw - 20}
	case 3: return []float32{px + 20, px + pw/2, px + pw - 20}
	case 4: return []float32{px + 14, px + pw*0.35, px + pw*0.65, px + pw - 14}
	default: return nil
	}
}

func (g *Game) updatePlay() {
	g.movePaddle()

	triggerPressed := ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justTriggered := inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	if justTriggered {
		released := false
		for i := range g.balls {
			if g.balls[i].active && g.balls[i].stuck {
				g.balls[i].stuck = false
				hitOffset := g.balls[i].stuckOffset
				normalized := hitOffset / (g.paddle.w / 2)
				angle := float64(normalized) * (math.Pi / 3.2)
				spd := float32(12.0) * g.ballSpeedMult
				g.balls[i].dx = spd * float32(math.Sin(angle))
				g.balls[i].dy = -spd * float32(math.Cos(angle))
				released = true
			}
		}
		if released { playSound(560, 70, 0.5, 160) }
	}

	if g.paddle.gunLevel > 0 {
		if g.laserCooldown > 0 { g.laserCooldown-- }
		if triggerPressed && g.laserCooldown == 0 {
			playSound(880, 45, 0.3, -300)
			for _, gx := range g.getGunPositions() {
				g.lasers = append(g.lasers, Laser{x: gx - 8, y: g.paddle.y - 26, w: 16, h: 40, dy: -22, active: true})
			}
			g.laserCooldown = 14
		}
	}

	if g.missileActive && g.tick%35 == 0 {
		g.missiles = append(g.missiles, Missile{
			x: g.paddle.x + g.paddle.w/2 - 6, y: g.paddle.y - 20, w: 12, h: 28, dx: 0, dy: -14, active: true,
		})
		playSound(720, 90, 0.4, 150)
	}

	for i := range g.lasers {
		l := &g.lasers[i]
		if !l.active { continue }
		l.y += l.dy
		if l.y < -40 { l.active = false; continue }
		for j := range g.bricks {
			if g.bricks[j].active && checkCol(l.x, l.y, l.w, l.h, g.bricks[j].x, g.bricks[j].y, g.bricks[j].w, g.bricks[j].h) {
				l.active = false; g.hitBrick(j, 1, false); break
			}
		}
		if !l.active { continue }
		for j := range g.enemies {
			e := &g.enemies[j]
			if e.active && checkCol(l.x, l.y, l.w, l.h, e.x, e.y, e.w, e.h) {
				l.active = false; g.damageEnemy(j, 1); break
			}
		}
	}

	for i := range g.missiles {
		m := &g.missiles[i]
		if !m.active { continue }
		m.x += m.dx; m.y += m.dy
		if m.y < -40 { m.active = false; continue }

		var targetX, targetY float32 = screenWidth / 2, 100
		foundTarget := false
		for _, e := range g.enemies {
			if e.active { targetX = e.x + e.w/2; targetY = e.y + e.h/2; foundTarget = true; break }
		}
		if !foundTarget {
			for _, br := range g.bricks {
				if br.active { targetX = br.x + br.w/2; targetY = br.y + br.h/2; foundTarget = true; break }
			}
		}
		if foundTarget {
			dx := targetX - m.x
			dy := targetY - m.y
			dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
			if dist > 1.0 {
				m.dx = (dx / dist) * 8.0
				m.dy = (dy / dist) * 8.0
			}
		}

		for j := range g.bricks {
			if g.bricks[j].active && checkCol(m.x, m.y, m.w, m.h, g.bricks[j].x, g.bricks[j].y, g.bricks[j].w, g.bricks[j].h) {
				m.active = false; g.hitBrick(j, 2, false); break
			}
		}
		for j := range g.enemies {
			e := &g.enemies[j]
			if e.active && checkCol(m.x, m.y, m.w, m.h, e.x, e.y, e.w, e.h) {
				m.active = false; g.damageEnemy(j, 3); break
			}
		}
	}

	g.updateEnemies()

	activeBalls := 0
	const subSteps = 4
	for i := range g.balls {
		b := &g.balls[i]
		if !b.active { continue }
		activeBalls++

		if b.stuck {
			b.x = g.paddle.x + g.paddle.w/2 + b.stuckOffset - b.size/2
			b.y = g.paddle.y - b.size - 3
			continue
		}

		if g.level >= 6 {
			gx, gy := float32(screenWidth/2), float32(350)
			distX := gx - b.x
			distY := gy - b.y
			d := math.Sqrt(float64(distX*distX + distY*distY))
			if d < 400 && d > 20 {
				b.dx += float32(float64(distX)/d) * 0.04
				b.dy += float32(float64(distY)/d) * 0.04
			}
		}

		for step := 0; step < subSteps; step++ {
			b.x += b.dx / subSteps
			b.y += b.dy / subSteps

			if g.leftPortal.active && b.x <= 10 && b.y > g.leftPortal.y && b.y < g.leftPortal.y+g.leftPortal.h {
				b.x = screenWidth - 40; b.y = g.rightPortal.y + g.rightPortal.h/2
				playSound(650, 100, 0.5, 300)
			} else if g.rightPortal.active && b.x+b.size >= screenWidth-10 && b.y > g.rightPortal.y && b.y < g.rightPortal.y+g.rightPortal.h {
				b.x = 40; b.y = g.leftPortal.y + g.leftPortal.h/2
				playSound(650, 100, 0.5, 300)
			}

			if b.x <= 0 {
				b.x = 0; b.dx = float32(math.Abs(float64(b.dx)))
				playSound(280, 25, 0.4, 0)
			} else if b.x+b.size >= screenWidth {
				b.x = screenWidth - b.size; b.dx = -float32(math.Abs(float64(b.dx)))
				playSound(280, 25, 0.4, 0)
			}
			if b.y <= 60 {
				b.y = 60; b.dy = float32(math.Abs(float64(b.dy)))
				playSound(280, 25, 0.4, 0)
			}

			if b.y > screenHeight {
				if g.shieldActive {
					g.shieldActive = false
					b.dy = -float32(math.Abs(float64(b.dy)))
					b.y = screenHeight - 40
					playSound(720, 250, 0.8, -200)
					g.shake = 12
				} else {
					b.active = false
					break
				}
			}

			if b.dy > 0 && checkCol(b.x, b.y, b.size, b.size, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
				hitOffset := (b.x + b.size/2) - (g.paddle.x + g.paddle.w/2)
				maxOff := g.paddle.w/2 - 12
				if hitOffset < -maxOff { hitOffset = -maxOff }
				if hitOffset > maxOff { hitOffset = maxOff }

				if g.paddle.sticky {
					b.stuck = true; b.stuckOffset = hitOffset
					b.y = g.paddle.y - b.size - 3
					playSound(460, 60, 0.5, 220)
					break
				}

				b.y = g.paddle.y - b.size
				normalized := hitOffset / (g.paddle.w / 2)
				speed := float32(12.0) * g.ballSpeedMult
				angle := float64(normalized) * (math.Pi / 3.0)
				b.dx = speed * float32(math.Sin(angle))
				b.dy = -speed * float32(math.Cos(angle))
				playSound(380, 40, 0.5, 80)
				break
			}

			for j := range g.bricks {
				br := &g.bricks[j]
				if !br.active { continue }
				if checkCol(b.x, b.y, b.size, b.size, br.x, br.y, br.w, br.h) {
					if br.isGold {
						bounceBox(b, br.x, br.y, br.w, br.h)
						playSound(310, 30, 0.4, 40)
						break
					}
					if b.fireball {
						if br.hp <= 1 {
							g.hitBrick(j, 1, b.bolt)
						} else {
							bounceBox(b, br.x, br.y, br.w, br.h)
							g.hitBrick(j, 2, b.bolt)
						}
					} else {
						bounceBox(b, br.x, br.y, br.w, br.h)
						g.hitBrick(j, 1, b.bolt)
					}
					break
				}
			}

			for j := range g.enemies {
				e := &g.enemies[j]
				if !e.active || e.invulnCooldown > 0 { continue }
				if checkCol(b.x, b.y, b.size, b.size, e.x, e.y, e.w, e.h) {
					bounceBox(b, e.x, e.y, e.w, e.h)
					e.invulnCooldown = 8
					dmg := 1
					if b.fireball { dmg = 2 }
					g.damageEnemy(j, dmg)
					break
				}
			}
		}
	}

	if activeBalls == 0 {
		g.loseLife()
	}

	for i := range g.powerUps {
		p := &g.powerUps[i]
		if !p.active { continue }
		p.y += p.dy
		if p.y > screenHeight { p.active = false; continue }

		if checkCol(p.x, p.y, p.w, p.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			p.active = false
			playSound(640, 150, 0.6, 250)
			g.score += 150
			switch p.pType {
			case PowerMultiball:
				spd := float32(12.0) * g.ballSpeedMult
				for x := 0; x < len(g.balls); x++ {
					if g.balls[x].active {
						orig := g.balls[x]
						g.balls = append(g.balls,
							Ball{x: orig.x, y: orig.y, dx: -spd * 0.6, dy: -spd * 0.8, size: orig.size, active: true, fireball: g.fireballActive, bolt: g.boltActive},
							Ball{x: orig.x, y: orig.y, dx: spd * 0.6, dy: -spd * 0.8, size: orig.size, active: true, fireball: g.fireballActive, bolt: g.boltActive},
						)
						break
					}
				}
			case PowerEnlarge: g.paddle.w = 310
			case PowerLaser: if g.paddle.gunLevel < 4 { g.paddle.gunLevel++ }
			case PowerSticky: g.paddle.sticky = true
			case Power1UP: g.lives++
			case PowerShrink:
				playSound(190, 300, 0.7, -90)
				g.paddle.w = 130; g.paddle.gunLevel = 0; g.paddle.sticky = false
				g.fireballActive = false; g.boltActive = false; g.shake = 12
				for x := range g.balls { g.balls[x].fireball = false; g.balls[x].bolt = false }
			case PowerSlow:
				playSound(420, 220, 0.6, -180)
				g.ballSpeedMult *= 0.72
				if g.ballSpeedMult < 0.55 { g.ballSpeedMult = 0.55 }
				g.applyBallSpeedMultiplier()
			case PowerFast:
				playSound(300, 240, 0.75, 340)
				g.ballSpeedMult *= 1.35
				if g.ballSpeedMult > 1.95 { g.ballSpeedMult = 1.95 }
				g.applyBallSpeedMultiplier(); g.shake = 8
			case PowerFireball:
				playSound(220, 300, 0.8, 420)
				g.fireballActive = true
				for x := range g.balls { g.balls[x].fireball = true; g.balls[x].bolt = false }
				g.shake = 8
			case PowerShield:
				playSound(550, 300, 0.7, 280); g.shieldActive = true
			case PowerMissile:
				playSound(680, 250, 0.7, 340); g.missileActive = true; g.missileTimer = 600
			case PowerBolt:
				playSound(780, 280, 0.8, 400)
				g.boltActive = true
				for x := range g.balls { g.balls[x].bolt = true; g.balls[x].fireball = false }
			case PowerWarp:
				playSound(950, 500, 0.9, 500)
				g.score += 2500; g.level++; g.generateLevel(); g.resetServe()
				return
			}
		}
	}

	bricksLeft := 0
	for _, br := range g.bricks {
		if br.active && !br.isGold { bricksLeft++ }
	}
	enemiesLeft := 0
	for _, e := range g.enemies {
		if e.active { enemiesLeft++ }
	}

	if bricksLeft == 0 {
		if (g.level >= 3 || g.level == 5) && !g.enemyWaveSpawn {
			g.spawnAlienWave()
		} else if enemiesLeft == 0 {
			g.level++
			playSound(880, 400, 0.7, 320)
			g.generateLevel()
			g.resetServe()
		}
	}
}

func (g *Game) updateEnemies() {
	stepDown := false
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.active { continue }
		if e.flash > 0 { e.flash-- }
		if e.invulnCooldown > 0 { e.invulnCooldown-- }

		if e.eType == EnemyInvader {
			if e.y < e.homeY { e.y += 4.0 } else {
				e.x += g.invaderDir * 3.4
				if (e.x+e.w > screenWidth-60 && g.invaderDir > 0) || (e.x < 60 && g.invaderDir < 0) { stepDown = true }
			}
			if rand.Float32() < 0.0045 {
				g.enemyBullets = append(g.enemyBullets, EnemyBullet{
					x: e.x + e.w/2 - 7, y: e.y + e.h, w: 14, h: 30, vx: 0, vy: 7.5, active: true,
				})
				playSound(320, 60, 0.3, -140)
			}
		} else if e.eType == EnemyGalaxian {
			e.divePhase += 0.045
			if !e.diving {
				targetX := e.homeX + float32(math.Sin(e.divePhase)*75.0)
				targetY := e.homeY + float32(math.Cos(e.divePhase*2)*24.0)
				e.x += (targetX - e.x) * 0.1
				e.y += (targetY - e.y) * 0.1

				if rand.Float32() < 0.004 {
					e.diving = true
					aimX := (g.paddle.x + g.paddle.w/2 - e.x) * 0.015
					e.vx = aimX; e.vy = 6.2
					playSound(620, 180, 0.45, -380)
				}
			} else {
				e.x += e.vx + float32(math.Sin(e.divePhase*2.5)*6.5)
				e.y += e.vy
				if rand.Float32() < 0.025 {
					g.enemyBullets = append(g.enemyBullets, EnemyBullet{
						x: e.x + e.w/2 - 7, y: e.y + e.h, w: 14, h: 30, vx: e.vx * 0.4, vy: 9.0, active: true,
					})
				}
				if e.y > screenHeight+60 { e.y = -70; e.x = e.homeX; e.diving = false }
			}
		} else if e.eType == EnemyMothership {
			targetX := float32(screenWidth/2-e.w/2) + float32(math.Sin(float64(g.tick)*0.03)*220)
			e.x += (targetX - e.x) * 0.08

			if rand.Float32() < 0.012 {
				g.enemyBullets = append(g.enemyBullets, EnemyBullet{
					x: e.x + 80 + rand.Float32()*(e.w-160), y: e.y + e.h, w: 18, h: 36, vx: 0, vy: 8.5, active: true,
				})
				playSound(220, 80, 0.4, -200)
			}
		}

		if checkCol(e.x, e.y, e.w, e.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			g.damageEnemy(i, 5)
			g.shake = 18; g.paddle.w = 135; g.paddle.gunLevel = 0
		}
	}

	if stepDown {
		g.invaderDir *= -1
		for i := range g.enemies {
			if g.enemies[i].active && g.enemies[i].eType == EnemyInvader {
				g.enemies[i].homeY += 28; g.enemies[i].y += 28
			}
		}
	}

	for i := range g.enemyBullets {
		eb := &g.enemyBullets[i]
		if !eb.active { continue }
		eb.x += eb.vx; eb.y += eb.vy
		if eb.y > screenHeight { eb.active = false; continue }
		if checkCol(eb.x, eb.y, eb.w, eb.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			eb.active = false; g.loseLife(); return
		}
	}
}

func (g *Game) damageEnemy(idx int, dmg int) {
	e := &g.enemies[idx]
	e.hp -= dmg
	e.flash = 6
	g.shake = 5
	playSound(310+float64((5-e.hp)*90), 55, 0.55, 100)

	pCol := color.RGBA{80, 255, 150, 255}
	if e.eType == EnemyGalaxian { pCol = color.RGBA{255, 70, 130, 255} }
	for i := 0; i < 14; i++ {
		ang := rand.Float64() * 2 * math.Pi
		spd := rand.Float64()*7 + 2
		g.particles = append(g.particles, Particle{
			x: e.x + e.w/2, y: e.y + e.h/2,
			dx: float32(math.Cos(ang) * spd), dy: float32(math.Sin(ang) * spd),
			size: float32(rand.Intn(4) + 3), life: 25 + rand.Intn(20), max: 45, color: pCol,
		})
	}

	if e.hp <= 0 {
		e.active = false
		pts := 500 * g.level
		if e.eType == EnemyMothership { pts = 10000 }
		g.score += pts
		g.shake = 16
		playSound(140, 350, 0.9, -150)

		for i := 0; i < 35; i++ {
			ang := rand.Float64() * 2 * math.Pi
			spd := rand.Float64()*12 + 2
			g.particles = append(g.particles, Particle{
				x: e.x + e.w/2, y: e.y + e.h/2,
				dx: float32(math.Cos(ang) * spd), dy: float32(math.Sin(ang) * spd),
				size: float32(rand.Intn(6) + 4), life: 35 + rand.Intn(25), max: 60,
				color: color.RGBA{255, 200, 60, 255},
			})
		}
	}
}

func (g *Game) loseLife() {
	g.lives--
	playSound(140, 500, 0.8, -80)
	g.shake = 20
	g.paddle.gunLevel = 0
	g.paddle.sticky = false
	g.fireballActive = false
	g.boltActive = false
	g.shieldActive = false
	g.missileActive = false
	if g.lives < 0 {
		g.state = StateGameOver
		g.saveHighScore()
	} else {
		g.resetServe()
	}
}

func bounceBox(b *Ball, rx, ry, rw, rh float32) {
	overlapLeft := (b.x + b.size) - rx
	overlapRight := (rx + rw) - b.x
	overlapTop := (b.y + b.size) - ry
	overlapBottom := (ry + rh) - b.y

	minX := overlapLeft
	if overlapRight < minX { minX = overlapRight }
	minY := overlapTop
	if overlapBottom < minY { minY = overlapBottom }

	if minX < minY { b.dx *= -1 } else { b.dy *= -1 }
}

func (g *Game) hitBrick(idx int, dmg int, isBolt bool) {
	b := &g.bricks[idx]
	pColor := brickParticleColors[b.hp]
	if pColor.A == 0 { pColor = brickParticleColors[1] }
	b.hp -= dmg
	b.flash = 5
	playSound(440+float64(b.hp*110), 45, 0.45, 90)

	if isBolt {
		for otherIdx := range g.bricks {
			if otherIdx != idx && g.bricks[otherIdx].active && !g.bricks[otherIdx].isGold {
				dx := g.bricks[otherIdx].x - b.x
				dy := g.bricks[otherIdx].y - b.y
				if math.Sqrt(float64(dx*dx+dy*dy)) < 180 {
					g.bricks[otherIdx].hp--
					if g.bricks[otherIdx].hp <= 0 { g.bricks[otherIdx].active = false }
				}
			}
		}
	}

	for i := 0; i < 12; i++ {
		angle := rand.Float64() * 2 * math.Pi
		spd := rand.Float64()*6.5 + 1.5
		g.particles = append(g.particles, Particle{
			x: b.x + b.w/2, y: b.y + b.h/2,
			dx: float32(math.Cos(angle) * spd), dy: float32(math.Sin(angle) * spd),
			size: float32(rand.Intn(4) + 3), life: 24 + rand.Intn(20), max: 44, color: pColor,
		})
	}

	if b.isBomb && b.hp <= 0 {
		playSound(150, 400, 0.9, -120)
		g.shake = 15
		for otherIdx := range g.bricks {
			if g.bricks[otherIdx].active && !g.bricks[otherIdx].isGold {
				dx := g.bricks[otherIdx].x - b.x
				dy := g.bricks[otherIdx].y - b.y
				if math.Sqrt(float64(dx*dx+dy*dy)) < 150 {
					g.bricks[otherIdx].active = false
				}
			}
		}
	}

	if b.hp <= 0 {
		b.active = false
		g.combo++
		g.comboTimer = 180
		pts := 25 * g.level * g.combo
		g.score += pts

		g.popups = append(g.popups, ScorePopup{
			x: b.x + b.w/2, y: b.y, text: fmt.Sprintf("+%d (%dx)", pts, g.combo), life: 45, color: color.RGBA{255, 220, 50, 255},
		})
		g.shake = 4

		if rand.Float32() < 0.34 {
			r := rand.Float32()
			var pType PowerUpType
			switch {
			case r < 0.12: pType = PowerMultiball
			case r < 0.22: pType = PowerEnlarge
			case r < 0.34: pType = PowerLaser
			case r < 0.44: pType = PowerSticky
			case r < 0.54: pType = PowerFireball
			case r < 0.62: pType = PowerShield
			case r < 0.70: pType = PowerMissile
			case r < 0.78: pType = PowerBolt
			case r < 0.85: pType = PowerSlow
			case r < 0.91: pType = PowerFast
			case r < 0.95: pType = PowerWarp
			case r < 0.98: pType = Power1UP
			default:       pType = PowerShrink
			}
			g.powerUps = append(g.powerUps, PowerUp{
				x: b.x + b.w/2 - 34, y: b.y, w: 68, h: 30, dy: 4.8,
				pType: pType, active: true,
			})
		}
	}
}

func drawGlow(dst *ebiten.Image, x, y, radius float32, clr color.RGBA) {
	op := &ebiten.DrawImageOptions{}
	scale := float64(radius*2) / 128.0
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(float64(x-radius), float64(y-radius))
	op.Blend = ebiten.BlendLighter
	op.ColorScale.ScaleWithColor(clr)
	dst.DrawImage(glowSprite, op)
}

// --- RENDERING ---
func (g *Game) Draw(screen *ebiten.Image) {
	bgIdx := (g.level - 1) % len(bgImages)
	g.offscreen.DrawImage(bgImages[bgIdx], nil)

	for _, s := range g.stars {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(s.size)/2.0, float64(s.size)/2.0)
		op.GeoM.Translate(float64(s.x), float64(s.y))
		op.ColorScale.ScaleWithColor(s.color)
		g.offscreen.DrawImage(starSprite, op)
	}

	if g.leftPortal.active {
		drawGlow(g.offscreen, 10, g.leftPortal.y+g.leftPortal.h/2, 60, color.RGBA{0, 200, 255, 90})
		drawRect(g.offscreen, 0, g.leftPortal.y, 14, g.leftPortal.h, color.RGBA{0, 230, 255, 220})
	}
	if g.rightPortal.active {
		drawGlow(g.offscreen, screenWidth-10, g.rightPortal.y+g.rightPortal.h/2, 60, color.RGBA{220, 50, 255, 90})
		drawRect(g.offscreen, screenWidth-14, g.rightPortal.y, 14, g.rightPortal.h, color.RGBA{220, 50, 255, 220})
	}

	for _, b := range g.bricks {
		if !b.active { continue }
		drawRect(g.offscreen, b.x+6, b.y+7, b.w, b.h, color.RGBA{0, 0, 0, 125})

		var sprite *ebiten.Image
		if b.isGold { sprite = brickSprites[99] } else if b.isBomb { sprite = brickSprites[100] } else {
			sprite = brickSprites[b.hp]
			if sprite == nil { sprite = brickSprites[1] }
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.x), float64(b.y))
		if b.flash > 0 { op.ColorScale.Scale(1.9, 1.9, 1.9, 1.0) }
		g.offscreen.DrawImage(sprite, op)

		if !b.isGold && !b.isBomb && b.hp < b.maxHP {
			cx, cy := b.x+b.w*0.5, b.y+b.h*0.5
			crackCol := color.RGBA{10, 15, 30, 210}
			vector.StrokeLine(g.offscreen, cx-18, cy-10, cx+4, cy+3, 2, crackCol, true)
			vector.StrokeLine(g.offscreen, cx+4, cy+3, cx+22, cy-8, 2, crackCol, true)
		}
	}

	for _, e := range g.enemies {
		if !e.active { continue }
		glowCol := color.RGBA{50, 255, 140, 90}
		sp := invaderSprite
		if e.eType == EnemyGalaxian {
			glowCol = color.RGBA{255, 60, 120, 95}; sp = galaxSprite
		} else if e.eType == EnemyMothership {
			glowCol = color.RGBA{255, 50, 200, 140}
		}
		drawGlow(g.offscreen, e.x+e.w/2, e.y+e.h/2, 85, glowCol)

		if e.eType == EnemyMothership {
			drawRect(g.offscreen, e.x, e.y, e.w, e.h, color.RGBA{45, 25, 65, 255})
			drawRect(g.offscreen, e.x+20, e.y+20, e.w-40, e.h-40, color.RGBA{110, 45, 160, 255})
			vector.DrawFilledCircle(g.offscreen, e.x+e.w/2, e.y+e.h/2, 45, color.RGBA{255, 50, 150, 255}, true)
		} else {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(e.x), float64(e.y))
			if e.flash > 0 { op.ColorScale.Scale(2.2, 2.2, 2.2, 1.0) }
			g.offscreen.DrawImage(sp, op)
		}

		barW := float32(80)
		barX := e.x + (e.w-barW)/2
		barY := e.y - 14
		drawRect(g.offscreen, barX, barY, barW, 7, color.RGBA{20, 25, 40, 220})
		fillW := barW * (float32(e.hp) / float32(e.maxHP))
		hpCol := color.RGBA{0, 240, 130, 255}
		if e.hp <= e.maxHP/3 { hpCol = color.RGBA{255, 60, 80, 255} }
		drawRect(g.offscreen, barX, barY, fillW, 7, hpCol)
	}

	for _, p := range g.particles {
		alpha := float32(p.life) / float32(p.max)
		c := color.RGBA{p.color.R, p.color.G, p.color.B, uint8(190 * alpha)}
		drawGlow(g.offscreen, p.x, p.y, p.size*3.2*alpha, c)
		
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(p.size*alpha)/2.0, float64(p.size*alpha)/2.0)
		op.GeoM.Translate(float64(p.x), float64(p.y))
		op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, uint8(255 * alpha)})
		g.offscreen.DrawImage(starSprite, op)
	}

	for _, po := range g.popups {
		alpha := float32(po.life) / 45.0
		clr := color.RGBA{po.color.R, po.color.G, po.color.B, uint8(255 * alpha)}
		drawText(g.offscreen, po.text, fontPill, po.x, po.y, text.AlignCenter, clr)
	}

	for _, p := range g.powerUps {
		if !p.active { continue }
		drawGlow(g.offscreen, p.x+p.w/2, p.y+p.h/2, 42, color.RGBA{100, 200, 255, 70})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(p.x), float64(p.y))
		g.offscreen.DrawImage(pillSprites[p.pType], op)
	}

	for _, l := range g.lasers {
		if !l.active { continue }
		drawGlow(g.offscreen, l.x+l.w/2, l.y+l.h/2, 34, color.RGBA{0, 220, 255, 110})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(l.x), float64(l.y))
		g.offscreen.DrawImage(laserSprite, op)
	}
	for _, m := range g.missiles {
		if !m.active { continue }
		drawGlow(g.offscreen, m.x+m.w/2, m.y+m.h/2, 38, color.RGBA{255, 180, 0, 120})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(m.x), float64(m.y))
		g.offscreen.DrawImage(mislSprite, op)
	}
	for _, eb := range g.enemyBullets {
		if !eb.active { continue }
		drawGlow(g.offscreen, eb.x+eb.w/2, eb.y+eb.h/2, 30, color.RGBA{255, 50, 80, 120})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(eb.x), float64(eb.y))
		g.offscreen.DrawImage(eBulletSprite, op)
	}

	if g.state == StateServe || g.state == StatePlay || g.state == StatePaused {
		g.drawSpacecraftPaddle()

		if g.shieldActive {
			pulse := uint8(160 + math.Sin(float64(g.tick)*0.25)*90)
			drawRect(g.offscreen, 0, screenHeight-32, screenWidth, 8, color.RGBA{0, 180, 255, pulse})
			drawGlow(g.offscreen, screenWidth/2, screenHeight-28, 120, color.RGBA{0, 160, 255, 75})
		}

		for _, b := range g.balls {
			if !b.active { continue }
			orbGlow := color.RGBA{0, 200, 255, 115}
			sp := ballSprite
			if b.fireball {
				orbGlow = color.RGBA{255, 85, 15, 185}; sp = fireBallSprite
			} else if b.bolt {
				orbGlow = color.RGBA{100, 255, 255, 185}; sp = boltBallSprite
			}
			drawGlow(g.offscreen, b.x+b.size/2, b.y+b.size/2, 38, orbGlow)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(b.x), float64(b.y))
			g.offscreen.DrawImage(sp, op)
		}

		for _, b := range g.balls {
			if b.active && b.stuck {
				startX := b.x + b.size/2
				startY := b.y
				vector.StrokeLine(g.offscreen, startX, startY, startX, startY-180, 2, color.RGBA{0, 230, 255, 150}, true)
			}
		}
	}

	drawRect(g.offscreen, 0, 0, screenWidth, 58, color.RGBA{10, 16, 32, 235})
	vector.StrokeLine(g.offscreen, 0, 58, screenWidth, 58, 2, color.RGBA{0, 210, 255, 175}, false)

	drawText(g.offscreen, fmt.Sprintf("SCORE: %06d   HI: %06d", g.score, g.highScore), fontHUD, 36, 29, text.AlignStart, color.White)
	drawText(g.offscreen, fmt.Sprintf("SECTOR %d", g.level), fontHUD, screenWidth/2-110, 29, text.AlignCenter, color.RGBA{0, 225, 255, 255})

	comboStr := ""
	if g.combo > 1 { comboStr = fmt.Sprintf("COMBO: %dx   ", g.combo) }
	drawText(g.offscreen, fmt.Sprintf("%sLIVES: %d", comboStr, g.lives), fontHUD, screenWidth-36, 29, text.AlignEnd, color.RGBA{255, 220, 110, 255})

	if g.waveBannerTime > 0 && g.state != StateStart {
		drawText(g.offscreen, g.waveBanner, fontSub, screenWidth/2, 92, text.AlignCenter, color.RGBA{255, 230, 90, 255})
	}

	switch g.state {
	case StateStart:
		g.drawOverlay(
			"ARKANOID: GALAXY OVERDRIVE",
			fmt.Sprintf("SELECT SECTOR: < %d > (Press Left/Right Arrows)   |   F: Fullscreen   |   P: Pause", g.level),
			"Dash: Shift/Right-Click   |   Aim Sight when Stuck   |   Portals, Bosses, Homing Missiles & Lightning!",
		)
	case StateServe:
		drawText(g.offscreen, "PRESS SPACE OR LEFT CLICK TO LAUNCH ORB", fontSub, screenWidth/2, screenHeight-150, text.AlignCenter, color.RGBA{150, 235, 255, 255})
	case StatePaused:
		g.drawOverlay("PAUSED", "Press P or SPACE to Resume", "Take a breather, Commander.")
	case StateGameOver:
		g.drawOverlay("MISSION FAILED", fmt.Sprintf("Final Score: %d   |   Sector Reached: %d", g.score, g.level), "Press SPACE or LEFT CLICK to Try Again")
	}

	op := &ebiten.DrawImageOptions{}
	if g.shake > 0 {
		dx := float64(rand.Intn(g.shake*2+1) - g.shake)
		dy := float64(rand.Intn(g.shake*2+1) - g.shake)
		op.GeoM.Translate(dx, dy)
	}
	screen.DrawImage(g.offscreen, op)
}

func (g *Game) drawSpacecraftPaddle() {
	px, py, pw, ph := g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h
	r := ph / 2

	SpeedFactor := float32(math.Min(1.0, math.Abs(float64(g.paddle.vx))/12.0))
	flameLen := 14 + SpeedFactor*16 + float32(math.Sin(float64(g.tick)*0.5)*4)
	leftThrusterX := px + pw*0.28
	rightThrusterX := px + pw*0.72

	drawGlow(g.offscreen, leftThrusterX, py+ph+flameLen*0.4, 34, color.RGBA{0, 180, 255, 140})
	drawGlow(g.offscreen, rightThrusterX, py+ph+flameLen*0.4, 34, color.RGBA{0, 180, 255, 140})
	vector.DrawFilledCircle(g.offscreen, leftThrusterX, py+ph+4, 7, color.RGBA{180, 245, 255, 255}, true)
	vector.DrawFilledCircle(g.offscreen, rightThrusterX, py+ph+4, 7, color.RGBA{180, 245, 255, 255}, true)
	drawRect(g.offscreen, leftThrusterX-4, py+ph, 8, flameLen, color.RGBA{0, 210, 255, 190})
	drawRect(g.offscreen, rightThrusterX-4, py+ph, 8, flameLen, color.RGBA{0, 210, 255, 190})

	for _, gx := range g.getGunPositions() {
		drawRect(g.offscreen, gx-6, py-14, 12, 16, color.RGBA{75, 95, 130, 255})
		drawRect(g.offscreen, gx-3, py-19, 6, 7, color.RGBA{0, 240, 255, 255})
		drawGlow(g.offscreen, gx, py-16, 18, color.RGBA{0, 220, 255, 100})
	}

	drawRect(g.offscreen, px+r, py+4, pw-2*r, ph, color.RGBA{0, 0, 0, 140})
	drawRect(g.offscreen, px+r, py, pw-2*r, ph, color.RGBA{45, 58, 82, 255})
	drawRect(g.offscreen, px+r, py+2, pw-2*r, ph-6, color.RGBA{150, 170, 200, 255})
	drawRect(g.offscreen, px+r+8, py+3, pw-2*r-16, ph*0.35, color.RGBA{235, 245, 255, 220})

	podCol := color.RGBA{0, 195, 255, 255}
	if g.fireballActive { podCol = color.RGBA{255, 90, 20, 255} } else if g.paddle.gunLevel > 0 { podCol = color.RGBA{255, 65, 115, 255} } else if g.paddle.sticky { podCol = color.RGBA{0, 240, 175, 255} }
	drawGlow(g.offscreen, px+r, py+r, 28, podCol)
	drawGlow(g.offscreen, px+pw-r, py+r, 28, podCol)
	vector.DrawFilledCircle(g.offscreen, px+r, py+r, r, podCol, true)
	vector.DrawFilledCircle(g.offscreen, px+pw-r, py+r, r, podCol, true)
	vector.DrawFilledCircle(g.offscreen, px+r, py+r, r*0.5, color.White, true)
	vector.DrawFilledCircle(g.offscreen, px+pw-r, py+r, r*0.5, color.White, true)

	coreW := pw * 0.28
	coreX := px + (pw-coreW)/2
	drawRect(g.offscreen, coreX, py+6, coreW, ph-12, podCol)
	drawRect(g.offscreen, coreX+4, py+8, coreW-8, ph-16, color.White)

	if g.paddle.sticky {
		pulse := uint8(150 + math.Sin(float64(g.tick)*0.2)*90)
		beamCol := color.RGBA{0, 255, 190, pulse}
		vector.StrokeLine(g.offscreen, px+r, py-3, px+pw-r, py-3, 3, beamCol, true)
		drawGlow(g.offscreen, px+pw/2, py-3, 45, color.RGBA{0, 255, 180, 65})
	}
}

func (g *Game) drawOverlay(title, sub1, sub2 string) {
	cardW, cardH := float32(1240), float32(340)
	cardX, cardY := float32(screenWidth-cardW)/2, float32(screenHeight-cardH)/2
	drawRect(g.offscreen, cardX, cardY, cardW, cardH, color.RGBA{8, 14, 28, 235})
	vector.StrokeRect(g.offscreen, cardX, cardY, cardW, cardH, 2.5, color.RGBA{0, 210, 255, 200}, true)

	drawText(g.offscreen, title, fontTitle, screenWidth/2, cardY+95, text.AlignCenter, color.RGBA{0, 230, 255, 255})
	drawText(g.offscreen, sub1, fontSub, screenWidth/2, cardY+190, text.AlignCenter, color.White)
	drawText(g.offscreen, sub2, fontPill, screenWidth/2, cardY+265, text.AlignCenter, color.RGBA{170, 215, 255, 255})
}

func drawText(dst *ebiten.Image, str string, face *text.GoTextFace, x, y float32, align text.Align, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.PrimaryAlign = align
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, str, face, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func checkCol(x1, y1, w1, h1, x2, y2, w2, h2 float32) bool {
	return x1 < x2+w2 && x1+w1 > x2 && y1 < y2+h2 && y1+h1 > y2
}

func main() {
	rand.Seed(time.Now().UnixNano())
	initAudio()
	initAssets()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSize(1600, 900)
	ebiten.SetWindowTitle("Arkanoid: Galaxy Overdrive")

	game := &Game{level: 1}
	game.loadHighScore()
	game.initGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
