package main

import (
	"embed"
	"flag"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/alex023/clock"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

func main() {

	var server = NewChronoServer()

	server.host = flag.String("h", GetLocalIP(), "network host to serve on")
	server.port = flag.String("p", "8811", "http port to serve on")
	server.osc = flag.String("o", "8812", "osc port to serve on")

	app := application.New(application.Options{
		Name:        "Chronono",
		LogLevel:    slog.LevelInfo,
		Description: "OSC and HTTP clock control",
		Assets: application.AssetOptions{
			FS: assets,
			Middleware: func(next http.Handler) http.Handler {
				go server.serverOnRealHTTPProtocol(next)
				return next
			},
			Handler: server,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	// Create window
	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "Chronono",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		Width:         480,
		Height:        720,
		DisableResize: true,
		URL:           "/",
	})

	systray := app.NewSystemTray()
	menu := app.NewMenu()
	url := "http://" + *server.host + ":" + *server.port
	menu.Add(url).OnClick(func(ctx *application.Context) {
		switch runtime.GOOS {
		case "linux":
			_ = exec.Command("xdg-open", url).Start()
		case "windows", "darwin":
			_ = exec.Command("open", url).Start()
		}
	})
	systray.SetMenu(menu)

	go serveOSC(*server)
	myClock := clock.NewClock()
	job, ok := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if server.startTime > 0 {
			server.Notifier <- []byte("time=" + strconv.FormatInt(server.offset, 10))
			broadcastOsc(server.offset)
			server.offset = makeTimestamp() - server.startTime
		}

		if math.Floor(float64(server.oldOffset)/1000) != math.Floor(float64(server.offset)/1000) {
			systray.SetLabel(fmtDuration(time.Duration(server.offset) * time.Millisecond))
			server.oldOffset = server.offset
		}
	})
	if !ok {
		log.Println("Fail to start timer")
	}
	defer job.Cancel()

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}

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
