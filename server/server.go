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
	mutex           sync.Mutex
	streamsMutex    sync.Mutex
	streams         []*smux.Stream
	eventQueueMutex sync.Mutex
	eventQueue      []event
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

	//Not using len and capacity because I won't use append
	//append is slower and it might eventually matter (server side at least)
	bytesToSend := make([]byte, common.MaxBytesToSendLength)
	bytesToSendIndex := 0
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

		playerBytes := make([]byte, common.MaxPlayerBytesLength)
		playerBytes[0] = common.PlayerByte
		length := addIntToBytes(1, playerBytes, event.player.id)
		length = addMovementBytes(length, playerBytes, event.player.x, event.player.y, tempX, tempY)

		for i := bytesToSendIndex; i < length+bytesToSendIndex; i++ {
			bytesToSend[i] = playerBytes[i-bytesToSendIndex]
		}

		bytesToSendIndex += length
	}

	gameMaps[mapIndex].streamsMutex.Lock()

	for _, stream := range gameMaps[mapIndex].streams {

		_, err := stream.Write(bytesToSend)
		if err != nil {
			log.Errorf("could not write to player's stream %s", err)
		}

	}

	gameMaps[mapIndex].streamsMutex.Unlock()

	gameMaps[mapIndex].mutex.Unlock()
}

func addMovementBytes(baseIndex int, bytes []byte, currentX uint32, currentY uint32, nextX uint32, nextY uint32) int {

	//byte 0 (baseIndex) contains the response type
	//if it's movement, byte 1 is the sign of x and byte 2 is the sign of y
	//and the later bytes contain the number of x and y
	bytes[baseIndex] = common.MovementByte

	bytes[baseIndex+1] = 1
	if currentX < 0 {
		bytes[baseIndex+1] = 0
		nextX = -nextX
	}

	bytes[baseIndex+2] = 1
	if currentY < 0 {
		bytes[baseIndex+2] = 0
		nextY = -nextY
	}

	//index to start adding numbers from
	baseIndex = baseIndex + 3
	return addPositionToBytes(baseIndex, bytes, nextX, nextY)
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
