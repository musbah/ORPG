package main

import (
	"musbah/multiplayer/common"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	kcp "github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

func startListening(port string) {
	listener, err := kcp.Listen(port)
	if err != nil {
		log.Errorf("could not listen for packets, %s", err)
		return
	}
	defer listener.Close()

	log.Debugf("connection addr %s", listener.Addr())

	for {

		connection, err := listener.Accept()
		if err != nil {
			log.Errorf("could not accept connection, %s", err)
			return
		}
		defer connection.Close()

		go initializeSmuxSession(connection)
	}
}

func initializeSmuxSession(connection net.Conn) {

	session, err := smux.Server(connection, nil)
	if err != nil {
		log.Errorf("could not create a connection, %s", err)
		return
	}
	defer session.Close()

	player := loadPlayer()
	for {
		stream, err := session.AcceptStream()
		if err != nil {

			if session.IsClosed() {
				log.Printf("session closed")
				return
			}

			log.Errorf("could not accept stream, %s", err)
			continue
		}
		defer stream.Close()

		gameMaps[player.mapIndex].playerStreamsMutex.Lock()
		gameMaps[player.mapIndex].playerStreams = append(gameMaps[player.mapIndex].playerStreams, stream)
		gameMaps[player.mapIndex].playerStreamsMutex.Unlock()

		go handleStream(stream, player)

	}

}

func handleStream(stream *smux.Stream, player *player) {

	var lastEvent time.Time
	for {

		buffer := make([]byte, 100)

		_, err := stream.Read(buffer)
		if err != nil {
			log.Errorf("could not read stream, %s", err)
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
			streamID: stream.ID(),
			keyPress: buffer,
			player:   player,
		}

		gameMaps[player.mapIndex].eventQueueMutex.Lock()
		gameMaps[player.mapIndex].eventQueue = append(gameMaps[player.mapIndex].eventQueue, event)
		gameMaps[player.mapIndex].eventQueueMutex.Unlock()
	}

}
