package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
)

func createPPPD(pppdInstance *pppdInstance) {
	pppdCmd := exec.Command("pppd", "notty", "file", "/etc/ppp/options.sstpd", "115200")
	pppdIn, err := pppdCmd.StdinPipe()
	handleErr(err)
	pppdOut, err := pppdCmd.StdoutPipe()
	handleErr(err)
	err = pppdCmd.Start()
	handleErr(err)
	pppdInstance.commandInst = pppdCmd
	pppdInstance.stdin = pppdIn
	pppdInstance.stdout = pppdOut
}

func addPPPDResponder(pppdInstance *pppdInstance, conn net.Conn) {
	defer pppdInstance.commandInst.Process.Kill()

	ch := make(chan []byte)
	eCh := make(chan error)

	// Start a goroutine to read from our net connection
	go func(ch chan []byte, eCh chan error, pppdOut io.ReadCloser) {
		for {
			// try to read the data
			data := make([]byte, 512)
			n, err := pppdOut.Read(data)
			log.Printf("pppd: %v bytes read", n)

			if err != nil {
				// send an error if it's encountered
				eCh <- err
				return
			}
			// send data if we read some.
			ch <- data[0:n]
		}
	}(ch, eCh, pppdInstance.stdout)

	//ticker := time.Tick(time.Second)
	// continuously read from the connection
	for {
		select {
		case data := <-ch: // This case means we recieved data on the connection
			// Do something with the data
			//log.Printf("%s\n", hex.Dump(data))
			packets := pppUnescape(data)
			for _, v := range packets {
				fmt.Print(hex.Dump(v))
				sendDataPacket(v, conn)
			}
		case err := <-eCh: // This case means we got an error and the goroutine has finished
			if err == io.EOF {
				log.Print("pppd disconnected")
				// TODO send abort packet
			} else {
				log.Fatalf("pppd: %s\n", err)
				// handle our error then exit for loop
				break
				// This will timeout on the read.
				//case <-ticker:
				// do nothing? this is just so we can time out if we need to.
				// you probably don't even need to have this here unless you want
				// do something specifically on the timeout.
			}
		}
	}
}
