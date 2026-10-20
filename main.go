package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	screenWidth  = 1920
	screenHeight = 1080
	sampleRate   = 44100
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
}

// --- FONTS & PROCEDURAL HD ASSETS ---
var (
	fontSource *text.GoTextFaceSource
	fontHUD    *text.GoTextFace
	fontTitle  *text.GoTextFace
	fontSub    *text.GoTextFace
	fontPill   *text.GoTextFace

	bgImage        *ebiten.Image
	ballSprite     *ebiten.Image
	fireBallSprite *ebiten.Image
	glowSprite     *ebiten.Image
	laserSprite    *ebiten.Image
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
	fontTitle = &text.GoTextFace{Source: fontSource, Size: 68}
	fontSub = &text.GoTextFace{Source: fontSource, Size: 28}
	fontPill = &text.GoTextFace{Source: fontSource, Size: 18}

	bgImage = createNebulaBackground(screenWidth, screenHeight)
	glowSprite = createRadialGlow(128)
	ballSprite = createChromeOrb(24)
	fireBallSprite = createFireOrb(24)
	laserSprite = createLaserSprite(16, 40, color.RGBA{0, 230, 255, 255})
	eBulletSprite = createLaserSprite(14, 30, color.RGBA{255, 60, 90, 255})

	invaderSprite = createInvaderShip(84, 58)
	galaxSprite = createGalaxianShip(76, 62)

	brickSprites = make(map[int]*ebiten.Image)
	brickSprites[1] = createCrystalBrick(104, 38, color.RGBA{0, 200, 255, 255})  // Cyan Crystal
	brickSprites[2] = createCrystalBrick(104, 38, color.RGBA{255, 140, 20, 255}) // Solar Amber
	brickSprites[3] = createCrystalBrick(104, 38, color.RGBA{255, 40, 100, 255}) // Ruby Core
	brickSprites[4] = createCrystalBrick(104, 38, color.RGBA{175, 80, 255, 255}) // Void Amethyst

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
}

// Pre-renders a clean deep-space nebula backdrop
func createNebulaBackground(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		for x := 0; x < w; x++ {
			fx := float64(x) / float64(w)
			n1 := math.Sin(fx*4.5+t*3.0) * math.Cos(t*5.0-fx*2.0)
			n2 := math.Sin(fx*8.0-t*4.0) * 0.5
			nebula := math.Max(0, (n1+n2)*0.5)

			r := uint8(math.Min(255, 7+t*12+nebula*28))
			g := uint8(math.Min(255, 10+t*18+nebula*12))
			b := uint8(math.Min(255, 24+t*36+nebula*48))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

// Soft radial light texture used for additive neon bloom
func createRadialGlow(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	r := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := (float64(x) + 0.5 - r) / r
			dy := (float64(y) + 0.5 - r) / r
			d := math.Sqrt(dx*dx + dy*dy)
			if d < 1.0 {
				a := math.Pow(1.0-d, 2.2)
				v := uint8(255 * a)
				img.Set(x, y, color.RGBA{v, v, v, v})
			}
		}
	}
	return img
}

// High-contrast 3D chrome sphere with rim lighting
func createChromeOrb(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
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

				red := math.Min(255, 180*dot+spec+rim*0.2)
				green := math.Min(255, 215*dot+spec+rim*0.8)
				blue := math.Min(255, 255*dot+spec+rim)

				img.Set(x, y, color.RGBA{
					R: uint8(red * edgeAlpha),
					G: uint8(green * edgeAlpha),
					B: uint8(blue * edgeAlpha),
					A: uint8(255 * edgeAlpha),
				})
			}
		}
	}
	return img
}

// Molten plasma sphere for the Fireball power-up
func createFireOrb(size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	r := float64(size) / 2.0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - r
			dy := float64(y) + 0.5 - r
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				edgeAlpha := math.Min(1.0, r-dist+0.5)
				normDist := dist / r
				// White-yellow hot core transitioning to blazing orange-red rim
				core := math.Pow(math.Max(0, 1.0-normDist), 1.4)
				red := 255.0
				green := math.Min(255, 65+core*190)
				blue := math.Min(255, core*150)

				img.Set(x, y, color.RGBA{
					R: uint8(red * edgeAlpha),
					G: uint8(green * edgeAlpha),
					B: uint8(blue * edgeAlpha),
					A: uint8(255 * edgeAlpha),
				})
			}
		}
	}
	return img
}

