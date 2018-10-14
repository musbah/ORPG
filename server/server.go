package main

import (
	"encoding/binary"
	key "musbah/multiplayer/common/keyboard"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"
	"golang.org/x/sys/windows"
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

		_, err := event.stream.Write(response)
		if err != nil {
			log.Errorf("could not write to stream %s", err)
		}
	}
}
