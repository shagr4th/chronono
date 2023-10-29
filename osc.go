package main

import (
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

var oscClients map[string]*osc.Client = make(map[string]*osc.Client)

func serveOSC(host string, port string) {
	addr := host + ":" + port
	server := &osc.Server{Addr: addr}

	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		log.Println("Couldn't listen: ", err)
	}
	defer conn.Close()

	log.Printf("Serving OSC on %s", addr)

	for {
		packet, err := server.ReceivePacket(conn)
		if err != nil {
			LogPrint("OSC Server error: " + err.Error())
		}

		if packet != nil {
			switch packet := packet.(type) {
			default:
				LogPrint("OSC : Unknow packet type!")

			case *osc.Message:
				manageOSCMessage(packet)

			case *osc.Bundle:
				bundle := packet
				for _, message := range bundle.Messages {
					manageOSCMessage(message)
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
	timeSkipRegex := regexp.MustCompile("/((inc)|(dec))([0-9]*)(s|m)?")
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

func initOscClients(clients string) error {
	for _, oscClient := range strings.Split(clients, ";") {
		oscClientHost := oscClient
		//oscClients[oscClient] = osc.NewClient(oscClientHost, 8765)
		//msg := osc.NewMessage("/chronono/init")
		//msg.Append(true)
		//oscClients[oscClient].Send(msg)
		LogPrintf("init OSC client: %s", oscClientHost)
	}
	return nil
}

func broadcastOsc(millis int64) {
	for _, oscClient := range oscClients {
		msg := osc.NewMessage("/osc/minutes")
		msg.Append(int32(millis / 60000))
		oscClient.Send(msg)
		msg = osc.NewMessage("/osc/seconds")
		msg.Append(int32((millis / 1000) % 60))
		oscClient.Send(msg)
	}
}

func manageOSCMessage(message *osc.Message) {
	LogPrint("Received OSC message : " + message.String())
	startMsg, _ := regexp.MatchString("/chronono_start.*(1)|(true)", message.String())
	stopMsg, _ := regexp.MatchString("/chronono_st(op)|(art.*0)|(art.*false)", message.String())
	resetMsg, _ := regexp.MatchString("/chronono_reset.*", message.String())
	if startMsg {
		start()
	} else if stopMsg {
		stop()
	} else if resetMsg {
		reset(0)
	} else {
		var timeSkip = getTimeSkip(message.Address)
		if timeSkip != 0 {
			incrementTime(timeSkip)
		}
	}
}
