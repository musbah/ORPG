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

		var response string

		for _, keyPress := range event.keyPress {
			switch keyPress {
			case key.Up:
				response += " up"
			case key.Down:
				response += " down"
			case key.Right:
				response += " right"
			case key.Left:
				response += " left"
			case 0:
				break
			default:
				response += " none"
			}
		}

		_, err := event.stream.Write([]byte(response))
		if err != nil {
			log.Errorf("could not write to stream %s", err)
		}
	}
}
