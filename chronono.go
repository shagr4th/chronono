package main

import (
	"flag"
	"fmt"
	"github.com/alex023/clock"
	"github.com/getlantern/systray"
	"github.com/gobuffalo/packr"
	"github.com/zserge/webview"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

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

// LogPrintf Log
func LogPrintf(format string, v ...interface{}) {
	log.Printf(format, v...)
	broadcast(fmt.Sprintf(format, v...))
}

// LogPrint Log
func LogPrint(v ...interface{}) {
	log.Print(v...)
	broadcast(fmt.Sprint(v...))
}

var startTime int64
var localIP = GetLocalIP()
var offset int64
var myClock = clock.NewClock()
var url string

func reset() {
	offset = 0
	hub.broadcast <- []byte("time=0")
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

	go hub.run()
	http.HandleFunc("/time", serveWs)
	go func() {
		log.Printf("Serving on %s", url)
		log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))
	}()
	go midiDevicesScan(midistart, midistop, midireset)
	job, inserted := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if startTime > 0 {
			var msg = []byte("time=" + strconv.FormatInt(offset, 10))
			hub.broadcast <- msg
		}
	})
	if !inserted {
		log.Println("failure")
	}
	defer job.Cancel()
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
