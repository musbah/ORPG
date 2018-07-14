package main

import (
	"net"

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

		go handleStream(stream)

	}

}

func handleStream(stream *smux.Stream) {

	for {

		buffer := make([]byte, 100)

		_, err := stream.Read(buffer)
		if err != nil {
			log.Errorf("could not read stream, %s", err)
			return
		}

		log.Debugf("read %s", buffer)
		event := event{
			stream:   stream,
			keyPress: buffer,
		}

		eventQueueMutex.Lock()
		eventQueue = append(eventQueue, event)
		eventQueueMutex.Unlock()
	}

}
