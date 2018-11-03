package main

import (
	"encoding/binary"
	"musbah/multiplayer/common"
	key "musbah/multiplayer/common/keyboard"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"golang.org/x/sys/windows"
)

type event struct {
	streamID uint32
	keyPress []byte
	player   *player
}

type gameMap struct {
	mutex                  sync.Mutex
	playerConnectionsMutex sync.Mutex
	playerConnections      []playerConnection
	eventQueueMutex        sync.Mutex
	eventQueue             []event
}

type playerConnection struct {
	id     uint32
	stream *smux.Stream
}

var gameMaps = make([]gameMap, common.TotalGameMaps)

func main() {

	//this handles terminal colors on windows
	var originalMode uint32
	stdout := windows.Handle(os.Stdout.Fd())

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	defer windows.SetConsoleMode(stdout, originalMode)

	log.SetLevel(log.DebugLevel)

	go mainGameLoop()

	port := ":29902"
	startListening(port)
}

func mainGameLoop() {

	//20 ticks per second
	tick := time.Tick(50 * time.Millisecond)

	for {
		select {
		case <-tick:
			processGameMaps()
		default:
		}
	}

}

func processGameMaps() {
	for index := range gameMaps {
		go processEvents(index)
	}
}

func processEvents(mapIndex int) {

	gameMaps[mapIndex].mutex.Lock()

	gameMaps[mapIndex].eventQueueMutex.Lock()
	queue := gameMaps[mapIndex].eventQueue
	gameMaps[mapIndex].eventQueue = nil
	gameMaps[mapIndex].eventQueueMutex.Unlock()

	for _, event := range queue {

		for _, keyPress := range event.keyPress {
			switch keyPress {
			case key.Up:
				event.player.y += key.MoveY
			case key.Down:
				event.player.y -= key.MoveY
			case key.Right:
				event.player.x += key.MoveX
			case key.Left:
				event.player.x -= key.MoveX
			case 0:
				break
			default:

			}
		}

		tempX := event.player.x
		tempY := event.player.y

		movementBytes, movementBytesLength := createMovementBytes(event.player.x, event.player.y, tempX, tempY)

		otherPlayerBytes := make([]byte, 1+common.MaxIntToBytesLength+movementBytesLength)
		otherPlayerBytes[0] = common.OtherPlayerByte
		addIntToBytes(1, otherPlayerBytes, event.player.id)

		gameMaps[mapIndex].playerConnectionsMutex.Lock()

		for _, playerConn := range gameMaps[mapIndex].playerConnections {
			if playerConn.id == event.player.id {

				_, err := playerConn.stream.Write(movementBytes)
				if err != nil {
					log.Errorf("could not write to player's stream %s", err)
				}

			} else {

				_, err := playerConn.stream.Write(otherPlayerBytes)
				if err != nil {
					log.Errorf("could not write to other player's stream %s", err)
				}

			}
		}

		gameMaps[mapIndex].playerConnectionsMutex.Unlock()
	}

	gameMaps[mapIndex].mutex.Unlock()
}

func createMovementBytes(currentX uint32, currentY uint32, nextX uint32, nextY uint32) ([]byte, int) {

	//byte 0 contains the response type
	//if it's movement, byte 1 is the sign of x and byte 2 is the sign of y
	//and the later bytes contain the number of x and y
	bytes := make([]byte, 3+common.MaxIntToBytesLength*2)

	bytes[0] = common.MovementByte

	bytes[1] = 1
	if currentX < 0 {
		bytes[1] = 0
		nextX = -nextX
	}

	bytes[2] = 1
	if currentY < 0 {
		bytes[2] = 0
		nextY = -nextY
	}

	//index to start adding numbers from
	baseIndex := 3
	length := addPositionToBytes(baseIndex, bytes, nextX, nextY)

	return bytes, length
}

func addPositionToBytes(baseIndex int, bytes []byte, tempX uint32, tempY uint32) int {

	length := addIntToBytes(baseIndex, bytes, tempX)
	length = addIntToBytes(length, bytes, tempY)

	return length
}

func addIntToBytes(baseIndex int, bytes []byte, numberToAppend uint32) int {
	byteNumber := make([]byte, common.MaxIntToBytesLength)
	binary.LittleEndian.PutUint32(byteNumber, numberToAppend)

	length := len(byteNumber) + baseIndex
	for i := baseIndex; i < length; i++ {
		bytes[i] = byteNumber[i-baseIndex]
	}

	return length
}
