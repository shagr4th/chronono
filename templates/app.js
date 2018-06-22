var RADIUS = 108;
var CIRCUMFERENCE = 2 * Math.PI * RADIUS;
var ws;
var hours = 0, minutes = 0, seconds = 0;
var blocked = false;

function setControlValue(name, value, max, nows) {
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
    if (!nows && ws)
        ws.send("time=" + 1000 * Math.floor(hours * 3600 + minutes * 60 + seconds));
}

function calculateAngle(x, y) {
    var k = Math.abs(y) / Math.abs(x);
    var angle = Math.atan(k) * 180 / Math.PI;
    if (y * x > 0) {
        angle = 90 - angle + (y < 0 ? 180 : 0);
    } else {
        angle = angle + (y < 0 ? 90 : 270);
    }

    return angle;
}

function registerControl(name) {
    var control = document.getElementById(name + '_control');
    var control_value = document.getElementById(name + '_value');

    control.addEventListener('click', function (event) {
        if (blocked)
            return;
        var deltaX = event.offsetX - 120;
        var deltaY = event.offsetY - 120;

        var value = Math.floor(calculateAngle(deltaY, deltaX) / (name == 'hours' ? 30 : 6));

        setControlValue(name, value, name == 'hours' ? 12 : 60, false);
    });
    control_value.style.strokeDasharray = CIRCUMFERENCE;
    control_value.style.strokeDashoffset = CIRCUMFERENCE;
}

function initWs() {
    var loc = window.location, new_uri;
    if (loc.protocol === "https:") {
        new_uri = "wss:";
    } else {
        new_uri = "ws:";
    }
    new_uri += "//" + loc.host + loc.pathname + "time";

    document.getElementById("info").innerHTML = "Address : " + loc.protocol + "//" + loc.host + loc.pathname;

    ws = new WebSocket(new_uri);
    if (!ws)
        return false;
    ws.onopen = function (evt) {
        console.log("OPEN");
    }
    ws.onclose = function (evt) {
        console.log("CLOSE");
        ws = null;
        showError("Server lost");
    }
    ws.onmessage = function (evt) {
        if (evt.data && evt.data.lastIndexOf('time=', 0) === 0) {
            var timeInMs = parseInt(evt.data.substring(5));
            if (timeInMs >= 0)
                setTime(timeInMs / 1000);
        } else {
            document.getElementById("logs").value += evt.data + "\n";
        }
    }
    ws.onerror = function (evt) {
        showError("ERROR: " + evt.data);
    }
    return true;
}

function showError(title) {
    document.getElementById("info").innerHTML = "<font color='red'>" + title + "</font>";
}

function setTime(time) {
    var h = Math.floor(time / 3600);
    setControlValue('hours', h, 12, true);
    setControlValue('minutes', Math.floor((time - h * 3600) / 60), 60, true);
    setControlValue('seconds', time % 60, 60, true);
}

function setControlStatus(b, name) {
    blocked = b;
    if (blocked)
        document.getElementById(name).setAttribute("disabled", "disabled");
    else
        document.getElementById(name).removeAttribute("disabled");
}

if (initWs()) {
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
    }

    document.getElementById("clear").onclick = function (evt) {
        document.getElementById("logs").value = "";
    }

    registerControl('hours');
    registerControl('minutes');
    registerControl('seconds');

} else {
    showError("Websocket error");
}