package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"golang.org/x/image/colornames"
	"golang.org/x/sys/windows"

	_ "image/png"

	//TODO: for tiled later
	// "github.com/lafriks/go-tiled"

	"musbah/multiplayer/common"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
)

var window *pixelgl.Window
var sprite *pixel.Sprite
var batch *pixel.Batch

func run() {

	cfg := pixelgl.WindowConfig{
		Title:  "Test",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  false,
	}

	var err error
	window, err = pixelgl.NewWindow(cfg)
	if err != nil {
		log.Errorf("could not create window, %s", err)
		return
	}

	//TODO: check if I need this on or not (depends on what I go with sprite wise and what not)
	window.SetSmooth(false)

	spriteSheet, err := loadPicture("sprites.png")
	if err != nil {
		log.Errorf("could not load picture, %s", err)
		return
	}

	batch = pixel.NewBatch(&pixel.TrianglesData{}, spriteSheet)

	var spriteFrames []pixel.Rect
	for x := spriteSheet.Bounds().Min.X; x < spriteSheet.Bounds().Max.X; x += 32 {
		for y := spriteSheet.Bounds().Min.Y; y < spriteSheet.Bounds().Max.Y; y += 32 {
			spriteFrames = append(spriteFrames, pixel.R(x, y, x+32, y+32))
		}
	}

	sprite = pixel.NewSprite(spriteSheet, spriteFrames[len(spriteFrames)-10])

	connection, err := kcp.Dial("localhost:29902")
	if err != nil {
		log.Error(err)
		return
	}
	defer connection.Close()

	session, err := smux.Client(connection, nil)
	if err != nil {
		log.Error(err)
		return
	}
	defer session.Close()

	log.Debugf("connection from %s to %s", connection.LocalAddr(), connection.RemoteAddr())

	//TODO: open more streams for different uses
	stream, err := session.OpenStream()
	if err != nil {
		log.Error(err)
		return
	}

	frames := 0
	second := time.Tick(time.Second)
	ms := time.Tick(30 * time.Millisecond)

	x := 0
	y := 0
	// last := time.Now()

	//units to move per second (in this case 10 tiles per sec)
	// speed := 500.0
	// tileSize := 50.0
	for !window.Closed() {
		// delta := time.Since(last).Seconds()
		// last = time.Now()

		window.Clear(colornames.Skyblue)

		var pressedKeys []byte
		select {
		case <-ms:
			if window.Pressed(pixelgl.KeyUp) {
				y++
				pressedKeys = append(pressedKeys, key.Up)
			}

			if window.Pressed(pixelgl.KeyDown) {
				y--
				pressedKeys = append(pressedKeys, key.Down)
			}

			if window.Pressed(pixelgl.KeyLeft) {
				x--
				pressedKeys = append(pressedKeys, key.Left)
			}

			if window.Pressed(pixelgl.KeyRight) {
				x++
				pressedKeys = append(pressedKeys, key.Right)
			}
		default:
		}

		//TODO: use delta for movement, for now just testing things
		// tilesPerSec := speed / tileSize * delta

		if len(pressedKeys) != 0 {
			go sendKeyPressAndCheckPosition(stream, pressedKeys, &x, &y)
		}

		drawPlayerPosition(x, y)

		frames++
		select {
		case <-second:
			window.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			log.Debugf("second passed")
			frames = 0
		default:
		}

	}

}

func sendKeyPressAndCheckPosition(stream *smux.Stream, pressedKeys []byte, x *int, y *int) {

	log.Infof("send keys %v", pressedKeys)
	_, err := stream.Write(pressedKeys)
	if err != nil {
		log.Error(err)
		return
	}

	response := make([]byte, 100)
	_, err = stream.Read(response)
	if err != nil {
		log.Error(err)
		return
	}

	//TODO: change capacity depending on the server changes
	tempX := response[2:6]
	tempY := response[6:10]
	newX := int(binary.LittleEndian.Uint32(tempX))
	newY := int(binary.LittleEndian.Uint32(tempY))

	if response[0] == 0 {
		newX = -newX
	}

	if response[1] == 0 {
		newY = -newY
	}

	log.Debugf("response x is %d, y is %d", newX, newY)
	log.Debugf("x is %d and y is %d", *x, *y)
	if *x != newX || *y != newY {
		log.Info("wrong player position, recalibrating")
		*x = newX
		*y = newY
	}

}

func drawPlayerPosition(x int, y int) {

	position := pixel.Vec{X: float64(x), Y: float64(y)}
	matrix := pixel.IM.Moved(position)

	batch.Clear()
	sprite.Draw(batch, matrix)
	batch.Draw(window)

	window.Update()
}

func main() {

	//this handles terminal colors on windows
	var originalMode uint32
	stdout := windows.Handle(os.Stdout.Fd())

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	defer windows.SetConsoleMode(stdout, originalMode)

	log.SetLevel(log.DebugLevel)

	pixelgl.Run(run)
}

func loadPicture(path string) (pixel.Picture, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return pixel.PictureDataFromImage(img), nil
}
