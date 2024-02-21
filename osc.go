package main

import (
	"log"
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
			msg := osc.NewMessage("/chronono/init")
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
	for _, oscClient := range server.oscClients {
		msg := osc.NewMessage("/chronono/minutes")
		msg.Append(int32(server.offset / 60000))
		oscClient.Send(msg)
		msg = osc.NewMessage("/chronono/seconds")
		msg.Append(int32((server.offset / 1000) % 60))
		oscClient.Send(msg)
	}
}

func manageOSCMessage(server *ChronoServer, message *osc.Message) {
	log.Printf("Received OSC message : " + message.String())
	startMsg, _ := regexp.MatchString("/chronono_start.*(1)|(true)", message.String())
	stopMsg, _ := regexp.MatchString("/chronono_st(op)|(art.*0)|(art.*false)", message.String())
	resetMsg, _ := regexp.MatchString("/chronono_reset.*", message.String())
	if startMsg {
		server.startTimer()
	} else if stopMsg {
		server.stopTimer()
	} else if resetMsg {
		server.resetTimer(0)
	} else {
		var timeSkip = getTimeSkip(message.Address)
		if timeSkip != 0 {
			server.incrementTime(timeSkip)
		}
	}
}
