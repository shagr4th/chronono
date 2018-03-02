package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net"
	"net/http"
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

var upgrader = websocket.Upgrader{} // use default options

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	log.Printf("New connection: %s", r.Host)
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
	log.Printf("Closed connection: %s", r.Host)
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

var localIP = GetLocalIP()

func main() {

	port := flag.String("p", "8811", "port to serve on")
	_ = port
	flag.Parse()

	log.Printf("Serving on http://%s\n", localIP+":"+*port)

	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(localIP+":"+*port, nil))

}

var homeTemplate = template.Must(template.New("").Parse(`
<style>

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
		<br/><h3>Heures</h3>
		<input id="hours_control" type="range" min="0" max="12" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="minutes_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" id="minutes_text">00</text>
		</svg>
		<br/><h3>Minutes</h3>
		<input id="minutes_control" type="range" min="0" max="59" value="0" />
	</div>

	<div class="clockdiv">
		
		<svg class="progress" width="240" height="240" viewBox="0 0 240 240">
			<circle class="progress__meter" cx="120" cy="120" r="108" stroke-width="24" />
			<circle class="progress__value" cx="120" cy="120" r="108" stroke-width="24" id="seconds_value"/>
			<text transform="rotate(90, 12, 70)" font-family="Helvetica" font-size="120" id="seconds_text">00</text>
		</svg>
		<br/><h3>Secondes</h3>
		<input id="seconds_control" type="range" min="0" max="59" value="0" />
	</div>

</div>
	
<script>
	
	var RADIUS = 108;
	var CIRCUMFERENCE = 2 * Math.PI * RADIUS;
	var ws;
	
	function registerControl(name) {
		var control = document.getElementById(name + '_control');
		var control_value = document.getElementById(name + '_value');
		var control_text_value = document.getElementById(name + '_text');
		
		control.addEventListener('input', function(event) {
			var value = event.target.valueAsNumber;
			var max = parseInt(control.max) + 1;
			var progress = value / max;
			var dashoffset = CIRCUMFERENCE * (1 - progress);
					
			control_value.style.strokeDashoffset = dashoffset;
			control_text_value.textContent = ('0' + value).slice(-2);
			if (ws)
				ws.send(name + "=" + value);
		});
		control_value.style.strokeDasharray = CIRCUMFERENCE;
		control_value.style.strokeDashoffset = CIRCUMFERENCE;
	}
	
	registerControl('hours');
	registerControl('minutes');
	registerControl('seconds');

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
		}
		ws.onmessage = function(evt) {
			console.log("RESPONSE: " + evt.data);
		}
		ws.onerror = function(evt) {
			console.log("ERROR: " + evt.data);
		}
		return true;
	}

	function showError(title) {
		document.getElementById("info").innerHTML = "<font color='red'>" + title + "</font>";
	}

	if (initWs("{{.WSLocation}}")) {
		console.log("Websocket initialized");
	} else {
		showError("Websocket error");
	}
	

</script>
`))
