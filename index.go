package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

func handleErr(err error) {
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

// MessageType is the type of message this packet is
type MessageType uint16

// Constants for MessageType values
const (
	MessageTypeCallConnectRequest = 1
	MessageTypeCallConnectAck     = 2
	MessageTypeCallConnectNak     = 3
	MessageTypeCallConnected      = 4
	MessageTypeCallAbort          = 5
	MessageTypeCallDisconnect     = 6
	MessageTypeCallDisconnectAck  = 7
	MessageTypeEchoRequest        = 8
	MessageTypeEchoResponse       = 9
)

func (k MessageType) String() string {
	switch k {
	case MessageTypeCallConnectRequest:
		return "CallConnectRequest"
	case MessageTypeCallConnectAck:
		return "CallConnectAck"
	case MessageTypeCallConnectNak:
		return "CallConnectNak"
	case MessageTypeCallConnected:
		return "CallConnected"
	case MessageTypeCallAbort:
		return "CallAbort"
	case MessageTypeCallDisconnect:
		return "CallDisconnect"
	case MessageTypeCallDisconnectAck:
		return "CallDisconnectAck"
	case MessageTypeEchoRequest:
		return "EchoRequest"
	case MessageTypeEchoResponse:
		return "EchoResponse"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

type sstpHeader struct {
	MajorVersion uint8
	MinorVersion uint8
	C            bool
	Length       uint16
}

type sstpControlHeader struct {
	sstpHeader
	MessageType      MessageType
	AttributesLength uint16
	Data             interface{}
}

func handlePacket(input []byte) {
	header := &sstpHeader{}

	header.MajorVersion = input[0] >> 4
	header.MinorVersion = input[0] & 0xf
	header.C = input[1] == 1
	header.Length = binary.BigEndian.Uint16(input[2:4])

	if header.C {
		controlHeader := &sstpControlHeader{}
		controlHeader.sstpHeader = *header
		controlHeader.MessageType = MessageType(binary.BigEndian.Uint16(input[4:6]))
		controlHeader.AttributesLength = binary.BigEndian.Uint16(input[6:8])
		fmt.Printf("hdr: %v", controlHeader)
		return
	}

	fmt.Printf("hdr: %v", header)
}

func main() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go func(c net.Conn) {
			// Shut down the connection.
			defer c.Close()

			// Echo all incoming data.
			//io.Copy(c, c)

			//b, err := ioutil.ReadAll(c)

			var method, path, version string
			n, err := fmt.Fscan(c, &method, &path, &version)
			handleErr(err)

			if n != 3 {
				log.Print("Malformed HTTP")
				return
			}
			if method != "SSTP_DUPLEX_POST" {
				log.Print("Wrong method.")
				return
			}
			if path != "/sra_{BA195980-CD49-458b-9E23-C84EE0ADCD75}/" {
				log.Print("Wrong path.")
				return
			}

			n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n",
				"HTTP/1.1 200 OK",
				"Date: Thu, 09 Nov 2006 00:51:09 GMT",
				"Server: Microsoft-HTTPAPI/2.0",
				"Content-Length: 18446744073709551615")
			handleErr(err)
			log.Printf("%v", n)

			/*b, err := ioutil.ReadAll(c)
			handleErr(err)
			log.Printf("%s\n", b)*/

			ch := make(chan []byte)
			eCh := make(chan error)

			// Start a goroutine to read from our net connection
			go func(ch chan []byte, eCh chan error) {
				for {
					// try to read the data
					data := make([]byte, 512)
					_, err := conn.Read(data)
					if err != nil {
						// send an error if it's encountered
						eCh <- err
						return
					}
					// send data if we read some.
					ch <- data
				}
			}(ch, eCh)

			//ticker := time.Tick(time.Second)
			// continuously read from the connection
			for {
				select {
				case data := <-ch: // This case means we recieved data on the connection
					// Do something with the data
					log.Printf("%s\n", data)
					handlePacket(data)
				case err := <-eCh: // This case means we got an error and the goroutine has finished
					log.Fatalf("%s\n", err)
					// handle our error then exit for loop
					break
					// This will timeout on the read.
					//case <-ticker:
					// do nothing? this is just so we can time out if we need to.
					// you probably don't even need to have this here unless you want
					// do something specifically on the timeout.
				}
			}
		}(conn)
	}
}
