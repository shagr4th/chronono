package main

import (
	"flag"
	"github.com/alex023/clock"
	"github.com/gorilla/websocket"
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
var startedJob clock.Job
var startTime int64
var localIP = GetLocalIP()
var offset int64

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Printf("New connection: %s", r.Host)
	defer c.Close()
	var myClock = clock.NewClock()
	job, inserted := myClock.AddJobRepeat(time.Duration(100*time.Millisecond), 0, func() {
		var delta int64 = -1
		if startTime > 0 {
			delta = makeTimestamp() - startTime + offset*1000
		}
		var ss = []byte(strconv.Itoa(int(delta)))
		err = c.WriteMessage(websocket.TextMessage, ss)
		if err != nil {
			log.Println("write:", err)
		}
	})
	if !inserted {
		log.Println("failure")
	}
	defer job.Cancel()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
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
	startTime = makeTimestamp()
}

func stop() {
	startTime = 0
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
	select {}
}

var homeTemplate = template.Must(template.New("").Parse(`
<style>

input[type=range] {
	height: 25px;
	-webkit-appearance: none;
	margin: 10px 0;
	width: 100%;
  }
  input[type=range]:focus {
	outline: none;
  }
  input[type=range]::-webkit-slider-runnable-track {
	width: 100%;
	height: 5px;
	cursor: pointer;
	animate: 0.2s;
	box-shadow: 0px 0px 0px #000000;
	background: #2497E3;
	border-radius: 1px;
	border: 0px solid #000000;
  }
  input[type=range]::-webkit-slider-thumb {
	box-shadow: 0px 0px 0px #000000;
	border: 1px solid #2497E3;
	height: 18px;
	width: 18px;
	border-radius: 25px;
	background: #A1D0FF;
	cursor: pointer;
	-webkit-appearance: none;
	margin-top: -7px;
  }
  input[type=range]:focus::-webkit-slider-runnable-track {
	background: #2497E3;
  }
  input[type=range]::-moz-range-track {
	width: 100%;
	height: 5px;
	cursor: pointer;
	animate: 0.2s;
	box-shadow: 0px 0px 0px #000000;
	background: #2497E3;
	border-radius: 1px;
	border: 0px solid #000000;
  }
  input[type=range]::-moz-range-thumb {
	box-shadow: 0px 0px 0px #000000;
	border: 1px solid #2497E3;
	height: 18px;
	width: 18px;
	border-radius: 25px;
	background: #A1D0FF;
	cursor: pointer;
  }
  input[type=range]::-ms-track {
	width: 100%;
	height: 5px;
	cursor: pointer;
	animate: 0.2s;
	background: transparent;
	border-color: transparent;
	color: transparent;
  }
  input[type=range]::-ms-fill-lower {
	background: #2497E3;
	border: 0px solid #000000;
	border-radius: 2px;
	box-shadow: 0px 0px 0px #000000;
  }
  input[type=range]::-ms-fill-upper {
	background: #2497E3;
	border: 0px solid #000000;
	border-radius: 2px;
	box-shadow: 0px 0px 0px #000000;
  }
  input[type=range]::-ms-thumb {
	margin-top: 1px;
	box-shadow: 0px 0px 0px #000000;
	border: 1px solid #2497E3;
	height: 18px;
	width: 18px;
	border-radius: 25px;
	background: #A1D0FF;
	cursor: pointer;
  }
  input[type=range]:focus::-ms-fill-lower {
	background: #2497E3;
  }
  input[type=range]:focus::-ms-fill-upper {
	background: #2497E3;
  }
  
	.progress {
		transform: rotate(-90deg);
	}
	
	.progress__meter,
	.progress__value {
		fill: none;
	}
	
	.progress__meter {
		stroke: #e6e6e6;
	}
	
	.progress__value {
		stroke: #f77a52;
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
		background-color: #005581;
		color: white;
		border: 2px solid #555555;
		padding: 15px 32px;
		text-align: center;
		text-decoration: none;
		display: inline-block;
		font-size: 16px;
		margin: 20px 10px;
		cursor: pointer;
	}
	
	.button:hover {
		background-color: rgb(137, 208, 230);
		color: black;
	}
	
</style>
	
<div class="clock">
	
	<h1>Chronono</h1>
	<h2 id="info">{{.HTTPLocation}}</h2>
	<div class="clockdiv">
		 
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="hours_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" id="hours_text">00</text>
		</svg>
		<br/><h3>Hours</h3>
		<input id="hours_control" class="control" type="range" min="0" max="11" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="minutes_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" id="minutes_text">00</text>
		</svg>
		<br/><h3>Minutes</h3>
		<input id="minutes_control" class="control" type="range" min="0" max="59" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="seconds_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" id="seconds_text">00</text>
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
		setControlValue('minutes', Math.floor((time - h) / 60));
		setControlValue('seconds', time % 60);
	}

	if (initWs("{{.WSLocation}}")) {
		console.log("Websocket initialized");

		document.getElementById("start").onclick = function (evt) {
			if (ws)
				ws.send("start="+(hours*3600+minutes*60+seconds));
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
