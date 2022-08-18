package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	_ "image/png"
	"net"
	"sync"

	key "musbah/ORPG/common/keyboard"

	"github.com/hajimehoshi/ebiten/v2"
	"go.uber.org/zap"
)

const (
	screenWidth         = 640
	screenHeight        = 480
	frame0X             = 0
	frame0Y             = 0
	frameWidth          = 32
	frameHeight         = 32
	frameAnimationCount = 3
)

type Game struct {
	frame  int
	player *player
}

//go:embed sprites/character.png
var characterPNG []byte

//TODO: get local ID from the server on login
var localPlayerID uint32

type player struct {
	sprite             *ebiten.Image
	animationFrame     int
	animationDirection int
	x                  int
	y                  int
}

var playersMutex sync.RWMutex
var players = make(map[uint32]*player)

var connection net.Conn
var spriteSheet *ebiten.Image

var log *zap.SugaredLogger

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any
	log = logger.Sugar()

	//TODO: temporary way to pick a playerID, will work differently later on (this has to match serverID)
	// reader := bufio.NewReader(os.Stdin)
	// fmt.Print("Enter playerID: ")
	// text, _ := reader.ReadString('\n')
	// inputPlayerID, err := strconv.Atoi(strings.TrimSpace(text))
	// if err != nil {
	// 	log.Errorf("Could not get playerID, %s", err)
	// }

	// localPlayerID = uint32(inputPlayerID)

	localPlayerID = uint32(1)

	img, _, err := image.Decode(bytes.NewReader(characterPNG))
	if err != nil {
		log.Fatal(err)
	}
	spriteSheet = ebiten.NewImageFromImage(img)

	// connection, err = net.Dial("tcp", "localhost:29902")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer connection.Close()

	// log.Debugf("connection from %s to %s", connection.LocalAddr(), connection.RemoteAddr())

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Testing title")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}

}

func (p *player) move(x int, y int) {
	p.animationFrame++
	p.x += x
	p.y += y

	if x > 0 {
		p.animationDirection = 2
	} else if x < 0 {
		p.animationDirection = 1
	}

	if y > 0 {
		p.animationDirection = 0
	} else if y < 0 {
		p.animationDirection = 3
	}
}

var receiveResponseChan = make(chan bool, 1)

