package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	var (
		listen   = flag.String("listen", ":8080", "HTTP listen")
		interval = flag.Duration("interval", 1*time.Second, "broadcast interval")
	)
	flag.Parse()
	http.HandleFunc("/", handleSocket(newNumberStation(*interval)))
	log.Printf("listening on %s", *listen)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

func handleSocket(s station) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if _, ok := err.(websocket.HandshakeError); ok {
			http.Error(w, "Not a websocket handshake", 400)
			return
		} else if err != nil {
			log.Println(err)
			return
		}
		s.Subscribe(conn)
		wait(conn)
	}
}

type station interface {
	Subscribe(*websocket.Conn)
}

type numberStation struct {
	sub   chan *websocket.Conn
	unsub chan *websocket.Conn
}

func newNumberStation(interval time.Duration) *numberStation {
	s := &numberStation{
		sub:   make(chan *websocket.Conn),
		unsub: make(chan *websocket.Conn),
	}
	go s.loop(interval)
	return s
}

func (s *numberStation) loop(interval time.Duration) {
	m := map[*websocket.Conn]struct{}{}
	tick := time.Tick(interval)
	for {
		select {
		case <-tick:
			if len(m) <= 0 {
				continue
			}
			log.Printf("broadcasting to %d", len(m))
			number := rand.Intn(255)
			for conn := range m {
				go send(conn, number, s.unsub)
			}
		case conn := <-s.sub:
			log.Printf("%s: subscribed", conn.RemoteAddr())
			m[conn] = struct{}{}
		case conn := <-s.unsub:
			log.Printf("%s: unsubscribed", conn.RemoteAddr())
			delete(m, conn)
		}
	}
}

func (s *numberStation) Subscribe(conn *websocket.Conn) {
	s.sub <- conn
}

func send(conn *websocket.Conn, number int, unsub chan<- *websocket.Conn) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%02x", number))); err != nil {
		log.Printf("%s: %s", conn.RemoteAddr(), err)
		unsub <- conn
	}
}

func wait(conn *websocket.Conn) {
	defer conn.Close()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
