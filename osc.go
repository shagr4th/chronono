package main

import (
	"github.com/hypebeast/go-osc/osc"
	"log"
	"net"
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
			switch packet.(type) {
			default:
				LogPrint("OSC : Unknow packet type!")

			case *osc.Message:
				manageOSCMessage(packet.(*osc.Message))

			case *osc.Bundle:
				bundle := packet.(*osc.Bundle)
				for _, message := range bundle.Messages {
					manageOSCMessage(message)
				}
			}
		}
	}
}

func manageOSCMessage(message *osc.Message) {
	LogPrint("Received OSC message : " + message.String())
	if message.String() == "/chronono_start ,f 1" {
		start()
	}
	if message.Address == "/chronono_stop" || message.String() == "/chronono_start ,f 0" {
		stop()
	}
	if message.Address == "/chronono_reset" {
		reset()
	}
}
