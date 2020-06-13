package app

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

type connection struct {
	streamOut chan []byte
	ws        *websocket.Conn
}

func (c *connection) read(srv *server) {
	for {
		var buff []byte

		err := websocket.Message.Receive(c.ws, &buff)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Printf("Error on try to read message: %s", err.Error())
		}

		srv.broadcasting <- buff
	}
}

func (c *connection) send() {
	for {
		select {
		case m := <-c.streamOut:
			err := websocket.Message.Send(c.ws, m)
			if err == io.EOF {
				return
			} else if err != nil {
				log.Printf("Error on try to send message: %s", err.Error())
			}
		}
	}
}

type server struct {
	pathToAssets string
	broadcasting chan []byte
	conns        map[*connection]bool
	srvMux       *http.ServeMux
}

func newServer(pathToAssets string) (*server, error) {
	if _, err := os.Stat(pathToAssets); os.IsNotExist(err) {
		return nil, errors.New("Folder to assets does not exists")
	}

	s := new(server)
	s.pathToAssets = pathToAssets
	s.broadcasting = make(chan []byte, 1)
	s.conns = make(map[*connection]bool)
	s.srvMux = http.NewServeMux()

	return s, nil
}

func (s *server) consumeBroadcast() {
	for {
		select {
		case m := <-s.broadcasting:
			log.Printf("Broadcasting to %v clients\r", len(s.conns))

			for c := range s.conns {
				select {
				case c.streamOut <- m:
					log.Println("Message from broadcast")
				default:
					log.Println("Client removed")
					delete(s.conns, c)
					close(c.streamOut)
				}
			}
		}
	}
}

// Run initiate the server using the address passed
func Run(addr, pathToAssets string) error {
	s, err := newServer(pathToAssets)
	if err != nil {
		return err
	}

	s.routes()

	go s.consumeBroadcast()

	log.Printf("Starting server at %s...\r", addr)
	if err := http.ListenAndServe(addr, s.srvMux); err != nil {
		return err
	}

	return nil
}
