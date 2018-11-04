package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png" //TODO: for tiled later
	"strconv"
	"strings"

	// "github.com/lafriks/go-tiled"
	"musbah/multiplayer/common"
	key "musbah/multiplayer/common/keyboard"
	"os"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	log "github.com/sirupsen/logrus"
	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"golang.org/x/image/colornames"
	"golang.org/x/sys/windows"
)

var window *pixelgl.Window
var sprite *pixel.Sprite
var batch *pixel.Batch

//TODO: get local ID from the server on login
var localPlayerID uint32

type player struct {
	sprite *pixel.Sprite
	x      int
	y      int
}

var players = make(map[uint32]player)

var spriteFrames []pixel.Rect
var spriteSheet pixel.Picture

func run() {

	//TODO: temporary way to pick a playerID, will work differently later on (this has to match serverID)
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter playerID: ")
	text, _ := reader.ReadString('\n')
	inputPlayerID, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		log.Errorf("Could not get playerID, %s", err)
	}

	localPlayerID = uint32(inputPlayerID)

	cfg := pixelgl.WindowConfig{
		Title:  "Test",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  false,
	}

	window, err = pixelgl.NewWindow(cfg)
	if err != nil {
		log.Errorf("could not create window, %s", err)
		return
	}

	//TODO: check if I need this on or not (depends on what I go with sprite wise and what not)
	window.SetSmooth(false)

	spriteSheet, err = loadPicture("sprites/character.png")
	if err != nil {
		log.Errorf("could not load picture, %s", err)
		return
	}

	batch = pixel.NewBatch(&pixel.TrianglesData{}, spriteSheet)

	for x := spriteSheet.Bounds().Min.X; x < spriteSheet.Bounds().Max.X; x += 32 {
		for y := spriteSheet.Bounds().Min.Y; y < spriteSheet.Bounds().Max.Y; y += 32 {
			spriteFrames = append(spriteFrames, pixel.R(x, y, x+32, y+32))
		}
	}

	sprite = pixel.NewSprite(spriteSheet, spriteFrames[1])

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
	keyTick := time.Tick(common.KeyTick + 10*time.Millisecond)

	x := 0
	y := 0
	// last := time.Now()

	receiveResponseChan := make(chan bool, 1)
	receiveResponseChan <- true

	//units to move per second (in this case 10 tiles per sec)
	// speed := 500.0
	// tileSize := 50.0
	for !window.Closed() {
		// delta := time.Since(last).Seconds()
		// last = time.Now()

		window.Clear(colornames.Skyblue)

		var pressedKeys []byte
		select {
		case <-keyTick:
			if window.Pressed(pixelgl.KeyUp) {
				y += key.MoveY
				pressedKeys = append(pressedKeys, key.Up)
			}

			if window.Pressed(pixelgl.KeyDown) {
				y -= key.MoveY
				pressedKeys = append(pressedKeys, key.Down)
			}

			if window.Pressed(pixelgl.KeyLeft) {
				x -= key.MoveX
				pressedKeys = append(pressedKeys, key.Left)
			}

			if window.Pressed(pixelgl.KeyRight) {
				x += key.MoveX
				pressedKeys = append(pressedKeys, key.Right)
			}
		default:
		}

		//TODO: use delta for movement, for now just testing things
		// tilesPerSec := speed / tileSize * delta

		if len(pressedKeys) != 0 {
			go sendKeyPress(stream, pressedKeys)
		}

		//Used so that only 1 receiveResponse goroutine is created
		select {
		case <-receiveResponseChan:
			receiveResponseChan = make(chan bool, 1)
			go receiveResponse(stream, &x, &y, receiveResponseChan)
		default:
		}

		drawPlayerPosition(x, y)

		frames++
		select {
		case <-second:
			window.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			frames = 0
		default:
		}

	}

}

func sendKeyPress(stream *smux.Stream, pressedKeys []byte) {

	// log.Infof("send keys %v", pressedKeys)
	_, err := stream.Write(pressedKeys)
	if err != nil {
		log.Error(err)
		return
	}
}

func receiveResponse(stream *smux.Stream, x *int, y *int, receiveResponseChan chan bool) {
	response := make([]byte, common.MaxBytesToSendLength)
	_, err := stream.Read(response)
	if err != nil {
		log.Error(err)
		return
	}

	for i := 0; response[i] == common.PlayerByte; i += common.MaxPlayerBytesLength {
		playerID := binary.LittleEndian.Uint32(response[i+1 : common.MaxIntToBytesLength+i+1])

		if response[common.MaxIntToBytesLength+i+1] == common.MovementByte {

			newX, newY := getMovementPositionFromBytes(common.MaxIntToBytesLength+i+2, response)

			if playerID == localPlayerID {

				log.Debugf("response x is %d, y is %d", newX, newY)
				log.Debugf("x is %d and y is %d", *x, *y)
				if *x != newX || *y != newY {
					log.Debug("wrong player position, recalibrating")
					*x = newX
					*y = newY
				}

			} else {
				playerValue, ok := players[playerID]
				if ok {
					playerValue.x = newX
					playerValue.y = newY
					players[playerID] = playerValue
				} else {
					players[playerID] = player{x: newX, y: newY, sprite: pixel.NewSprite(spriteSheet, spriteFrames[1])}
				}
			}
		}

	}

	receiveResponseChan <- true
}

func drawPlayerPosition(x int, y int) {

	//TODO: combine current player and other players
	position := pixel.Vec{X: float64(x), Y: float64(y)}
	matrix := pixel.IM.Moved(position)

	batch.Clear()
	sprite.Draw(batch, matrix)

	for _, player := range players {
		position := pixel.Vec{X: float64(player.x), Y: float64(player.y)}
		matrix := pixel.IM.Moved(position)
		player.sprite.Draw(batch, matrix)
	}

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

func getMovementPositionFromBytes(baseIndex int, bytes []byte) (int, int) {
	tempX := bytes[baseIndex : common.MaxIntToBytesLength+baseIndex]
	tempY := bytes[common.MaxIntToBytesLength+baseIndex : (common.MaxIntToBytesLength*2)+baseIndex]
	newX := int(binary.LittleEndian.Uint32(tempX))
	newY := int(binary.LittleEndian.Uint32(tempY))

	return newX, newY
}
