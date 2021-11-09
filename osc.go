package main

import (
	"log"
	"net"
	"regexp"
	"strings"

	"github.com/hypebeast/go-osc/osc"
)

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
	} else if strings.HasPrefix(message.Address, "/inc10") {
		incrementTime(600)
	} else if strings.HasPrefix(message.Address, "/inc") {
		incrementTime(60)
	} else if strings.HasPrefix(message.Address, "/dec10") {
		incrementTime(-600)
	} else if strings.HasPrefix(message.Address, "/dec") {
		incrementTime(-60)
	}
}
