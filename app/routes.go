package app

import (
	"net/http"

	"golang.org/x/net/websocket"
)

func (s *server) routes() {
	s.srvMux.Handle("/", s.FileServer())
	// s.srvMux.HandleFunc("/", s.HelloIndex())
	s.srvMux.Handle("/ws", s.HandleWS())
}

// func (s *server) HelloIndex() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {

// 	}
// }

func (s *server) HandleWS() websocket.Handler {
	return func(conn *websocket.Conn) {
		defer conn.Close()

		c := new(connection)
		c.streamOut = make(chan []byte, 1)
		c.ws = conn
		s.conns[c] = true

		go c.send()
		c.read(s)
	}
}

func (s *server) FileServer() http.Handler {
	return http.FileServer(http.Dir(s.pathToAssets))
}
