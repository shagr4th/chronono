package main

import (
	"encoding/hex"
	"flag"
	"github.com/alex023/clock"
	"github.com/getlantern/systray"
	"github.com/gorilla/websocket"
	"github.com/thestk/rtmidi/contrib/go/rtmidi"
	"html/template"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
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
		var delta int64 = -1
		if startTime > 0 {
			delta = makeTimestamp() - startTime + offset*1000
		}
		var msg = []byte("time=" + strconv.Itoa(int(delta)))
		err = c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			log.Println("Websocket client write error :", err)
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
		if s == "clear" {
			offset = 0
			log.Print("Clear defaults")
		} else if strings.HasPrefix(s, "start=") {
			s = strings.TrimPrefix(s, "start=")
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Println("Convert error :", err)
				break
			}
			offset = i
			start()
		} else if s == "stop" {
			stop()
		}

	}
	log.Printf("Closed connection: %s", r.RemoteAddr)
}

func start() {
	if startTime == 0 {
		startTime = makeTimestamp()
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
func midiDevicesScan(midistartcode *string, midistopcode *string) {

	var midiActiveDevices = make(map[string]rtmidi.MIDIIn)

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
					if strings.HasPrefix(dst, *midistartcode) {
						log.Print("Received MIDI start event")
						start()
					}
					if strings.HasPrefix(dst, *midistopcode) {
						log.Print("Received MIDI stop event")
						stop()
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

func onReady() {

	systray.SetIcon(MyArray)
	systray.SetTooltip("Chronono")
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	go func() {
		<-mQuit.ClickedCh
		log.Println("Requesting quit")
		systray.Quit()
		log.Println("Finished quitting")
	}()

	port := flag.String("p", "8811", "http port to serve on")
	midistartcode := flag.String("midistartcode", "FA", "MIDI pattern for clock start")
	midistopcode := flag.String("midistopcode", "FC", "MIDI pattern for clock stop")
	flag.Parse()

	http.HandleFunc("/time", timeMsg)
	http.HandleFunc("/", home)
	go func() {
		log.Printf("Serving on http://%s\n", localIP+":"+*port)
		log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))
	}()
	go midiDevicesScan(midistartcode, midistopcode)
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
		stroke: #606060;
	}
	
	.progress__value {
		stroke: #3893AE;
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
	
	.button:hover {
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
	<div class="clockdiv">
		 
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="hours_value"/>
			<text transform="rotate(90, 19, 55)" font-family="Courier" font-size="140" fill="#808080" id="hours_text">00</text>
		</svg>
		<br/><h3>Hours</h3>
		<input id="hours_control" class="control" type="range" min="0" max="11" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="minutes_value"/>
			<text transform="rotate(90, 19, 55)" font-family="Courier" font-size="140" fill="#808080" id="minutes_text">00</text>
		</svg>
		<br/><h3>Minutes</h3>
		<input id="minutes_control" class="control" type="range" min="0" max="59" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="seconds_value"/>
			<text transform="rotate(90, 19, 55)" font-family="Courier" font-size="140" fill="#808080" id="seconds_text">00</text>
		</svg>
		<br/><h3>Seconds</h3>
		<input id="seconds_control" class="control" type="range" min="0" max="59" value="0" />
	</div>

	<h3 id="info">Address : {{.HTTPLocation}}</h3>
</div>

<div class="clock">

	<input id="start" type="button" class="button" value="Start" />
	<input id="stop"  type="button" class="button" value="Stop" />
	<input id="clear" type="button" class="button" value="Clear" />

</div>
	
<script>
	
	var RADIUS = 108;
	var CIRCUMFERENCE = 2 * Math.PI * RADIUS;
	var ws;
	var hours = 0, minutes = 0, seconds = 0;

	function setControlValue(name, value) {
		var control = document.getElementById(name + '_control');
		var control_value = document.getElementById(name + '_value');
		var control_text_value = document.getElementById(name + '_text');

		var max = parseInt(control.max) + 1;
		var progress = value / max;
		var dashoffset = CIRCUMFERENCE * (1 - progress);
			
		control_value.style.strokeDashoffset = dashoffset;
		control_text_value.textContent = ('0' + Math.floor(value)).slice(-2);
		control.value = value;
		if (name == 'hours')
			hours = value;
		if (name == 'minutes')
			minutes = value;
		if (name == 'seconds')
			seconds = value;
	}
	
	function registerControl(name) {
		var control = document.getElementById(name + '_control');
		var control_value = document.getElementById(name + '_value');
		
		control.addEventListener('input', function(event) {
			setControlValue(name, event.target.valueAsNumber);
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
		setControlValue('hours', h);
		setControlValue('minutes', Math.floor((time - h * 3600) / 60));
		setControlValue('seconds', time % 60);
	}

	if (initWs("{{.WSLocation}}")) {
		console.log("Websocket initialized");

		document.getElementById("start").onclick = function (evt) {
			if (ws)
				ws.send("start="+Math.floor(hours*3600+minutes*60+seconds));
		}
	
		document.getElementById("stop").onclick = function (evt) {
			if (ws)
				ws.send("stop");
		}

		document.getElementById("clear").onclick = function (evt) {
			if (ws)
				ws.send("clear");
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
