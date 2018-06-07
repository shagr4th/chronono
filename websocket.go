package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Client connection
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// Hub central hub
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

var hub = newHub()
var upgrader = websocket.Upgrader{} // use default options

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func broadcast(msg string) {
	go func() {
		hub.broadcast <- []byte(msg)
	}()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			LogPrintf("Opened client connection: %s", client.conn.RemoteAddr().String())
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				LogPrintf("Closed client connection: %s", client.conn.RemoteAddr().String())
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		return
	}

	client := &Client{hub: hub, conn: c, send: make(chan []byte, 256)}
	client.hub.register <- client

	go client.readPump()
	go client.writePump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Println("Websocket client read error : ", err)
			break
		}
		s := string(message)
		if s == "reset" {
			LogPrint("Reset server time to 0 seconds")
			reset()
		} else if s == "start" {
			start()
		} else if s == "stop" {
			stop()
		} else if strings.HasPrefix(s, "time=") {
			s = strings.TrimPrefix(s, "time=")
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				LogPrint("Convert error :", err)
				break
			}
			if startTime == 0 {
				LogPrint("Set server time to " + strconv.Itoa(int(i/1000)) + " seconds")
				offset = i
				broadcast("time=" + strconv.FormatInt(offset, 10))
			}
		}

	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	if startTime == 0 {
		err := c.conn.WriteMessage(websocket.TextMessage, []byte("time="+strconv.FormatInt(offset, 10)))
		if err != nil {
			return
		}
	}
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}
		}
	}
}
