package main

import (
	"embed"
	"flag"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/alex023/clock"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {

	var server = NewChronoServer(assets)

	server.host = flag.String("h", server.GetLocalIP(), "network host to serve on")
	server.port = flag.String("p", "8811", "http port to serve on")
	server.osc = flag.String("o", "8812", "osc port to serve on")

	app := application.New(application.Options{
		Name:        "Chronono",
		LogLevel:    slog.LevelInfo,
		Icon:        icon,
		Description: "OSC and HTTP clock control / v2.2",
		Assets: application.AssetOptions{
			Middleware: func(next http.Handler) http.Handler {
				go server.startHTTPListener(next)
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
		Height:        700,
		DisableResize: true,
		URL:           "/",
	})

	systray := app.NewSystemTray()
	systray.SetIcon(icon)
	menu := app.NewMenu()
	url := server.getHTTPUrl()
	menu.Add("Open " + url).OnClick(func(ctx *application.Context) {
		switch runtime.GOOS {
		case "linux":
			_ = exec.Command("xdg-open", url).Start()
		case "windows", "darwin":
			_ = exec.Command("open", url).Start()
		}
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systray.SetMenu(menu)

	go server.oscServe()
	myClock := clock.NewClock()
	job, ok := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		if server.startTime > 0 {
			server.offset = makeTimestamp() - server.startTime
			server.sseBroadcastTime()
			server.oscBroadcastTime()
		}

		if math.Floor(float64(server.oldOffset)/1000) != math.Floor(float64(server.offset)/1000) {
			systray.SetLabel(fmtDuration(time.Duration(server.offset) * time.Millisecond))
		}
		server.oldOffset = server.offset
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
