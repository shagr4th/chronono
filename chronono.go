package main

import (
	"flag"
	"fmt"
	"github.com/alex023/clock"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/websocket"
	"github.com/zserge/webview"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

import _ "net/http/pprof"

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
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

var upgrader = websocket.Upgrader{} // use default options
var startTime int64
var localIP = GetLocalIP()
var offset int64
var myClock = clock.NewClock()
var url string

func timeMsg(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Printf("New connection: %s", r.RemoteAddr)
	defer c.Close()
	job, inserted := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if startTime > 0 {
			var msg = []byte("time=" + strconv.FormatInt(offset, 10))
			err = c.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Println("Websocket client write error :", err)
			}
		}
	})
	if !inserted {
		log.Println("failure")
	}
	defer job.Cancel()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Websocket client read error : ", err)
			break
		}
		s := string(message)
		if s == "reset" {
			reset()
		} else if s == "start" {
			start()
		} else if s == "stop" {
			stop()
		} else if strings.HasPrefix(s, "time=") {
			s = strings.TrimPrefix(s, "time=")
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Println("Convert error :", err)
				break
			}
			if startTime == 0 {
				log.Print("Time set to " + strconv.Itoa(int(i/1000)) + " seconds")
				offset = i
			}
		}

	}
	log.Printf("Closed connection: %s", r.RemoteAddr)
}

func reset() {
	offset = 0
	log.Print("Reset defaults")
	systray.SetTitle(fmtDuration(time.Duration(0) * time.Millisecond))
}

func start() {
	if startTime == 0 {
		startTime = makeTimestamp() - offset
		log.Print("Clock started")
	}
}

func stop() {
	if startTime > 0 {
		startTime = 0
		log.Print("Clock stopped")
	}
}

func main() {

	box := packr.NewBox("./templates")

	port := flag.String("p", "8811", "http port to serve on")
	midistart := flag.String("midistart", "(BF7F7F)|(FA).*", "MIDI regex for clock start")
	midistop := flag.String("midistop", "(BF7F00)|(FC).*", "MIDI regex for clock stop")
	midireset := flag.String("midireset", "FF.*", "MIDI regex for clock reset")
	flag.Parse()

	url = "http://" + localIP + ":" + *port

	http.Handle("/", http.FileServer(box))
	http.HandleFunc("/time", timeMsg)
	go func() {
		log.Printf("Serving on %s\n", url)
		log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))
	}()
	go midiDevicesScan(midistart, midistop, midireset)

	w := webview.New(webview.Settings{
		Width:     1024,
		Height:    600,
		Title:     "Chronono",
		Resizable: true,
		URL:       url,
	})
	defer w.Exit()
	systray.Run(setupSystray, func() {
		w.Exit()
	})
	w.Run()
}

func linkListener(url string, mLink systray.MenuItem) {
	<-mLink.ClickedCh
	switch runtime.GOOS {
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows", "darwin":
		_ = exec.Command("open", url).Start()
	}
	linkListener(url, mLink)
}

func setupSystray() {

	systray.SetIcon(MyArray)
	systray.SetTooltip("Chronono")
	mLink := systray.AddMenuItem("Chronono", "Launch browser page")
	go linkListener(url, *mLink)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	go func() {
		<-mQuit.ClickedCh
		log.Println("Requesting quit")
		systray.Quit()
		log.Println("Finished quitting")
	}()

	var oldOffset int64 = -1
	myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if startTime > 0 {
			offset = makeTimestamp() - startTime
		}

		if math.Floor(float64(oldOffset)/1000) != math.Floor(float64(offset)/1000) {
			systray.SetTitle(fmtDuration(time.Duration(offset) * time.Millisecond))
			oldOffset = offset
		}
	})

}
