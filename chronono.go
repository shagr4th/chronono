package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/alex023/clock"
	"github.com/getlantern/systray"
	"github.com/gorilla/websocket"
	"github.com/thestk/rtmidi/contrib/go/rtmidi"
	"html/template"
	"log"
	"math"
	"net"
	"net/http"
	"os/exec"
	"regexp"
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

func home(w http.ResponseWriter, r *http.Request) {
	data := struct {
		WSLocation   string
		HTTPLocation string
	}{
		WSLocation:   "ws://" + r.Host + "/time",
		HTTPLocation: "http://" + r.Host,
	}
	homeTemplate.Execute(w, data)
}

// http://midi.teragonaudio.com/tech/midispec.htm

// Scan midi devices
// TODO: detect disconnections and release appropriate objects (not sure if RtMidi is able to do it)
func midiDevicesScan(midistart *string, midistop *string, midireset *string) {

	var midiActiveDevices = make(map[string]rtmidi.MIDIIn)
	reStart, _ := regexp.Compile("(?i)" + *midistart)
	reStop, _ := regexp.Compile("(?i)" + *midistop)
	reReset, _ := regexp.Compile("(?i)" + *midireset)

	midiDefaultInput, err := rtmidi.NewMIDIInDefault()
	if err != nil {
		log.Print(err)
		return
	}
	defer midiDefaultInput.Close()

	for {

		portCount, err := midiDefaultInput.PortCount()
		if err != nil {
			log.Print(err)
			return
		}

		for i := 0; i < portCount; i++ {
			inp, err := midiDefaultInput.PortName(i)
			if err != nil {
				log.Print(err)
				continue
			}

			_, ok := midiActiveDevices[inp]
			if ok {
				continue
			}

			log.Printf("Found new Midi device : %s\n", inp)
			midiActiveDevices[inp], err = rtmidi.NewMIDIInDefault()
			if err != nil {
				log.Print(err)
				continue
			} else {
				if err := midiActiveDevices[inp].OpenPort(i, inp); err != nil {
					log.Fatal(err)
				}
				midiActiveDevices[inp].SetCallback(func(m rtmidi.MIDIIn, msg []byte, t float64) {
					dst := strings.ToUpper(hex.EncodeToString(msg))
					log.Println(dst)
					if reStart.Match([]byte(dst)) {
						log.Print("Received MIDI start event")
						start()
					} else if reStop.Match([]byte(dst)) {
						log.Print("Received MIDI stop event")
						stop()
					} else if reReset.Match([]byte(dst)) {
						log.Print("Received MIDI reset event")
						reset()
					}
				})
			}
		}

		time.Sleep(time.Duration(10 * time.Second))

	}

}

func main() {

	systray.Run(onReady, onExit)

}

func showSystray() {
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

func onReady() {

	port := flag.String("p", "8811", "http port to serve on")
	midistart := flag.String("midistart", "(BF7F7F)|(FA).*", "MIDI regex for clock start")
	midistop := flag.String("midistop", "(BF7F00)|(FC).*", "MIDI regex for clock stop")
	midireset := flag.String("midireset", "FF.*", "MIDI regex for clock reset")
	flag.Parse()

	var url = "http://" + localIP + ":" + *port

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

	http.HandleFunc("/time", timeMsg)
	http.HandleFunc("/", home)
	go func() {
		log.Printf("Serving on %s\n", url)
		log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))
	}()
	go midiDevicesScan(midistart, midistop, midireset)
	go showSystray()
	select {}
}

func onExit() {
	// clean up here
}

