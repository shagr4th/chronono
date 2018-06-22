package main

import (
	"fmt"
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

	log.Printf("Serving HTTP on %s", addr)

	for {
		packet, err := server.ReceivePacket(conn)
		if err != nil {
			log.Println("Server error: " + err.Error())
		}

		if packet != nil {
			switch packet.(type) {
			default:
				log.Println("Unknow packet type!")

			case *osc.Message:
				osc.PrintMessage(packet.(*osc.Message))

			case *osc.Bundle:
				bundle := packet.(*osc.Bundle)
				for _, message := range bundle.Messages {
					LogPrint("OSC received : " + message.Address)
					log.Print(message.Arguments[0])
					if message.Address == "/chronono_start" && fmt.Sprintf("%v", message.Arguments[0]) == "1" {
						start()
					}
					if message.Address == "/chronono_stop" || (message.Address == "/chronono_start" && fmt.Sprintf("%v", message.Arguments[0]) == "0") {
						stop()
					}
					if message.Address == "/chronono_reset" {
						reset()
					}
				}
			}
		}
	}
}
