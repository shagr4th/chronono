package main

import (
	"flag"
	"github.com/alex023/clock"
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

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Printf("New connection: %s", r.Host)
	defer c.Close()
	job, inserted := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		var delta int64 = -1
		if startTime > 0 {
			delta = makeTimestamp() - startTime + offset*1000
		}
		var ss = []byte(strconv.Itoa(int(delta)))
		err = c.WriteMessage(websocket.TextMessage, ss)
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
	log.Printf("Closed connection: %s", r.Host)
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
		WSLocation:   "ws://" + r.Host + "/echo",
		HTTPLocation: "http://" + r.Host,
	}
	homeTemplate.Execute(w, data)
}

func midiEventScan(deviceName string, portNumber int) {
	log.Printf("Listening to Midi device : %s\n", deviceName)
	midiDeviceInput, err := rtmidi.NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}

	defer midiDeviceInput.Destroy()
	if err := midiDeviceInput.OpenPort(portNumber, "RtMidi"); err != nil {
		log.Fatal(err)
		log.Printf("Disconnected MIDI Device %s", deviceName)
	}
	defer midiDeviceInput.Close()

	for {
		m, t, err := midiDeviceInput.Message()
		if len(m) > 0 && m[0] != 224 {
			log.Println(deviceName, m, t, err)
			if m[0] == 128 && m[1] == 36 {
				log.Print("Received C1 on MIDI Channel 1")
				start()
			}
		}
	}

}

func midiDevicesScan() {
	midiInput, err := rtmidi.NewMIDIInDefault()
	if err != nil {
		log.Fatal(err)
	}

	portCount, err := midiInput.PortCount()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < portCount; i++ {
		inp, err := midiInput.PortName(i)

		//_ = err
		if err != nil {
			log.Fatal(err)
		}

		go midiEventScan(inp, i)
	}

	defer midiInput.Close()

}

func main() {

	port := flag.String("p", "8811", "port to serve on")
	_ = port
	flag.Parse()

	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	go func() {
		log.Printf("Serving on http://%s\n", localIP+":"+*port)
		log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))
	}()
	midiDevicesScan()
	select {}
}

var homeTemplate = template.Must(template.New("").Parse(`
<style>

  body {
	  background: black;
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
		stroke: #46b8da;
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
		background-color: #46b8da;
		color: black;
		border: 2px solid #A0A0A0;
		padding: 15px 32px;
		text-align: center;
		text-decoration: none;
		display: inline-block;
		font-size: 16px;
		margin: 20px 10px;
		cursor: pointer;
	}
	
	.button:hover {
		background-color: #86d0e7;
		color: white;
	}

	h1 {
		text-shadow: 0 0 10px #46b8da;
		font-size: 60px;
	}
	
</style>
	
<div class="clock">
	
	<h1>Chronono</h1>
	<h2 id="info">{{.HTTPLocation}}</h2>
	<div class="clockdiv">
		 
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="hours_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" fill="#808080" id="hours_text">00</text>
		</svg>
		<br/><h3>Hours</h3>
		<input id="hours_control" class="control" type="range" min="0" max="11" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="minutes_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" fill="#808080" id="minutes_text">00</text>
		</svg>
		<br/><h3>Minutes</h3>
		<input id="minutes_control" class="control" type="range" min="0" max="59" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="seconds_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" fill="#808080" id="seconds_text">00</text>
		</svg>
		<br/><h3>Seconds</h3>
		<input id="seconds_control" class="control" type="range" min="0" max="59" value="0" />
	</div>

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
			var timeInMs = parseInt(evt.data);
			if (timeInMs >= 0)
				setTime(timeInMs/1000);
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