var homeTemplate = template.Must(template.New("").Parse(`
<style>

  body {
	  background: #303030;
	  color: gray;
  }
  
	.progress {
		transform: rotate(-90deg);
	}
	
	.progress__meter,
	.progress__value {
		fill: none;
	}
	
	.progress__meter {
		stroke: #404040;
	}
	
	.progress__value {
		stroke: #38738E;
		stroke-linecap: round;
	}
	
	.clock {
		display: block;
		margin: auto;
		text-align: center;
		font-family: Helvetica;
	}
	
	.clockdiv{
		display: inline-block;
		text-align: center;
	}

	.control {
		width: 200px
	}

	.button {
		background-color: black;
		color: white;
		border: 0px solid #A0A0A0;
		padding: 15px 32px;
		text-align: center;
		text-decoration: none;
		display: inline-block;
		font-size: 16px;
		margin: 20px 10px;
		cursor: pointer;
	}
	
	.button:hover:not([disabled]) {
		background-color: lightgray;
		color: black;
	}

	.button:focus {outline:0;}

	.title {
		text-shadow: 0 0 20px black;
		color: lightgray;
		font-size: 40px;
	}
	
</style>
	
<div class="clock">
	
	<h2 class="title">chronono</h2>
	<div class="clockdiv" id="hours_control" >

		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24"/>
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="hours_value"/>
			<text x="40" y="-75" transform="rotate(90, 0, 0)" font-family="Courier" font-size="140" letter-spacing="-10" fill="#A0A0A0" id="hours_text">00</text>
		</svg>

	</div>

	<div class="clockdiv" id="minutes_control">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24"/>
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="minutes_value"/>
			<text x="40" y="-75" transform="rotate(90, 0, 0)" font-family="Courier" font-size="140" letter-spacing="-10" fill="#A0A0A0" id="minutes_text">00</text>
		</svg>

	</div>

	<div class="clockdiv" id="seconds_control">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24"/>
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="seconds_value"/>
			<text x="40" y="-75" transform="rotate(90, 0, 0)" font-family="Courier" font-size="140" letter-spacing="-10" fill="#A0A0A0" id="seconds_text">00</text>
		</svg>

	</div>

	<h3 id="info">Address : {{.HTTPLocation}}</h3>
</div>

<div class="clock">

	<input id="start" type="button" class="button" value="Start" />
	<input id="stop"  type="button" class="button" value="Stop" />
	<input id="reset" type="button" class="button" value="Reset" />

</div>
	
<script>
	
	var RADIUS = 108;
	var CIRCUMFERENCE = 2 * Math.PI * RADIUS;
	var ws;
	var hours = 0, minutes = 0, seconds = 0;
	var blocked = false;

	function setControlValue(name, value, max) {
		var control_value = document.getElementById(name + '_value');
		var control_text_value = document.getElementById(name + '_text');

		var progress = value / max;
		var dashoffset = CIRCUMFERENCE * (1 - progress);
			
		control_value.style.strokeDashoffset = dashoffset;
		control_text_value.textContent = ('0' + Math.floor(value)).slice(-2);
		if (name == 'hours')
			hours = value;
		if (name == 'minutes')
			minutes = value;
		if (name == 'seconds')
			seconds = value;
		if (ws)
			ws.send("time="+1000*Math.floor(hours*3600+minutes*60+seconds));
	}

	function calculateAngle(x, y) {
		var k = Math.abs(y) / Math.abs(x);
		var angle = Math.atan(k) * 180 / Math.PI;
		if(y * x > 0){
			angle = 90 - angle + (y < 0 ? 180 : 0);
		} else {
		  angle = angle + (y < 0 ? 90 : 270);
		}
		
		return angle;
	  }
	
	function registerControl(name) {
		var control = document.getElementById(name + '_control');
		var control_value = document.getElementById(name + '_value');
		
		control.addEventListener('click', function(event) {
			if (blocked)
				return;
			var deltaX = event.offsetX - 120;
			var deltaY = event.offsetY - 120;
			
			var value = Math.floor(calculateAngle(deltaY, deltaX) / (name == 'hours' ? 30 : 6));

			setControlValue(name, value, name == 'hours' ? 12 : 60);
		});
		control_value.style.strokeDasharray = CIRCUMFERENCE;
		control_value.style.strokeDashoffset = CIRCUMFERENCE;
	}
	
	function initWs(url) {
		ws = new WebSocket(url);
		if (!ws)
			return false;
		ws.onopen = function(evt) {
			console.log("OPEN");
		}
		ws.onclose = function(evt) {
			console.log("CLOSE");
			ws = null;
			showError("Server lost");
		}
		ws.onmessage = function(evt) {
			if (evt.data && evt.data.lastIndexOf('time=', 0) === 0) {
				var timeInMs = parseInt(evt.data.substring(5));
				if (timeInMs >= 0)
					setTime(timeInMs/1000);
			}
		}
		ws.onerror = function(evt) {
			showError("ERROR: " + evt.data);
		}
		return true;
	}

	function showError(title) {
		document.getElementById("info").innerHTML = "<font color='red'>" + title + "</font>";
	}

	function setTime(time) {
		var h = Math.floor(time / 3600);
		setControlValue('hours', h, 12);
		setControlValue('minutes', Math.floor((time - h * 3600) / 60), 60);
		setControlValue('seconds', time % 60, 60);
	}

	function setControlStatus(b, name) {
		blocked = b;
		if (blocked)
			document.getElementById(name).setAttribute("disabled","disabled");
		else
			document.getElementById(name).removeAttribute("disabled");
	}

	if (initWs("{{.WSLocation}}")) {
		console.log("Websocket initialized");

		document.getElementById("start").onclick = function (evt) {
			if (ws)
				ws.send("start");
		}
	
		document.getElementById("stop").onclick = function (evt) {
			if (ws)
				ws.send("stop");
		}

		document.getElementById("reset").onclick = function (evt) {
			if (ws)
				ws.send("reset");
			setTime(0);
		}

		registerControl('hours');
		registerControl('minutes');
		registerControl('seconds');
	
	} else {
		showError("Websocket error");
	}
	

</script>
`))
