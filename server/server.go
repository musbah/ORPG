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
	playerStreamsMutex sync.Mutex
	playerStreams      []*smux.Stream
	eventQueueMutex    sync.Mutex
	eventQueue         []event
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

		//byte 0 is the sign of x and byte 1 is the sign of y
		//and then byte 2 and 3 are the values of each
		response := make([]byte, 10)

		tempX := event.player.x
		tempY := event.player.y

		response[0] = 1
		if event.player.x < 0 {
			response[0] = 0
			tempX = -tempX
		}

		response[1] = 1
		if event.player.y < 0 {
			response[1] = 0
			tempY = -tempY
		}

		//TODO: change capacity depending on max X and max Y
		byteX := make([]byte, 4)
		binary.LittleEndian.PutUint32(byteX, tempX)
		for i := 2; i < len(byteX)+2; i++ {
			response[i] = byteX[i-2]
		}

		byteY := make([]byte, 4)
		binary.LittleEndian.PutUint32(byteY, tempY)
		for i := len(byteX) + 2; i < len(byteX)+len(byteY)+2; i++ {
			response[i] = byteY[i-len(byteY)-2]
		}

		gameMaps[mapIndex].playerStreamsMutex.Lock()

		for _, stream := range gameMaps[mapIndex].playerStreams {
			if stream.ID() == event.streamID {
				_, err := stream.Write(response)
				if err != nil {
					log.Errorf("could not write to stream %s", err)
				}
			} else {
				//TODO: send this player's position to the appropriate player
			}
		}

		gameMaps[mapIndex].playerStreamsMutex.Unlock()
	}
}
