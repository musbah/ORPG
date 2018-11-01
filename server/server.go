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
	id     uint64
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

		//byte 0 contains the response type
		//if it's movement, byte 1 is the sign of x and byte 2 is the sign of y
		//and the later bytes contain the number of x and y
		response := make([]byte, 11)

		response[0] = common.MovementByte

		tempX := event.player.x
		tempY := event.player.y

		response[1] = 1
		if event.player.x < 0 {
			response[1] = 0
			tempX = -tempX
		}

		response[2] = 1
		if event.player.y < 0 {
			response[2] = 0
			tempY = -tempY
		}

		//index to start adding numbers from
		baseIndex := 3

		//TODO: change capacity depending on max X and max Y
		byteX := make([]byte, 4)
		binary.LittleEndian.PutUint32(byteX, tempX)
		for i := baseIndex; i < len(byteX)+baseIndex; i++ {
			response[i] = byteX[i-baseIndex]
		}

		byteY := make([]byte, 4)
		binary.LittleEndian.PutUint32(byteY, tempY)
		for i := len(byteX) + baseIndex; i < len(byteX)+len(byteY)+baseIndex; i++ {
			response[i] = byteY[i-len(byteY)-baseIndex]
		}

		gameMaps[mapIndex].playerConnectionsMutex.Lock()

		for _, playerConn := range gameMaps[mapIndex].playerConnections {
			if playerConn.id == event.player.id {
				_, err := playerConn.stream.Write(response)
				if err != nil {
					log.Errorf("could not write to stream %s", err)
				}
			} else {
				//TODO: send this player's position to the appropriate player
			}
		}

		gameMaps[mapIndex].playerConnectionsMutex.Unlock()
	}

	gameMaps[mapIndex].mutex.Unlock()
}