// Faceted crystalline gem brick with glowing inner core
func createCrystalBrick(w, h int, base color.RGBA) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	bevel := 7
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x+y < 4 || (w-1-x)+y < 4 || x+(h-1-y) < 4 || (w-1-x)+(h-1-y) < 4 {
				continue
			}

			dTop := y
			dBot := h - 1 - y
			dLeft := x
			dRight := w - 1 - x
			minD := dTop
			if dBot < minD {
				minD = dBot
			}
			if dLeft < minD {
				minD = dLeft
			}
			if dRight < minD {
				minD = dRight
			}

			mult := 1.0
			spec := 0.0
			if minD < bevel {
				if minD == dTop {
					mult = 1.38
					spec = 45
				} else if minD == dLeft {
					mult = 1.18
					spec = 15
				} else if minD == dRight {
					mult = 0.72
				} else {
					mult = 0.52
				}
			} else {
				cx := (float64(x) - float64(w)*0.5) / (float64(w) * 0.5)
				cy := (float64(y) - float64(h)*0.5) / (float64(h) * 0.5)
				coreDist := math.Sqrt(cx*cx + cy*cy)
				mult = 0.85 + 0.40*math.Max(0, 1.0-coreDist)
				if y < h/2 {
					spec = 22 * (1.0 - float64(y)/float64(h/2))
				}
			}

			if minD == 0 || minD == bevel {
				mult *= 1.25
				spec += 30
			}

			r := math.Min(255, float64(base.R)*mult+spec)
			g := math.Min(255, float64(base.G)*mult+spec)
			b := math.Min(255, float64(base.B)*mult+spec)
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	return img
}

func createInvaderShip(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	cx := float32(w) / 2

	vector.DrawFilledRect(img, 6, 18, float32(w-12), 24, color.RGBA{45, 210, 120, 255}, true)
	vector.DrawFilledRect(img, 16, 8, float32(w-32), 34, color.RGBA{65, 240, 145, 255}, true)
	vector.DrawFilledCircle(img, cx, 22, 22, color.RGBA{90, 255, 175, 255}, true)
	vector.DrawFilledCircle(img, cx, 18, 14, color.RGBA{190, 255, 225, 255}, true)

	vector.DrawFilledRect(img, 10, 40, 12, 14, color.RGBA{30, 170, 95, 255}, true)
	vector.DrawFilledRect(img, float32(w-22), 40, 12, 14, color.RGBA{30, 170, 95, 255}, true)
	vector.DrawFilledRect(img, 28, 42, 10, 10, color.RGBA{30, 170, 95, 255}, true)
	vector.DrawFilledRect(img, float32(w-38), 42, 10, 10, color.RGBA{30, 170, 95, 255}, true)

	vector.DrawFilledCircle(img, cx-12, 26, 5, color.RGBA{255, 40, 70, 255}, true)
	vector.DrawFilledCircle(img, cx+12, 26, 5, color.RGBA{255, 40, 70, 255}, true)
	vector.DrawFilledCircle(img, cx-12, 26, 2.5, color.White, true)
	vector.DrawFilledCircle(img, cx+12, 26, 2.5, color.White, true)

	return img
}

func createGalaxianShip(w, h int) *ebiten.Image {
	img := ebiten.NewImage(w, h)
	cx := float32(w) / 2

	var path vector.Path
	path.MoveTo(cx, 4)
	path.LineTo(float32(w-4), 38)
	path.LineTo(float32(w-14), 56)
	path.LineTo(cx, 42)
	path.LineTo(14, 56)
	path.LineTo(4, 38)
	path.Close()

	vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].ColorR = 1.0
		vs[i].ColorG = 0.22
		vs[i].ColorB = 0.45
		vs[i].ColorA = 1.0
	}
	img.DrawTriangles(vs, is, emptySubImage(), &ebiten.DrawTrianglesOptions{AntiAlias: true})

	vector.DrawFilledRect(img, cx-10, 14, 20, 34, color.RGBA{255, 200, 40, 255}, true)
	vector.DrawFilledCircle(img, cx, 28, 8, color.RGBA{0, 240, 255, 255}, true)
	vector.DrawFilledCircle(img, cx-2, 26, 4, color.White, true)

	return img
}

var whitePixel *ebiten.Image

func emptySubImage() *ebiten.Image {
	if whitePixel == nil {
		whitePixel = ebiten.NewImage(1, 1)
		whitePixel.Fill(color.White)
	}
	return whitePixel
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
	r, g, b := c.R, c.G, c.B
	vector.DrawFilledRect(img, 2, 2, float32(w-4), float32(h-4), color.RGBA{r, g, b, 120}, true)
	vector.DrawFilledRect(img, 5, 0, float32(w-10), float32(h), color.RGBA{240, 255, 255, 255}, true)
	return img
}

// --- GAME STRUCTS ---
type GameState int

const (
	StateStart GameState = iota
	StateServe
	StatePlay
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
)

type EnemyType int

const (
	EnemyInvader  EnemyType = iota // Space Invaders marching dreadnought (5 HP)
	EnemyGalaxian                  // Galaxian agile dive-bomber (5 HP)
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
	x, y, w, h float32
	vx         float32
	speed      float32
	gunLevel   int // 0 to 4
	sticky     bool
}

type Ball struct {
	x, y, dx, dy, size float32
	active             bool
	stuck              bool
	stuckOffset        float32
	fireball           bool
}

