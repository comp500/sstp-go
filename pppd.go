package main

import (
	"io"
	"log"
	"net"
	"os/exec"
)

type pppdInstance struct {
	commandInst *exec.Cmd
	stdin       io.WriteCloser
	unescaper   pppUnescaper
}

type packetHandler struct {
	conn net.Conn
}

func (p packetHandler) Write(data []byte) (int, error) {
	packetBytes := packDataPacketFast(data)
	go p.conn.Write(packetBytes)
	return len(data), nil
}

func createPPPD(pppdInstance *pppdInstance, conn net.Conn) {
	pppdCmd := exec.Command("pppd", "notty", "file", "/etc/ppp/options.sstpd", "115200")
	pppdIn, err := pppdCmd.StdinPipe()
	handleErr(err)
	pppdCmd.Stdout = pppdInstance.unescaper
	err = pppdCmd.Start()
	handleErr(err)
	pppdInstance.commandInst = pppdCmd
	pppdInstance.stdin = pppdIn

	go func() {
		defer log.Print("pppd disconnected")
		pppdCmd.Wait()
	}()
}
