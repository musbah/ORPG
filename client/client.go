package main

import (
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

func run() {

	cfg := pixelgl.WindowConfig{
		Title:  "Test",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  false,
	}

	window, err := pixelgl.NewWindow(cfg)
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

	batch := pixel.NewBatch(&pixel.TrianglesData{}, spriteSheet)

	var spriteFrames []pixel.Rect
	for x := spriteSheet.Bounds().Min.X; x < spriteSheet.Bounds().Max.X; x += 32 {
		for y := spriteSheet.Bounds().Min.Y; y < spriteSheet.Bounds().Max.Y; y += 32 {
			spriteFrames = append(spriteFrames, pixel.R(x, y, x+32, y+32))
		}
	}

	sprite := pixel.NewSprite(spriteSheet, spriteFrames[len(spriteFrames)-10])

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

	//20 ticks per second
	tick := time.Tick(50 * time.Millisecond)

	x := 0.0
	y := 0.0
	last := time.Now()

	tileSize := 100.0
	for !window.Closed() {
		delta := time.Since(last).Seconds()
		last = time.Now()

		window.Clear(colornames.Skyblue)

		var pressedKeys []byte
		changes := false
		if window.Pressed(pixelgl.KeyUp) {
			changes = true
			y += tileSize * delta
			pressedKeys = append(pressedKeys, key.Up)
		}

		if window.Pressed(pixelgl.KeyDown) {
			changes = true
			y -= tileSize * delta
			pressedKeys = append(pressedKeys, key.Down)
		}

		if window.Pressed(pixelgl.KeyLeft) {
			changes = true
			x -= tileSize * delta
			pressedKeys = append(pressedKeys, key.Left)
		}

		if window.Pressed(pixelgl.KeyRight) {
			changes = true
			x += tileSize * delta
			pressedKeys = append(pressedKeys, key.Right)
		}

		matrix := pixel.IM.Moved(pixel.Vec{X: x, Y: y})

		select {
		case <-tick:
			if changes {

				_, err := stream.Write(pressedKeys)
				if err != nil {
					log.Error(err)
					return
				}

				pressedKeys = nil

				response := make([]byte, 100)
				_, err = stream.Read(response)
				if err != nil {
					log.Error(err)
					return
				}

				log.Debugf("response is %s", response)
			}
		default:
		}

		batch.Clear()
		sprite.Draw(batch, matrix)
		batch.Draw(window)

		window.Update()

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