type Brick struct {
	x, y, w, h float32
	active     bool
	hp, maxHP  int
	flash      int
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

type Particle struct {
	x, y, dx, dy, size float32
	life, max          int
	color              color.RGBA
}

type Star struct {
	x, y, speed, size float32
	color             color.RGBA
}

type Game struct {
	state          GameState
	level          int
	score          int
	lives          int
	shake          int
	tick           int
	lastMouseX     int
	ballSpeedMult  float32
	fireballActive bool
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
	particles      []Particle
	stars          []Star
	laserCooldown  int
}

var brickParticleColors = map[int]color.RGBA{
	1: {0, 200, 255, 255},
	2: {255, 140, 20, 255},
	3: {255, 40, 100, 255},
	4: {175, 80, 255, 255},
}

func (g *Game) initGame() {
	g.score = 0
	g.level = 1
	g.lives = 3
	g.ballSpeedMult = 1.0
	g.fireballActive = false
	if g.offscreen == nil {
		g.offscreen = ebiten.NewImage(screenWidth, screenHeight)
	}
	mx, _ := ebiten.CursorPosition()
	g.lastMouseX = mx

	g.stars = make([]Star, 160)
	starTints := []color.RGBA{
		{255, 255, 255, 220},
		{150, 220, 255, 200},
		{255, 210, 170, 180},
		{200, 160, 255, 190},
	}
	for i := range g.stars {
		depth := rand.Float32()
		c := starTints[rand.Intn(len(starTints))]
		c.A = uint8(70 + depth*185)
		g.stars[i] = Star{
			x:     rand.Float32() * screenWidth,
			y:     rand.Float32() * screenHeight,
			speed: 0.4 + depth*2.8,
			size:  1.0 + depth*2.2,
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
	}}
	g.powerUps = nil
	g.lasers = nil
	g.enemyBullets = nil
	g.state = StateServe
}

// Scales all active balls to match the current speed multiplier
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

	cols := 15
	bWidth := float32(104)
	bHeight := float32(38)
	padX := float32(10)
	padY := float32(10)
	startX := (screenWidth - (float32(cols)*(bWidth+padX) - padX)) / 2
	startY := float32(115)

	rows := 4 + (g.level / 2)
	if rows > 9 {
		rows = 9
	}

	pattern := g.level % 3

	for row := 0; row < rows; row++ {
		for col := 0; col <= cols/2; col++ {
			skip := false
			if pattern == 1 && (row+col)%3 == 0 {
				skip = true
			} else if pattern == 2 && col == row%4 {
				skip = true
			} else if rand.Float32() < 0.12 {
				skip = true
			}
			if skip {
				continue
			}

			hp := 1
			if row < 2 && g.level >= 2 {
				hp = 2
			}
			if row == 0 && g.level >= 3 {
				hp = 3
			}
			if g.level >= 4 && (row+col)%4 == 0 {
				hp = 4
			}

			xLeft := startX + float32(col)*(bWidth+padX)
			yPos := startY + float32(row)*(bHeight+padY)
			g.bricks = append(g.bricks, Brick{
				x: xLeft, y: yPos, w: bWidth, h: bHeight, active: true, hp: hp, maxHP: hp,
			})

			mirrorCol := cols - 1 - col
			if mirrorCol != col {
				xRight := startX + float32(mirrorCol)*(bWidth+padX)
				g.bricks = append(g.bricks, Brick{
					x: xRight, y: yPos, w: bWidth, h: bHeight, active: true, hp: hp, maxHP: hp,
				})
			}
		}
	}

	if g.level == 3 || (g.level > 3 && g.level%2 == 1) {
		g.waveBanner = fmt.Sprintf("SECTOR %d  —  ALIEN SIGNAL DETECTED BEHIND BRICKS!", g.level)
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

	isGalaxian := g.level >= 4 && g.level%2 == 0

	if !isGalaxian {
		g.waveBanner = "WARNING: SPACE INVADER ARMADA ATTACKING! (5 HITS EACH)"
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
		g.waveBanner = "DANGER: GALAXIAN DIVE-BOMBER SWARM ENGAGED! (5 HITS EACH)"
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
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}

	// Debug shortcut: Press N to clear current bricks and test waves
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && (g.state == StatePlay || g.state == StateServe) {
		for i := range g.bricks {
			g.bricks[i].active = false
		}
	}

	for i := range g.stars {
		g.stars[i].y += g.stars[i].speed
		if g.stars[i].y > screenHeight {
			g.stars[i].y = 0
			g.stars[i].x = rand.Float32() * screenWidth
		}
	}

	if g.shake > 0 {
		g.shake--
	}
	if g.waveBannerTime > 0 {
		g.waveBannerTime--
	}

	for i := range g.bricks {
		if g.bricks[i].flash > 0 {
			g.bricks[i].flash--
		}
	}

	switch g.state {
	case StateStart, StateGameOver:
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.initGame()
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
			if rand.Intn(2) == 0 {
				g.balls[0].dx = -g.balls[0].dx
			}
			g.balls[0].dy = -spd * 0.93
			g.state = StatePlay
			playSound(520, 70, 0.5, 180)
		}
	case StatePlay:
		g.updatePlay()
	}

	for i := len(g.particles) - 1; i >= 0; i-- {
		p := &g.particles[i]
		p.x += p.dx
		p.y += p.dy
		p.dy += 0.18
		p.life--
		if p.life <= 0 {
			g.particles = append(g.particles[:i], g.particles[i+1:]...)
		}
	}

	return nil
}

