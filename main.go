package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/websocket"
)

type connection struct {
	ctx  context.Context
	send chan string
	ws   *websocket.Conn
}

func newConnection(ctx context.Context, ws *websocket.Conn) *connection {
	return &connection{
		ctx:  ctx,
		send: make(chan string, 1),
		ws:   ws,
	}
}

var srvCtx context.Context
var srvCancel context.CancelFunc
var hubConnections map[*connection]bool
var broadcast chan string

func main() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
	defer srvCancel()

	hubConnections = make(map[*connection]bool)
	broadcast = make(chan string, 1)
	defer close(broadcast)

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.Handle("/ws", websocket.Handler(func(conn *websocket.Conn) {
		log.Printf("New connection %v\r", conn)

		conn.MaxPayloadBytes = 1024 * 1024

		c := newConnection(srvCtx, conn)
		hubConnections[c] = true
		handleWSConnection(c)
	}))

	go handleWSBroadcast()

	log.Println("Listening on :3000...")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}

func gracefulShutdown() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		srvCancel()
		log.Println("Ctrl+C pressed in Terminal")
		os.Exit(0)
	}()
}

func handleWSBroadcast() {
	for {
		select {
		case <-srvCtx.Done():
			return
		case m := <-broadcast:
			log.Printf("Hub length: %v", len(hubConnections))

			for c := range hubConnections {
				select {
				case c.send <- m:
					log.Println("Message from broadcast")
				default:
					log.Println("Client removed")
					delete(hubConnections, c)
					close(c.send)
				}
			}
		}
	}
}

func readFromConnection(conn *connection) {
	for {
		log.Printf("Read to send to client - %s\r", conn.ws.RemoteAddr())

		m, ok := <-conn.send
		if !ok {
			log.Println("Message not ok to read")
			return
		}

		err := websocket.Message.Send(conn.ws, m)

		if err != nil {
			log.Printf("Error when try send message: %v", err)
		}
	}
}

func handleWSConnection(conn *connection) {
	defer conn.ws.Close()

	// send to client
	go (func() {

	})()

	// read from client
	for {
		select {
		case <-conn.ctx.Done():
			return
		default:
			var msg string

			err := websocket.Message.Receive(conn.ws, &msg)
			if err == io.EOF {
				return
			} else if err != nil {
				log.Printf("Error when try read message: %v", err)
			}

			broadcast <- msg

			log.Printf("Received from client %s\r", conn.ws.RemoteAddr().String())
		}
	}
}
