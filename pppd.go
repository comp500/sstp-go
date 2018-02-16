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
	sendDataPacket(data, p.conn)
	return len(data), nil
}

func createPPPD(pppdInstance *pppdInstance, conn net.Conn) {
	pr, pw := io.Pipe()
	pppdCmd := exec.Command("pppd", "notty", "file", "/etc/ppp/options.sstpd", "115200")
	pppdIn, err := pppdCmd.StdinPipe()
	handleErr(err)
	pppdCmd.Stdout = pw
	err = pppdCmd.Start()
	handleErr(err)
	pppdInstance.commandInst = pppdCmd
	pppdInstance.stdin = pppdIn

	go func() {
		defer pppdInstance.commandInst.Process.Kill()
		defer pr.Close()
		defer log.Print("error reading")
		_, err := io.Copy(pppdInstance.unescaper, pr)
		if err != nil {
			log.Print(err)
		}
	}()

	go func() {
		defer pw.Close()
		defer log.Print("pppd disconnected")
		pppdCmd.Wait()
	}()
}