func (g *Game) movePaddle() {
	mx, _ := ebiten.CursorPosition()
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
		if g.paddle.vx > g.paddle.speed {
			g.paddle.vx = g.paddle.speed
		}
		if g.paddle.vx < -g.paddle.speed {
			g.paddle.vx = -g.paddle.speed
		}
		g.paddle.x += g.paddle.vx
	}

	if g.paddle.x < 24 {
		g.paddle.x = 24
		g.paddle.vx = 0
	}
	if g.paddle.x+g.paddle.w > screenWidth-24 {
		g.paddle.x = screenWidth - 24 - g.paddle.w
		g.paddle.vx = 0
	}
}

func (g *Game) getGunPositions() []float32 {
	px := g.paddle.x
	pw := g.paddle.w
	switch g.paddle.gunLevel {
	case 1:
		return []float32{px + pw/2}
	case 2:
		return []float32{px + 20, px + pw - 20}
	case 3:
		return []float32{px + 20, px + pw/2, px + pw - 20}
	case 4:
		return []float32{px + 14, px + pw*0.35, px + pw*0.65, px + pw - 14}
	default:
		return nil
	}
}

func (g *Game) updatePlay() {
	g.movePaddle()

	triggerPressed := ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justTriggered := inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	// Release magnetically stuck balls on Space / Click
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
		if released {
			playSound(560, 70, 0.5, 160)
		}
	}

	// 1. Lasers (Upgradable 1 to 4)
	if g.paddle.gunLevel > 0 {
		if g.laserCooldown > 0 {
			g.laserCooldown--
		}
		if triggerPressed && g.laserCooldown == 0 {
			playSound(880, 45, 0.3, -300)
			for _, gx := range g.getGunPositions() {
				g.lasers = append(g.lasers, Laser{
					x: gx - 8, y: g.paddle.y - 26, w: 16, h: 40, dy: -22, active: true,
				})
			}
			g.laserCooldown = 14
		}
	}

	for i := range g.lasers {
		l := &g.lasers[i]
		if !l.active {
			continue
		}
		l.y += l.dy
		if l.y < -40 {
			l.active = false
			continue
		}
		for j := range g.bricks {
			if g.bricks[j].active && checkCol(l.x, l.y, l.w, l.h, g.bricks[j].x, g.bricks[j].y, g.bricks[j].w, g.bricks[j].h) {
				l.active = false
				g.hitBrick(j, 1)
				break
			}
		}
		if !l.active {
			continue
		}
		for j := range g.enemies {
			e := &g.enemies[j]
			if e.active && checkCol(l.x, l.y, l.w, l.h, e.x, e.y, e.w, e.h) {
				l.active = false
				g.damageEnemy(j, 1)
				break
			}
		}
	}

	// 2. Update Alien Ships & Enemy Bullets
	g.updateEnemies()

	// 3. Balls (with 4x sub-stepping to handle FAST ball speeds cleanly)
	activeBalls := 0
	const subSteps = 4
	for i := range g.balls {
		b := &g.balls[i]
		if !b.active {
			continue
		}
		activeBalls++

		if b.stuck {
			b.x = g.paddle.x + g.paddle.w/2 + b.stuckOffset - b.size/2
			b.y = g.paddle.y - b.size - 3
			continue
		}

		// Spawn rising flame embers when Fireball is active
		if b.fireball && g.tick%2 == 0 {
			g.particles = append(g.particles, Particle{
				x:     b.x + b.size/2 + float32(rand.Intn(10)-5),
				y:     b.y + b.size/2 + float32(rand.Intn(10)-5),
				dx:    float32(rand.Float64()*2 - 1),
				dy:    float32(rand.Float64()*2 - 1),
				size:  float32(rand.Intn(3) + 3),
				life:  12 + rand.Intn(10),
				max:   22,
				color: color.RGBA{255, uint8(90 + rand.Intn(130)), 20, 255},
			})
		}

		for step := 0; step < subSteps; step++ {
			b.x += b.dx / subSteps
			b.y += b.dy / subSteps

			if b.x <= 0 {
				b.x = 0
				b.dx = float32(math.Abs(float64(b.dx)))
				playSound(280, 25, 0.4, 0)
			} else if b.x+b.size >= screenWidth {
				b.x = screenWidth - b.size
				b.dx = -float32(math.Abs(float64(b.dx)))
				playSound(280, 25, 0.4, 0)
			}
			if b.y <= 60 {
				b.y = 60
				b.dy = float32(math.Abs(float64(b.dy)))
				playSound(280, 25, 0.4, 0)
			}
			if b.y > screenHeight {
				b.active = false
				break
			}

			// Paddle Collision
			if b.dy > 0 && checkCol(b.x, b.y, b.size, b.size, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
				hitOffset := (b.x + b.size/2) - (g.paddle.x + g.paddle.w/2)
				maxOff := g.paddle.w/2 - 12
				if hitOffset < -maxOff {
					hitOffset = -maxOff
				}
				if hitOffset > maxOff {
					hitOffset = maxOff
				}

				if g.paddle.sticky {
					b.stuck = true
					b.stuckOffset = hitOffset
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

			// Brick collisions (Fireball pierces right through 1-HP bricks without bouncing!)
			for j := range g.bricks {
				br := &g.bricks[j]
				if !br.active {
					continue
				}
				if checkCol(b.x, b.y, b.size, b.size, br.x, br.y, br.w, br.h) {
					if b.fireball {
						if br.hp <= 1 {
							// Smash straight through 1-HP bricks without bouncing!
							g.hitBrick(j, 1)
						} else {
							// Deal 2x damage to armored bricks and bounce if it survives
							bounceBox(b, br.x, br.y, br.w, br.h)
							g.hitBrick(j, 2)
						}
					} else {
						bounceBox(b, br.x, br.y, br.w, br.h)
						g.hitBrick(j, 1)
					}
					break
				}
			}

			// Alien Ship collisions
			for j := range g.enemies {
				e := &g.enemies[j]
				if !e.active || e.invulnCooldown > 0 {
					continue
				}
				if checkCol(b.x, b.y, b.size, b.size, e.x, e.y, e.w, e.h) {
					bounceBox(b, e.x, e.y, e.w, e.h)
					e.invulnCooldown = 8
					dmg := 1
					if b.fireball {
						dmg = 2
					}
					g.damageEnemy(j, dmg)
					break
				}
			}
		}
	}

	if activeBalls == 0 {
		g.loseLife()
	}

	// 4. Power-ups
	for i := range g.powerUps {
		p := &g.powerUps[i]
		if !p.active {
			continue
		}
		p.y += p.dy
		if p.y > screenHeight {
			p.active = false
			continue
		}

		if checkCol(p.x, p.y, p.w, p.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			p.active = false
			playSound(640, 150, 0.6, 250)
			g.score += 100
			switch p.pType {
			case PowerMultiball:
				spd := float32(12.0) * g.ballSpeedMult
				for x := 0; x < len(g.balls); x++ {
					if g.balls[x].active {
						orig := g.balls[x]
						g.balls = append(g.balls,
							Ball{x: orig.x, y: orig.y, dx: -spd * 0.6, dy: -spd * 0.8, size: orig.size, active: true, fireball: g.fireballActive},
							Ball{x: orig.x, y: orig.y, dx: spd * 0.6, dy: -spd * 0.8, size: orig.size, active: true, fireball: g.fireballActive},
						)
						break
					}
				}
			case PowerEnlarge:
				g.paddle.w = 310
			case PowerLaser:
				if g.paddle.gunLevel < 4 {
					g.paddle.gunLevel++
				}
			case PowerSticky:
				g.paddle.sticky = true
			case Power1UP:
				g.lives++
			case PowerShrink:
				playSound(190, 300, 0.7, -90)
				g.paddle.w = 130
				g.paddle.gunLevel = 0
				g.paddle.sticky = false
				g.fireballActive = false
				for x := range g.balls {
					g.balls[x].fireball = false
				}
				g.shake = 12
			case PowerSlow:
				playSound(420, 220, 0.6, -180)
				g.ballSpeedMult *= 0.72
				if g.ballSpeedMult < 0.55 {
					g.ballSpeedMult = 0.55
				}
				g.applyBallSpeedMultiplier()
			case PowerFast:
				playSound(300, 240, 0.75, 340)
				g.ballSpeedMult *= 1.35
				if g.ballSpeedMult > 1.95 {
					g.ballSpeedMult = 1.95
				}
				g.applyBallSpeedMultiplier()
				g.shake = 8
			case PowerFireball:
				playSound(220, 300, 0.8, 420)
				g.fireballActive = true
				for x := range g.balls {
					g.balls[x].fireball = true
				}
				g.shake = 8
			}
		}
	}

	// Check Bricks & Alien Wave Progression
	bricksLeft := 0
	for _, br := range g.bricks {
		if br.active {
			bricksLeft++
		}
	}
	enemiesLeft := 0
	for _, e := range g.enemies {
		if e.active {
			enemiesLeft++
		}
	}

	if bricksLeft == 0 {
		if g.level >= 3 && !g.enemyWaveSpawn {
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
		if !e.active {
			continue
		}
		if e.flash > 0 {
			e.flash--
		}
		if e.invulnCooldown > 0 {
			e.invulnCooldown--
		}

		if e.eType == EnemyInvader {
			if e.y < e.homeY {
				e.y += 4.0
			} else {
				e.x += g.invaderDir * 3.4
				if (e.x+e.w > screenWidth-60 && g.invaderDir > 0) || (e.x < 60 && g.invaderDir < 0) {
					stepDown = true
				}
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
					e.vx = aimX
					e.vy = 6.2
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
				if e.y > screenHeight+60 {
					e.y = -70
					e.x = e.homeX
					e.diving = false
				}
			}
		}

		if checkCol(e.x, e.y, e.w, e.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			g.damageEnemy(i, 5)
			g.shake = 18
			g.paddle.w = 135
			g.paddle.gunLevel = 0
		}
	}

	if stepDown {
		g.invaderDir *= -1
		for i := range g.enemies {
			if g.enemies[i].active && g.enemies[i].eType == EnemyInvader {
				g.enemies[i].homeY += 28
				g.enemies[i].y += 28
			}
		}
	}

	for i := range g.enemyBullets {
		eb := &g.enemyBullets[i]
		if !eb.active {
			continue
		}
		eb.x += eb.vx
		eb.y += eb.vy
		if eb.y > screenHeight {
			eb.active = false
			continue
		}
		if checkCol(eb.x, eb.y, eb.w, eb.h, g.paddle.x, g.paddle.y, g.paddle.w, g.paddle.h) {
			eb.active = false
			g.loseLife()
			return
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
	if e.eType == EnemyGalaxian {
		pCol = color.RGBA{255, 70, 130, 255}
	}
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
		g.score += 250 * g.level
		g.shake = 12
		playSound(160, 280, 0.8, -100)

		for i := 0; i < 28; i++ {
			ang := rand.Float64() * 2 * math.Pi
			spd := rand.Float64()*10 + 2
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
	if g.lives < 0 {
		g.state = StateGameOver
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
	if overlapRight < minX {
		minX = overlapRight
	}
	minY := overlapTop
	if overlapBottom < minY {
		minY = overlapBottom
	}

	if minX < minY {
		b.dx *= -1
	} else {
		b.dy *= -1
	}
}

func (g *Game) hitBrick(idx int, dmg int) {
	b := &g.bricks[idx]
	pColor := brickParticleColors[b.hp]
	if pColor.A == 0 {
		pColor = brickParticleColors[1]
	}
	b.hp -= dmg
	b.flash = 5
	playSound(440+float64(b.hp*110), 45, 0.45, 90)

	for i := 0; i < 12; i++ {
		angle := rand.Float64() * 2 * math.Pi
		spd := rand.Float64()*6.5 + 1.5
		g.particles = append(g.particles, Particle{
			x:     b.x + b.w/2,
			y:     b.y + b.h/2,
			dx:    float32(math.Cos(angle) * spd),
			dy:    float32(math.Sin(angle) * spd),
			size:  float32(rand.Intn(4) + 3),
			life:  24 + rand.Intn(20),
			max:   44,
			color: pColor,
		})
	}

	if b.hp <= 0 {
		b.active = false
		g.score += 25 * g.level
		g.shake = 4

		// 32% chance to drop a capsule (includes FIRE, SLOW, and FAST)
		if rand.Float32() < 0.32 {
			r := rand.Float32()
			var pType PowerUpType
			switch {
			case r < 0.14:
				pType = PowerMultiball
			case r < 0.26:
				pType = PowerEnlarge
			case r < 0.42:
				pType = PowerLaser
			case r < 0.56:
				pType = PowerSticky
			case r < 0.72:
				pType = PowerFireball
			case r < 0.83:
				pType = PowerSlow
			case r < 0.92:
				pType = PowerFast
			case r < 0.96:
				pType = Power1UP
			default:
				pType = PowerShrink
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
	op.ColorScale.Scale(
		float32(clr.R)/255.0,
		float32(clr.G)/255.0,
		float32(clr.B)/255.0,
		float32(clr.A)/255.0,
	)
	dst.DrawImage(glowSprite, op)
}

// --- RENDERING ---
func (g *Game) Draw(screen *ebiten.Image) {
	// 1. Clean Deep-Space Nebula Backdrop
	g.offscreen.DrawImage(bgImage, nil)

	// 2. Parallax Stars
	for _, s := range g.stars {
		vector.DrawFilledCircle(g.offscreen, s.x, s.y, s.size, s.color, true)
	}

	// 3. Faceted Crystal Bricks with Cracks
	for _, b := range g.bricks {
		if !b.active {
			continue
		}
		vector.DrawFilledRect(g.offscreen, b.x+6, b.y+7, b.w, b.h, color.RGBA{0, 0, 0, 125}, false)

		sprite := brickSprites[b.hp]
		if sprite == nil {
			sprite = brickSprites[1]
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(b.x), float64(b.y))
		if b.flash > 0 {
			op.ColorScale.Scale(1.9, 1.9, 1.9, 1.0)
		}
		g.offscreen.DrawImage(sprite, op)

		if b.hp < b.maxHP {
			cx, cy := b.x+b.w*0.5, b.y+b.h*0.5
			crackCol := color.RGBA{10, 15, 30, 210}
			vector.StrokeLine(g.offscreen, cx-18, cy-10, cx+4, cy+3, 2, crackCol, true)
			vector.StrokeLine(g.offscreen, cx+4, cy+3, cx+22, cy-8, 2, crackCol, true)
			vector.StrokeLine(g.offscreen, cx+4, cy+3, cx-6, cy+14, 2, crackCol, true)
		}
	}

	// 4. Alien Ships with 5-HP Bars
	for _, e := range g.enemies {
		if !e.active {
			continue
		}
		glowCol := color.RGBA{50, 255, 140, 90}
		sp := invaderSprite
		if e.eType == EnemyGalaxian {
			glowCol = color.RGBA{255, 60, 120, 95}
			sp = galaxSprite
		}
		drawGlow(g.offscreen, e.x+e.w/2, e.y+e.h/2, 58, glowCol)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(e.x), float64(e.y))
		if e.flash > 0 {
			op.ColorScale.Scale(2.2, 2.2, 2.2, 1.0)
		}
		g.offscreen.DrawImage(sp, op)

		barW := float32(64)
		barX := e.x + (e.w-barW)/2
		barY := e.y - 12
		vector.DrawFilledRect(g.offscreen, barX, barY, barW, 6, color.RGBA{20, 25, 40, 220}, true)
		fillW := barW * (float32(e.hp) / float32(e.maxHP))
		hpCol := color.RGBA{0, 240, 130, 255}
		if e.hp <= 2 {
			hpCol = color.RGBA{255, 60, 80, 255}
		} else if e.hp <= 3 {
			hpCol = color.RGBA{255, 180, 30, 255}
		}
		vector.DrawFilledRect(g.offscreen, barX, barY, fillW, 6, hpCol, true)
	}

	// 5. Additive Glowing Particles
	for _, p := range g.particles {
		alpha := float32(p.life) / float32(p.max)
		c := color.RGBA{p.color.R, p.color.G, p.color.B, uint8(190 * alpha)}
		drawGlow(g.offscreen, p.x, p.y, p.size*3.2*alpha, c)
		vector.DrawFilledCircle(g.offscreen, p.x, p.y, p.size*alpha, color.White, true)
	}

	// 6. PowerUp Capsules
	for _, p := range g.powerUps {
		if !p.active {
			continue
		}
		glowCol := color.RGBA{100, 200, 255, 70}
		if p.pType == PowerFireball {
			glowCol = color.RGBA{255, 90, 20, 130}
		}
		drawGlow(g.offscreen, p.x+p.w/2, p.y+p.h/2, 42, glowCol)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(p.x), float64(p.y))
		g.offscreen.DrawImage(pillSprites[p.pType], op)
	}

	// 7. Player Lasers & Alien Plasma Bolts
	for _, l := range g.lasers {
		if !l.active {
			continue
		}
		drawGlow(g.offscreen, l.x+l.w/2, l.y+l.h/2, 34, color.RGBA{0, 220, 255, 110})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(l.x), float64(l.y))
		g.offscreen.DrawImage(laserSprite, op)
	}
	for _, eb := range g.enemyBullets {
		if !eb.active {
			continue
		}
		drawGlow(g.offscreen, eb.x+eb.w/2, eb.y+eb.h/2, 30, color.RGBA{255, 50, 80, 120})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(eb.x), float64(eb.y))
		g.offscreen.DrawImage(eBulletSprite, op)
	}

	// 8. Vaus Spacecraft Paddle & Chrome/Fire Orb Balls
	if g.state == StateServe || g.state == StatePlay {
		g.drawSpacecraftPaddle()

		for _, b := range g.balls {
			if !b.active {
				continue
			}
			orbGlow := color.RGBA{0, 200, 255, 115}
			sp := ballSprite
			if b.fireball {
				orbGlow = color.RGBA{255, 85, 15, 185}
				sp = fireBallSprite
			} else if g.ballSpeedMult > 1.15 {
				orbGlow = color.RGBA{255, 120, 20, 145}
			} else if g.ballSpeedMult < 0.85 {
				orbGlow = color.RGBA{100, 240, 255, 115}
			}
			drawGlow(g.offscreen, b.x+b.size/2, b.y+b.size/2, 38, orbGlow)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(b.x), float64(b.y))
			g.offscreen.DrawImage(sp, op)
		}
	}

	// 9. Top Glass HUD Bar
	vector.DrawFilledRect(g.offscreen, 0, 0, screenWidth, 58, color.RGBA{10, 16, 32, 235}, false)
	vector.StrokeLine(g.offscreen, 0, 58, screenWidth, 58, 2, color.RGBA{0, 210, 255, 175}, false)

	drawText(g.offscreen, fmt.Sprintf("SCORE: %06d", g.score), fontHUD, 36, 29, text.AlignStart, color.White)
	drawText(g.offscreen, fmt.Sprintf("SECTOR %d", g.level), fontHUD, screenWidth/2-110, 29, text.AlignCenter, color.RGBA{0, 225, 255, 255})

	stickyStatus := "OFF"
	if g.paddle.sticky {
		stickyStatus = "ON"
	}
	fireStatus := "OFF"
	if g.fireballActive {
		fireStatus = "ON"
	}
	speedPct := int(math.Round(float64(g.ballSpeedMult * 100)))
	drawText(
		g.offscreen,
		fmt.Sprintf("FIRE: %s   SPD: %d%%   MAG: %s   GUNS: %d/4   LIVES: %d", fireStatus, speedPct, stickyStatus, g.paddle.gunLevel, g.lives),
		fontHUD,
		screenWidth-36,
		29,
		text.AlignEnd,
		color.RGBA{255, 220, 110, 255},
	)

	if g.waveBannerTime > 0 && g.state != StateStart {
		drawText(g.offscreen, g.waveBanner, fontSub, screenWidth/2, 92, text.AlignCenter, color.RGBA{255, 230, 90, 255})
	}

	switch g.state {
	case StateStart:
		g.drawOverlay(
			"ARKANOID: GALAXY OVERDRIVE",
			"Move with MOUSE or A/D/Arrows   |   SPACE or LEFT CLICK to Launch & Fire   |   F: Fullscreen",
			"[FIRE] Pierce Bricks   [CATCH] Magnet   [LASER] 1-4x Guns   [SLOW/FAST] Speed   [MULTI] Multiball   [WIDE] Enlarge",
		)
	case StateServe:
		drawText(g.offscreen, "PRESS SPACE OR LEFT CLICK TO LAUNCH ORB", fontSub, screenWidth/2, screenHeight-150, text.AlignCenter, color.RGBA{150, 235, 255, 255})
	case StateGameOver:
		g.drawOverlay(
			"MISSION FAILED",
			fmt.Sprintf("Final Score: %d   |   Sector Reached: %d", g.score, g.level),
			"Press SPACE or LEFT CLICK to Play Again",
		)
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
	vector.DrawFilledRect(g.offscreen, leftThrusterX-4, py+ph, 8, flameLen, color.RGBA{0, 210, 255, 190}, true)
	vector.DrawFilledRect(g.offscreen, rightThrusterX-4, py+ph, 8, flameLen, color.RGBA{0, 210, 255, 190}, true)

	for _, gx := range g.getGunPositions() {
		vector.DrawFilledRect(g.offscreen, gx-6, py-14, 12, 16, color.RGBA{75, 95, 130, 255}, true)
		vector.DrawFilledRect(g.offscreen, gx-3, py-19, 6, 7, color.RGBA{0, 240, 255, 255}, true)
		drawGlow(g.offscreen, gx, py-16, 18, color.RGBA{0, 220, 255, 100})
	}

	vector.DrawFilledRect(g.offscreen, px+r, py+4, pw-2*r, ph, color.RGBA{0, 0, 0, 140}, true)
	vector.DrawFilledRect(g.offscreen, px+r, py, pw-2*r, ph, color.RGBA{45, 58, 82, 255}, true)
	vector.DrawFilledRect(g.offscreen, px+r, py+2, pw-2*r, ph-6, color.RGBA{150, 170, 200, 255}, true)
	vector.DrawFilledRect(g.offscreen, px+r+8, py+3, pw-2*r-16, ph*0.35, color.RGBA{235, 245, 255, 220}, true)

	podCol := color.RGBA{0, 195, 255, 255}
	if g.fireballActive {
		podCol = color.RGBA{255, 90, 20, 255}
	} else if g.paddle.gunLevel > 0 {
		podCol = color.RGBA{255, 65, 115, 255}
	} else if g.paddle.sticky {
		podCol = color.RGBA{0, 240, 175, 255}
	}
	drawGlow(g.offscreen, px+r, py+r, 28, podCol)
	drawGlow(g.offscreen, px+pw-r, py+r, 28, podCol)
	vector.DrawFilledCircle(g.offscreen, px+r, py+r, r, podCol, true)
	vector.DrawFilledCircle(g.offscreen, px+pw-r, py+r, r, podCol, true)
	vector.DrawFilledCircle(g.offscreen, px+r, py+r, r*0.5, color.White, true)
	vector.DrawFilledCircle(g.offscreen, px+pw-r, py+r, r*0.5, color.White, true)

	coreW := pw * 0.28
	coreX := px + (pw-coreW)/2
	vector.DrawFilledRect(g.offscreen, coreX, py+6, coreW, ph-12, podCol, true)
	vector.DrawFilledRect(g.offscreen, coreX+4, py+8, coreW-8, ph-16, color.White, true)

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
	vector.DrawFilledRect(g.offscreen, cardX, cardY, cardW, cardH, color.RGBA{8, 14, 28, 235}, true)
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

	game := &Game{}
	game.initGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
