package main

import (
	"io"
	"musbah/ORPG/common"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

func startListening(port string) {

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Errorf("could not listen for packets, %s", err)
		return
	}
	defer close(listener)

	log.Debugf("connection addr %s", listener.Addr())

	for {

		connection, err := listener.Accept()
		if err != nil {
			log.Errorf("could not accept connection, %s", err)
			return
		}
		defer close(connection)

		log.Debug("accepted new connection")
		go handleConnection(connection)
	}
}

func handleConnection(connection net.Conn) {

	defer close(connection)
	var lastEvent time.Time
	player := loadPlayer()
	for {
		gameMaps[player.mapIndex].playerConnectionsMutex.Lock()
		gameMaps[player.mapIndex].playerConnections = append(gameMaps[player.mapIndex].playerConnections, &playerConnection{connection: connection, isConnected: true})
		gameMaps[player.mapIndex].playerConnectionsMutex.Unlock()

		buffer := make([]byte, 10)

		_, err := io.ReadFull(connection, buffer)
		if err != nil {
			log.Errorf("could not read connection, %s", err)
			return
		}

		//Used to limit key event interval
		if lastEvent.Add(common.KeyTick).After(time.Now()) {
			log.Debug("skipping key event")
			continue
		}

		lastEvent = time.Now()

		log.Debugf("read %s", buffer)
		event := event{
			keyPress: buffer,
			player:   player,
		}

		gameMaps[player.mapIndex].eventQueueMutex.Lock()
		gameMaps[player.mapIndex].eventQueue = append(gameMaps[player.mapIndex].eventQueue, event)
		gameMaps[player.mapIndex].eventQueueMutex.Unlock()
	}

}

func close(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Error(err)
	}
}
