package main

import (
	"net"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"

	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

func main() {

	//this handles terminal colors on windows
	var originalMode uint32
	stdout := windows.Handle(os.Stdout.Fd())

	windows.GetConsoleMode(stdout, &originalMode)
	windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	defer windows.SetConsoleMode(stdout, originalMode)

	log.SetLevel(log.DebugLevel)

	port := ":29902"

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

	buf := make([]byte, 100)

	for {
		_, err := stream.Read(buf)
		if err != nil {
			log.Errorf("could not read stream, %s", err)
			return
		}

		log.Debugf("read %s", buf)

		_, err = stream.Write([]byte("yup"))
		if err != nil {
			log.Errorf("could not write to stream %s", err)
		}
	}

}
