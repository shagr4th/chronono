package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

func (server *ChronoServer) oscServe() {
	addr := server.getOSCUrl()
	oscServer := &osc.Server{Addr: addr}

	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Println("Couldn't listen: ", err)
	}
	defer conn.Close()

	log.Printf("Serving OSC on %s", addr)

	for {
		packet, err := oscServer.ReceivePacket(conn)
		if err != nil {
			log.Printf("OSC Server error: " + err.Error())
		}

		if packet != nil {
			switch packet := packet.(type) {
			default:
				log.Printf("OSC : Unknow packet type!")

			case *osc.Message:
				manageOSCMessage(server, packet)

			case *osc.Bundle:
				bundle := packet
				for _, message := range bundle.Messages {
					manageOSCMessage(server, message)
				}
			}
		}
	}
}

func getTimeSkip(message string) int64 {
	/*
		/incX : incrément de X minutes (si X n'est pas défini, 1 minute par défaut)
		/decX : décrement de X minutes (si X n'est pas défini, 1 minute par défaut)
		/incXs : si le message se termine par 's', on gère des secondes et non des minutes
	*/
	timeSkipRegex := regexp.MustCompile("(?:/chronono)?/((inc)|(dec))([0-9]*)(s|m)?")
	match := timeSkipRegex.FindStringSubmatch(message)
	skip := int64(0)
	if len(match) == 6 {
		skip = int64(1)
		if len(match[4]) > 0 {
			skip, _ = strconv.ParseInt(match[4], 10, 64)
		}
		if match[5] != "s" {
			skip = skip * 60
		}
		if match[1] == "dec" {
			skip = -skip
		}
	}
	return skip
}

func (server *ChronoServer) oscInitClients(clients string, remoteaddr string) error {
	if strings.Contains(remoteaddr, ":") {
		remoteaddr = remoteaddr[:strings.Index(remoteaddr, ":")]
	}
	array := strings.Split(clients, ";")
	for _, oscClientAddress := range array {
		parts := strings.Split(oscClientAddress, ":")
		oscPort := 8765
		if len(parts) > 1 {
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				continue
			}
			oscPort = port
		}
		oscClientAddress = remoteaddr + "," + oscClientAddress
		_, ok := server.oscClients[oscClientAddress]
		if !ok {
			server.oscClients[oscClientAddress] = osc.NewClient(parts[0], oscPort)
			msg := osc.NewMessage("/chronono/start")
			msg.Append(true)
			server.oscClients[oscClientAddress].Send(msg)
			server.LogPrintf("Init OSC client: %s", oscClientAddress[(len(remoteaddr)+1):])
		}
	}
	for oscClientAddress := range server.oscClients {
		if strings.HasPrefix(oscClientAddress, remoteaddr) && !slices.Contains(array, oscClientAddress[(len(remoteaddr)+1):]) {
			server.LogPrintf("Removed OSC client: %s", oscClientAddress[(len(remoteaddr)+1):])
			delete(server.oscClients, oscClientAddress)
		}
	}
	return nil
}

func (server *ChronoServer) oscBroadcastTime() {
	oldSeconds := math.Floor(float64(server.oldOffset) / 1000)
	seconds := math.Floor(float64(server.offset) / 1000)

	if oldSeconds != seconds {
		for _, oscClient := range server.oscClients {
			if int32(seconds)/60 != int32(oldSeconds)/60 {
				msg := osc.NewMessage("/chronono/minutes")
				msg.Append(int32(server.offset / 60000))
				oscClient.Send(msg)
			}
			if int32(seconds)%60 != int32(oldSeconds)%60 {
				msg := osc.NewMessage("/chronono/seconds")
				msg.Append(int32((server.offset / 1000) % 60))
				oscClient.Send(msg)
			}
		}
	}
}

func manageOSCMessage(server *ChronoServer, message *osc.Message) {
	log.Printf("Received OSC message : " + message.String())
	startMsg, _ := regexp.MatchString("/chronono(_|/)start.*", message.String())
	stopMsg, _ := regexp.MatchString("/chronono(_|/)st((op)|(art.*0)|(art.*false))", message.String())
	resetMsg, _ := regexp.MatchString("/chronono(_|/)reset.*", message.String())
	minutesMsg, _ := regexp.MatchString("/chronono(_|/)minutes", message.String())
	secondsMsg, _ := regexp.MatchString("/chronono(_|/)seconds", message.String())
	if stopMsg {
		server.stopTimer()
		server.sseBroadcastTime()
	} else if startMsg {
		server.startTimer()
		server.sseBroadcastTime()
	} else if resetMsg {
		server.resetTimer(0)
		server.sseBroadcastTime()
	} else if (minutesMsg || secondsMsg) && message.CountArguments() == 1 {
		newTime := int64(0)
		s := fmt.Sprintf("%v", message.Arguments[0])
		newTime, _ = strconv.ParseInt(s, 10, 64)
		actual := server.offset / 1000
		if minutesMsg {
			newTime = newTime*60 + actual%60
		} else {
			newTime = newTime + int64(actual/60)*60
		}
		server.resetTimer(newTime * 1000)
		server.sseBroadcastTime()
	} else {
		var timeSkip = getTimeSkip(message.Address)
		if timeSkip != 0 && server.startTime == 0 {
			server.resetTimer(server.offset + timeSkip*1000)
			server.sseBroadcastTime()
		}
	}
}
