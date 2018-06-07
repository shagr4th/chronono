package main

import (
	"encoding/hex"
	"github.com/thestk/rtmidi/contrib/go/rtmidi"
	"log"
	"regexp"
	"strings"
	"time"
)

// http://midi.teragonaudio.com/tech/midispec.htm

// Scan midi devices
func midiDevicesScan(midistart *string, midistop *string, midireset *string) {

	var midiDevices = make(map[string]rtmidi.MIDIIn)
	reStart, _ := regexp.Compile("(?i)" + *midistart)
	reStop, _ := regexp.Compile("(?i)" + *midistop)
	reReset, _ := regexp.Compile("(?i)" + *midireset)

	log.Print("Listen to all midi inputs")

	for {

		activeDevices := make(map[string]int)

		midiDefaultInput, err := rtmidi.NewMIDIInDefault()
		if err != nil {
			log.Print(err)
			return
		}

		portCount, err := midiDefaultInput.PortCount()
		if err != nil {
			log.Print(err)
			return
		}

		for i := 0; i < portCount; i++ {
			inp, err := midiDefaultInput.PortName(i)
			if err != nil {
				log.Print(err)
				continue
			}

			activeDevices[inp] = i

			_, ok := midiDevices[inp]
			if ok {
				continue
			}

			log.Printf("Found midi input : %s (%d)\n", inp, i)
			midiDevices[inp], err = rtmidi.NewMIDIInDefault()
			if err != nil {
				log.Print(err)
				continue
			} else {
				if err := midiDevices[inp].OpenPort(i, inp); err != nil {
					log.Fatal(err)
				}
				midiDevices[inp].SetCallback(func(m rtmidi.MIDIIn, msg []byte, t float64) {
					dst := strings.ToUpper(hex.EncodeToString(msg))
					log.Printf("Received from %s, %s", inp, dst)
					if reStart.Match([]byte(dst)) {
						log.Print("Received MIDI start event")
						start()
					} else if reStop.Match([]byte(dst)) {
						log.Print("Received MIDI stop event")
						stop()
					} else if reReset.Match([]byte(dst)) {
						log.Print("Received MIDI reset event")
						reset()
					}
				})
			}
		}

		for inp, midiIn := range midiDevices {
			_, ok := activeDevices[inp]
			if !ok {
				log.Printf("Closing input device %s", inp)
				midiIn.Close()
				delete(midiDevices, inp)
			}
		}

		midiDefaultInput.Close()

		time.Sleep(time.Duration(10 * time.Second))

	}

}
