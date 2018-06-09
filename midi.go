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

	LogPrint("Listen to all midi inputs")

	for {

		activeDevices := make(map[string]int)

		midiDefaultInput, err := rtmidi.NewMIDIInDefault()
		if err != nil {
			LogPrint(err)
			return
		}

		portCount, err := midiDefaultInput.PortCount()
		if err != nil {
			LogPrint(err)
			return
		}

		for i := 0; i < portCount; i++ {
			inp, err := midiDefaultInput.PortName(i)
			if err != nil {
				LogPrint(err)
				continue
			}

			activeDevices[inp] = i

			_, ok := midiDevices[inp]
			if ok {
				continue
			}

			LogPrintf("Found midi input : %s (%d)", inp, i)
			midiDevices[inp], err = rtmidi.NewMIDIInDefault()
			if err != nil {
				LogPrint(err)
				continue
			} else {
				if err := midiDevices[inp].OpenPort(i, inp); err != nil {
					log.Fatal(err)
				}
				midiDevices[inp].SetCallback(func(m rtmidi.MIDIIn, msg []byte, t float64) {
					dst := strings.ToUpper(hex.EncodeToString(msg))
					LogPrintf("Received from %s, %s", inp, dst)
					if reStart.Match([]byte(dst)) {
						LogPrint("Received MIDI start event")
						start()
					} else if reStop.Match([]byte(dst)) {
						LogPrint("Received MIDI stop event")
						stop()
					} else if reReset.Match([]byte(dst)) {
						LogPrint("Received MIDI reset event")
						reset()
					}
				})
			}
		}

		for inp, midiIn := range midiDevices {
			_, ok := activeDevices[inp]
			if !ok {
				LogPrintf("Closing input device %s", inp)
				midiIn.Close()
				delete(midiDevices, inp)
			}
		}

		midiDefaultInput.Close()

		time.Sleep(time.Duration(10 * time.Second))

	}

}

func midiOutputStartTest() {
	midiDefaultOutput, err := rtmidi.NewMIDIOutDefault()

	if err != nil {
		log.Print(err)
		return
	}

	outputPort := -1
	portCount, err := midiDefaultOutput.PortCount()
	if err != nil {
		log.Print(err)
		return
	}

	for i := 0; i < portCount; i++ {
		inp, err := midiDefaultOutput.PortName(i)
		if err != nil {
			log.Print(err)
			continue
		}
		if strings.Contains(strings.ToLower(inp), "chronono") {
			outputPort = i
			log.Printf("Found virtual output : %s (%d)", inp, i)
			break
		}

	}

	defer func() {
		log.Print("Closing output device")
		midiDefaultOutput.Close()
	}()

	if outputPort == -1 {
		return
	}

	time.Sleep(time.Duration(500 * time.Millisecond))

	err = midiDefaultOutput.OpenPort(outputPort, "")
	if err != nil {
		log.Print(err)
		return
	}

	data, err := hex.DecodeString("903C7F")
	if err != nil {
		log.Print(err)
		return
	}
	midiDefaultOutput.SendMessage(data)

	time.Sleep(time.Duration(500 * time.Millisecond))

	data, err = hex.DecodeString("903C00")
	if err != nil {
		log.Print(err)
		return
	}
	midiDefaultOutput.SendMessage(data)

}
