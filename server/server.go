package main

import (
	"encoding/binary"
	"fmt"
	"musbah/ORPG/common"
	key "musbah/ORPG/common/keyboard"
	"net"
	"net/http"
	"sync"
	"time"

	_ "net/http/pprof"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type event struct {
	keyPress []byte
	player   *player
}

type gameMap struct {
	playerConnectionsMutex sync.Mutex
	playerConnections      []*playerConnection
	eventQueueMutex        sync.Mutex
	eventQueue             []event
}

type playerConnection struct {
	connection  net.Conn
	isConnected bool
}

var gameMaps = make([]gameMap, common.TotalGameMaps)

var log *zap.SugaredLogger

func main() {
	//TODO: create a better system to move between dev and production config
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		fmt.Printf("can't initialize zap logger: %v", err)
		return
	}
	defer logger.Sync() // flushes buffer, if any
	log = logger.Sugar()

	//TODO: remove later
	go func() {
		log.Info(http.ListenAndServe("localhost:6060", nil))
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
			go deleteDisconnectedConnections(index)
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
	gameMaps[mapIndex].playerConnectionsMutex.Lock()

	var wg sync.WaitGroup
	wg.Add(len(gameMaps[mapIndex].playerConnections))

	for _, connection := range gameMaps[mapIndex].playerConnections {

		go func(player *playerConnection) {

			if player.isConnected {
				//TODO: find a better way to set a deadline when someone disconnects
				err := player.connection.SetWriteDeadline(time.Now().Add(3 * time.Millisecond))
				if err != nil {
					log.Errorf("could not set write deadline, %s", err)
				}

				_, err = player.connection.Write(bytesToSend)
				if err != nil {
					log.Errorf("could not write to player's connection %s", err)
					player.isConnected = false
				}
			}

			wg.Done()

		}(connection)

	}

	wg.Wait()

	gameMaps[mapIndex].playerConnectionsMutex.Unlock()
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

func deleteDisconnectedConnections(mapIndex int) {
	gameMaps[mapIndex].playerConnectionsMutex.Lock()
	for i := 0; i < len(gameMaps[mapIndex].playerConnections); i++ {

		if !gameMaps[mapIndex].playerConnections[i].isConnected {
			gameMaps[mapIndex].playerConnections = deleteFromConnections(gameMaps[mapIndex].playerConnections, i)
			i--
		}

	}

	gameMaps[mapIndex].playerConnectionsMutex.Unlock()
}

func deleteFromConnections(array []*playerConnection, index int) []*playerConnection {
	//delete from array (overwrite value with last element's value)
	array[index] = array[len(array)-1]

	//this is needed for the gc (since the array contains pointers)
	array[len(array)-1] = nil

	return array[:len(array)-1]
}
