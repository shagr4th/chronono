package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

type ChronoServer struct {
	http.Handler
	host *string
	port *string
	osc  *string

	startTime int64
	offset    int64
	oldOffset int64

	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool

	oscClients map[string]*osc.Client
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func fmtDuration(d time.Duration) string {
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// LogPrintf Log
func (server *ChronoServer) LogPrintf(format string, v ...interface{}) {
	log.Printf(format, v...)
	server.sseBroadcast(fmt.Sprintf(format, v...))
}

// LogPrint Log
func (server *ChronoServer) LogPrint(v ...interface{}) {
	log.Print(v...)
	server.sseBroadcast(fmt.Sprint(v...))
}

func (server *ChronoServer) listen() {
	for {
		select {
		case s := <-server.newClients:

			// A new client has connected.
			// Register their message channel
			server.clients[s] = true
			log.Printf("Client added. %d registered clients", len(server.clients))
		case s := <-server.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(server.clients, s)
			log.Printf("Removed client. %d registered clients", len(server.clients))
		case event := <-server.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range server.clients {
				clientMessageChan <- event
			}
		}
	}

}
func NewChronoServer() *ChronoServer {
	// Instantiate a server
	var server = &ChronoServer{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
		oldOffset:      -1,
		oscClients:     make(map[string]*osc.Client),
	}

	// Set it running - listening and broadcasting events
	go server.listen()

	return server
}

func (server *ChronoServer) startHTTPListener(handler http.Handler) {
	log.Fatal(http.ListenAndServe(*server.host+":"+*server.port, handler))
}

func (server *ChronoServer) getHTTPUrl() string {
	return "http://" + *server.host + ":" + *server.port
}

func (server *ChronoServer) getOSCUrl() string {
	return *server.host + ":" + *server.osc
}

func (server *ChronoServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/sse" {

		// Make sure that the writer supports flushing.
		flusher, ok := w.(http.Flusher)

		if !ok {
			log.Println("Flusher unsupported!")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", server.getHTTPUrl())

		// Each connection registers its own message channel with the Broker's connections registry
		messageChan := make(chan []byte)

		// Signal the broker that we have a new connection
		server.newClients <- messageChan

		// Remove this client from the map of connected clients
		// when this handler exits.
		defer func() {
			server.closingClients <- messageChan
		}()

		// Listen to connection close and un-register messageChan
		// notify := rw.(http.CloseNotifier).CloseNotify()
		notify := r.Context().Done()

		go func() {
			<-notify
			server.closingClients <- messageChan
		}()

		go func() {
			time.Sleep(200 * time.Millisecond)
			server.sseBroadcast("http=" + server.getHTTPUrl() +
				", OSC: " + server.getOSCUrl())
			server.sseBroadcastTime()
			server.sseBroadcast("New Client ! " + r.RemoteAddr)
		}()

		for {

			// Write to the ResponseWriter
			// Server Sent Events compatible
			fmt.Fprintf(w, "data: %s\n\n", <-messageChan)

			// Flush the data immediatly instead of buffering it for later.
			if ok {
				flusher.Flush()
			}
		}

	} else if r.URL.Path == "/reset" {
		server.resetTimer(0)
		server.sseBroadcastTime()
	} else if r.URL.Path == "/start" {
		server.startTimer()
		server.sseBroadcastTime()
	} else if r.URL.Path == "/stop" {
		server.stopTimer()
		server.sseBroadcastTime()
	} else if r.URL.Path == "/config" && strings.HasPrefix(r.URL.RawQuery, "clients=") {
		var s = strings.TrimPrefix(r.URL.RawQuery, "clients=")
		server.oscInitClients(s)
	} else if r.URL.Path == "/config" && strings.HasPrefix(r.URL.RawQuery, "time=") {
		var s = strings.TrimPrefix(r.URL.RawQuery, "time=")
		f, err := strconv.ParseFloat(s, 64)
		i := int64(f)
		if err != nil {
			server.LogPrintf("Convert error : %s", err)
			w.WriteHeader(422)
			return
		}
		if server.startTime == 0 {
			server.LogPrintf("Set server time to " + strconv.Itoa(int(i/1000)) + " seconds")
			server.offset = i
			server.sseBroadcastTime()
		}
	} else {
		w.WriteHeader(404)
		return
	}
}

func (server *ChronoServer) sseBroadcast(msg string) {
	server.Notifier <- []byte(msg)
}

func (server *ChronoServer) sseBroadcastTime() {
	server.sseBroadcast("time=" + strconv.FormatInt(server.offset, 10))
}

func (server *ChronoServer) resetTimer(newOffsetMilliseconds int64) {
	server.LogPrintf("Reset server time to 0 seconds")
	if server.offset != newOffsetMilliseconds {
		server.offset = newOffsetMilliseconds
		if server.offset < 0 {
			server.offset = 0
		}
		server.LogPrintf("Reset %d", server.offset)
	}
}

func (server *ChronoServer) startTimer() {
	if server.startTime == 0 {
		server.startTime = makeTimestamp() - server.offset
		log.Print("Clock started")
	}
}

func (server *ChronoServer) stopTimer() {
	if server.startTime > 0 {
		server.startTime = 0
		log.Print("Clock stopped")
	}
}

func (server *ChronoServer) incrementTime(secondes int64) {
	if server.startTime == 0 {
		server.resetTimer(server.offset + secondes*1000)
	}
}

// GetLocalIP returns the non loopback local IP of the host
func (server *ChronoServer) GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
