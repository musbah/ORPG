package main

import (
	"encoding/binary"
	"musbah/ORPG/common"
	key "musbah/ORPG/common/keyboard"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xtaci/smux"

	_ "net/http/pprof"
)

type event struct {
	keyPress []byte
	player   *player
}

type gameMap struct {
	playerStreamsMutex sync.Mutex
	playerStreams      []*playerStream
	eventQueueMutex    sync.Mutex
	eventQueue         []event
}

type playerStream struct {
	stream      *smux.Stream
	isConnected bool
}

var gameMaps = make([]gameMap, common.TotalGameMaps)

func main() {

	log.SetLevel(log.DebugLevel)

	//TODO: remove later
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	go mainGameLoop()
	go objectDeletionLoop()

	port := ":29902"
	startListening(port)
}

func mainGameLoop() {

	maxProcessRoutines := make(chan int, 10)

	//20 ticks per second
	tick := time.Tick(50 * time.Millisecond)

	for range tick {
		for index := 0; index < common.TotalGameMaps; index++ {
			maxProcessRoutines <- 1
			go processEvents(index, maxProcessRoutines)
		}
	}

}

func objectDeletionLoop() {

	tick := time.Tick(1 * time.Hour)

	for range tick {
		for index := 0; index < common.TotalGameMaps; index++ {
			go deleteDisconnectedStreams(index)
		}
	}
}

func processEvents(mapIndex int, maxProcessRoutines chan int) {

	gameMaps[mapIndex].eventQueueMutex.Lock()

	//Not using len and capacity because I won't use append
	//append is slower and it might eventually matter (server side at least)
	bytesToSend := make([]byte, common.MaxBytesToSendLength)
	bytesToSendIndex := 0
	for _, event := range gameMaps[mapIndex].eventQueue {

	breakKeyPress:
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
				break breakKeyPress
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

	gameMaps[mapIndex].eventQueue = nil
	gameMaps[mapIndex].eventQueueMutex.Unlock()

	if bytesToSendIndex != 0 {
		writeToClients(bytesToSend, mapIndex)
	}

	<-maxProcessRoutines
}

func writeToClients(bytesToSend []byte, mapIndex int) {
	gameMaps[mapIndex].playerStreamsMutex.Lock()

	var wg sync.WaitGroup
	wg.Add(len(gameMaps[mapIndex].playerStreams))

	for _, stream := range gameMaps[mapIndex].playerStreams {

		go func(player *playerStream) {

			if player.isConnected {
				//TODO: find a better way to set a deadline when someone disconnects
				err := player.stream.SetWriteDeadline(time.Now().Add(3 * time.Millisecond))
				if err != nil {
					log.Errorf("could not set write deadline, %s", err)
				}

				_, err = player.stream.Write(bytesToSend)
				if err != nil {
					log.Errorf("could not write to player's stream %s", err)
					player.isConnected = false
				}
			}

			wg.Done()

		}(stream)

	}

	wg.Wait()

	gameMaps[mapIndex].playerStreamsMutex.Unlock()
}

func addMovementBytes(baseIndex int, bytes []byte, currentX uint32, currentY uint32, nextX uint32, nextY uint32) int {

	//byte 0 (baseIndex) contains the response type
	//if it's movement, byte 1 is the sign of x and byte 2 is the sign of y
	//and the later bytes contain the number of x and y
	bytes[baseIndex] = common.MovementByte

	//index to start adding numbers from
	baseIndex = baseIndex + 1
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

func deleteDisconnectedStreams(mapIndex int) {
	gameMaps[mapIndex].playerStreamsMutex.Lock()
	for i := 0; i < len(gameMaps[mapIndex].playerStreams); i++ {

		if !gameMaps[mapIndex].playerStreams[i].isConnected {
			gameMaps[mapIndex].playerStreams = deleteFromStream(gameMaps[mapIndex].playerStreams, i)
			i--
		}

	}

	gameMaps[mapIndex].playerStreamsMutex.Unlock()
}

func deleteFromStream(array []*playerStream, index int) []*playerStream {
	//delete from array (overwrite value with last element's value)
	array[index] = array[len(array)-1]

	//this is needed for the gc (since the array contains pointers)
	array[len(array)-1] = nil

	return array[:len(array)-1]
}
