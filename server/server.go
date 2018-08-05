package main

import (
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"

	"musbah/multiplayer/common"

	"github.com/xtaci/smux"
)

type event struct {
	stream   *smux.Stream
	keyPress []byte
	player   *player
}

var eventQueueMutex sync.Mutex
var eventQueue []event

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
			processEvents()
		default:
		}
	}

}

func processEvents() {

	eventQueueMutex.Lock()
	queue := eventQueue
	eventQueue = nil
	eventQueueMutex.Unlock()

	for _, event := range queue {

		for _, keyPress := range event.keyPress {
			switch keyPress {
			case key.Up:
				event.player.y++
			case key.Down:
				event.player.y--
			case key.Right:
				event.player.x++
			case key.Left:
				event.player.x--
			case 0:
				break
			default:

			}
		}

		//TODO: change later, but for now byte 0 is the sign of x and byte 2 is the sign of y
		//and then byte 1 and 3 are the values of each
		response := make([]byte, 4)

		tempX := event.player.x
		tempY := event.player.y

		response[0] = 1
		if event.player.x < 0 {
			response[0] = 0
			tempX = -tempX
		}

		response[2] = 1
		if event.player.y < 0 {
			response[2] = 0
			tempY = -tempY
		}

		response[1] = byte(tempX)
		response[3] = byte(tempY)

		_, err := event.stream.Write(response)
		if err != nil {
			log.Errorf("could not write to stream %s", err)
		}
	}
}