func (g *Game) Update() error {
	g.frame++

	//Run first update call
	if g.frame == 1 {
		g.player = &player{sprite: spriteSheet, x: 0, y: 0}
		players[localPlayerID] = g.player
		receiveResponseChan <- true
		return nil
	}

	// var pressedKeys []byte

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.player.move(0, -key.MoveY)
		// pressedKeys = append(pressedKeys, key.Up)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.player.move(0, key.MoveY)
		// pressedKeys = append(pressedKeys, key.Down)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.player.move(-key.MoveX, 0)
		// pressedKeys = append(pressedKeys, key.Left)
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.player.move(key.MoveX, 0)
		// pressedKeys = append(pressedKeys, key.Right)
	}

	//TODO: use delta for movement, for now just testing things
	// tilesPerSec := speed / tileSize * delta

	// if len(pressedKeys) != 0 {
	// 	go sendKeyPress(connection, pressedKeys)
	// }

	//Used so that only 1 receiveResponse goroutine is created
	// select {
	// case <-receiveResponseChan:
	// 	receiveResponseChan = make(chan bool, 1)
	// 	go receiveResponse(connection, receiveResponseChan)
	// default:
	// }

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	drawPlayers(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

//TODO: remove old code (keeping it for later ideas)
// func run() {
// for x := spriteSheet.Bounds().Min.X; x < spriteSheet.Bounds().Max.X; x += 32 {
// 	for y := spriteSheet.Bounds().Min.Y; y < spriteSheet.Bounds().Max.Y; y += 32 {
// 		spriteFrames = append(spriteFrames, pixel.R(x, y, x+32, y+32))
// 	}
// }

// PlayerSprite := pixel.NewSprite(spriteSheet, spriteFrames[1])

// frames := 0
// second := time.Tick(time.Second)
// keyTick := time.Tick(common.KeyTick + 10*time.Millisecond)

// localPlayer := &player{sprite: PlayerSprite, x: 0, y: 0}
// players[localPlayerID] = localPlayer

// receiveResponseChan := make(chan bool, 1)
// receiveResponseChan <- true

//units to move per second (in this case 10 tiles per sec)
// speed := 500.0
// tileSize := 50.0
// for !window.Closed() {
// delta := time.Since(last).Seconds()
// last = time.Now()

// window.Clear(colornames.Skyblue)

// var pressedKeys []byte
// select {
// case <-keyTick:
// 	if window.Pressed(pixelgl.KeyUp) {
// 		localPlayer.y += key.MoveY
// 		pressedKeys = append(pressedKeys, key.Up)
// 	}

// 	if window.Pressed(pixelgl.KeyDown) {
// 		localPlayer.y -= key.MoveY
// 		pressedKeys = append(pressedKeys, key.Down)
// 	}

// 	if window.Pressed(pixelgl.KeyLeft) {
// 		localPlayer.x -= key.MoveX
// 		pressedKeys = append(pressedKeys, key.Left)
// 	}

// 	if window.Pressed(pixelgl.KeyRight) {
// 		localPlayer.x += key.MoveX
// 		pressedKeys = append(pressedKeys, key.Right)
// 	}
// default:
// }

// //TODO: use delta for movement, for now just testing things
// // tilesPerSec := speed / tileSize * delta

// if len(pressedKeys) != 0 {
// 	go sendKeyPress(connection, pressedKeys)
// }

// //Used so that only 1 receiveResponse goroutine is created
// select {
// case <-receiveResponseChan:
// 	receiveResponseChan = make(chan bool, 1)
// 	go receiveResponse(connection, receiveResponseChan)
// default:
// }

// drawPlayerPosition()

// 	}

// }

// func sendKeyPress(connection net.Conn, pressedKeys []byte) {

// 	// log.Infof("send keys %v", pressedKeys)
// 	_, err := connection.Write(pressedKeys)
// 	if err != nil {
// 		log.Errorf("could not send key press, %s", err)
// 		return
// 	}
// }

// func receiveResponse(connection net.Conn, receiveResponseChan chan bool) {
// 	response := make([]byte, common.MaxBytesToSendLength)
// 	_, err := connection.Read(response)
// 	if err != nil {
// 		log.Error(err)
// 		return
// 	}

// 	for i := 0; response[i] == common.PlayerByte; i += common.MaxPlayerBytesLength {
// 		playerID := binary.LittleEndian.Uint32(response[i+1 : common.MaxIntToBytesLength+i+1])

// 		if response[common.MaxIntToBytesLength+i+1] == common.MovementByte {

// 			//TODO: using float64 now, so fix this
// 			newX, newY := getMovementPositionFromBytes(common.MaxIntToBytesLength+i+2, response)

// 			log.Debugf("player id is %d, x is %d , y is %d", playerID, newX, newY)
// 			playersMutex.Lock()
// 			playerValue, ok := players[playerID]
// 			if ok {
// 				playerValue.x = newX
// 				playerValue.y = newY
// 				players[playerID] = playerValue
// 			} else {
// 				players[playerID] = &player{x: newX, y: newY, sprite: spriteSheet}
// 			}
// 			playersMutex.Unlock()
// 		}

// 	}

// 	receiveResponseChan <- true
// }

func drawPlayers(screen *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	screen.Fill(color.White)

	playersMutex.RLock()
	for _, player := range players {
		op.GeoM.Reset()
		op.GeoM.Translate(float64(player.x), float64(player.y))
		i := (player.animationFrame / 5) % frameAnimationCount
		j := player.animationDirection
		sx, sy := frame0X+i*frameWidth, frame0Y+j*frameHeight
		screen.DrawImage(spriteSheet.SubImage(image.Rect(sx, sy, sx+frameWidth, sy+frameHeight)).(*ebiten.Image), op)
	}
	playersMutex.RUnlock()

}

// func getMovementPositionFromBytes(baseIndex int, bytes []byte) (int, int) {
// 	tempX := bytes[baseIndex : common.MaxIntToBytesLength+baseIndex]
// 	tempY := bytes[common.MaxIntToBytesLength+baseIndex : (common.MaxIntToBytesLength*2)+baseIndex]
// 	newX := int(binary.LittleEndian.Uint32(tempX))
// 	newY := int(binary.LittleEndian.Uint32(tempY))

// 	return newX, newY
// }
