package main

import (
	"io"
	"musbah/multiplayer/common"
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
		go initializeSmuxSession(connection)
	}
}

func initializeSmuxSession(connection io.ReadWriteCloser) {

	session, err := smux.Server(connection, nil)
	if err != nil {
		log.Errorf("could not create a connection, %s", err)
		return
	}
	defer close(session)

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

		gameMaps[player.mapIndex].streamsMutex.Lock()
		gameMaps[player.mapIndex].streams = append(gameMaps[player.mapIndex].streams, &streamWrapper{stream: stream, isConnected: true})
		gameMaps[player.mapIndex].streamsMutex.Unlock()

		go handleStream(stream, player)

	}

}

func handleStream(stream *smux.Stream, player *player) {

	var lastEvent time.Time
	for {

		buffer := make([]byte, 10)

		_, err := stream.Read(buffer)
		if err != nil {
			log.Errorf("could not read stream, %s", err)
			close(stream)
			return
		}

		//Used to limit key event interval
		if lastEvent.Add(common.KeyTick).After(time.Now()) {
			log.Debug("skipping key event")
			continue
		}

		lastEvent = time.Now()

		// log.Debugf("read %s", buffer)
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
