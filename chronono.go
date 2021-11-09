package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/alex023/clock"
	"github.com/getlantern/systray"
	"github.com/webview/webview"
)

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
var offset int64
var oldOffset int64 = -1

func reset(newOffsetMilliseconds int64) {
	offset = newOffsetMilliseconds
	if offset < 0 {
		offset = 0
	}
	broadcast("time=" + strconv.Itoa(int(offset)))
	log.Printf("Reset %d", offset)
	systray.SetTitle(fmtDuration(time.Duration(offset)))
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

func incrementTime(secondes int64) {
	if startTime == 0 {
		reset(offset + secondes*1000)
	}
}

func main() {

	host := flag.String("h", GetLocalIP(), "network host to serve on")
	port := flag.String("p", "8811", "http port to serve on")
	osc := flag.String("o", "8812", "osc port to serve on")
	flag.Parse()

	go serveHTTP(*host, *port)
	go serveOSC(*host, *osc)
	myClock := clock.NewClock()
	job, ok := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if startTime > 0 {
			broadcast("time=" + strconv.FormatInt(offset, 10))
			offset = makeTimestamp() - startTime
		}

		if math.Floor(float64(oldOffset)/1000) != math.Floor(float64(offset)/1000) {
			systray.SetTitle(fmtDuration(time.Duration(offset) * time.Millisecond))
			oldOffset = offset
		}
	})
	if !ok {
		log.Println("Fail to start timer")
	}
	defer job.Cancel()
	w := webview.New(webview.Settings{
		Width:     480,
		Height:    620,
		Title:     "Chronono",
		Resizable: true,
		URL:       "http://" + *host + ":" + *port,
	})
	defer w.Exit()
	systray.Run(func() {
		setupSystray("http://" + *host + ":" + *port)
	}, func() {
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

func setupSystray(url string) {

	systray.SetTitle(fmtDuration(0))
	systray.SetIcon(TrayIcon)
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

}
