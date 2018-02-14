package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
)

func handleErr(err error) {
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

type sstpHeader struct {
	MajorVersion uint8
	MinorVersion uint8
	C            bool
	Length       uint16
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

type sstpControlHeader struct {
	sstpHeader
	MessageType      MessageType
	AttributesLength uint16
	Attributes       []sstpAttribute
}

// AttributeID is the type of attribute this attribute is
type AttributeID uint8

// Constants for MessageType values
const (
	AttributeIDEncapsulatedProtocolID = 1
	AttributeIDStatusInfo             = 2
	AttributeIDCryptoBinding          = 3
	AttributeIDCryptoBindingReq       = 4
)

func (k AttributeID) String() string {
	switch k {
	case AttributeIDEncapsulatedProtocolID:
		return "EncapsulatedProtocolID"
	case AttributeIDStatusInfo:
		return "StatusInfo"
	case AttributeIDCryptoBinding:
		return "CryptoBinding"
	case AttributeIDCryptoBindingReq:
		return "CryptoBindingReq"
	default:
		return fmt.Sprintf("Unknown(%d)", k)
	}
}

type sstpAttribute struct {
	Reserved    byte
	AttributeID AttributeID
	Length      uint16
	Data        []byte
}

type sstpDataHeader struct {
	sstpHeader
	Data []byte
}

func handlePacket(input []byte, conn net.Conn, pppdInstance **exec.Cmd) {
	header := sstpHeader{}

	header.MajorVersion = input[0] >> 4
	header.MinorVersion = input[0] & 0xf
	header.C = input[1] == 1
	header.Length = binary.BigEndian.Uint16(input[2:4])

	if header.C {
		controlHeader := sstpControlHeader{}
		controlHeader.sstpHeader = header
		controlHeader.MessageType = MessageType(binary.BigEndian.Uint16(input[4:6]))
		controlHeader.AttributesLength = binary.BigEndian.Uint16(input[6:8])

		attributes := make([]sstpAttribute, int(controlHeader.AttributesLength))
		consumedBytes := 8
		for i := 0; i < len(attributes); i++ {
			attribute := sstpAttribute{}
			// ignore Reserved byte
			attribute.AttributeID = AttributeID(input[consumedBytes+1])
			attribute.Length = binary.BigEndian.Uint16(input[(consumedBytes + 2):(consumedBytes + 4)])
			attribute.Data = input[(consumedBytes + 4):(consumedBytes + int(attribute.Length))]
			consumedBytes += int(attribute.Length)

			attributes[i] = attribute
		}
		controlHeader.Attributes = attributes

		handleControlPacket(controlHeader, conn, pppdInstance)
		return
	}

	dataHeader := sstpDataHeader{}
	dataHeader.sstpHeader = header
	dataHeader.Data = input[4:(len(input) - 4)]

	if pppdInstance == nil {
		log.Fatal("pppd instance not started test")
	}

	handleDataPacket(dataHeader, conn, pppdInstance)
}

func handleDataPacket(dataHeader sstpDataHeader, conn net.Conn, pppdInstance **exec.Cmd) {
	log.Printf("read: %v\n", dataHeader)
	if pppdInstance == nil {
		log.Fatal("pppd instance not started")
	} else {
		pppIn, err := (*pppdInstance).StdinPipe()
		handleErr(err)
		n, err := pppIn.Write(dataHeader.Data)
		handleErr(err)
		log.Printf("%v bytes written to pppd", n)
	}
}

func handleControlPacket(controlHeader sstpControlHeader, conn net.Conn, pppdInstance **exec.Cmd) {
	log.Printf("read: %v\n", controlHeader)

	if controlHeader.MessageType == MessageTypeCallConnectRequest {
		sendConnectionAckPacket(conn)
		// TODO: implement Nak?
		// -> if protocols specified by req not supported
		// however there is only PPP currently, so not a problem
		pppdInstanceValue := createPPPD()
		pppdInstance = &pppdInstanceValue
		log.Print("pppd instance created")
		if pppdInstance == nil {
			log.Print("instanceptr is nil")
		}
		if *pppdInstance == nil {
			log.Print("instanceptr2 is nil")
		}
		addPPPDResponder(*pppdInstance, conn)
	} else if controlHeader.MessageType == MessageTypeCallDisconnect {
		sendDisconnectAckPacket(conn)
	} else if controlHeader.MessageType == MessageTypeEchoRequest {
		// TODO: implement hello timer and echo request?
		sendEchoResponsePacket(conn)
	} else if controlHeader.MessageType == MessageTypeCallAbort {
		// TODO: parse error
		log.Fatal("error encountered, connection aborted")
	}
	// TODO: implement connected
}

func packHeader(header sstpHeader, outputBytes []byte) {
	var version = (header.MajorVersion << 4) + header.MinorVersion
	outputBytes[0] = version
	if header.C {
		outputBytes[1] = 1
	} else {
		outputBytes[1] = 0
	}
	binary.BigEndian.PutUint16(outputBytes[2:4], header.Length)
}

func packAttribute(attribute sstpAttribute, outputBytes []byte) {
	// Don't set 0, should be reserved
	outputBytes[1] = uint8(attribute.AttributeID)
	binary.BigEndian.PutUint16(outputBytes[2:4], attribute.Length)
	copy(outputBytes[5:(len(outputBytes)-5)], attribute.Data)
}

func packControlHeader(header sstpControlHeader, outputBytes []byte) {
	packHeader(header.sstpHeader, outputBytes[0:4])
	binary.BigEndian.PutUint16(outputBytes[4:6], uint16(header.MessageType))
	binary.BigEndian.PutUint16(outputBytes[6:8], header.AttributesLength)
	currentPosition := 8
	for _, v := range header.Attributes {
		nextPosition := currentPosition + int(v.Length)
		packAttribute(v, outputBytes[currentPosition:nextPosition])
		currentPosition = nextPosition
	}
}

func sendConnectionAckPacket(conn net.Conn) {
	// Fake attribute, we don't actually implement crypto binding
	header := sstpHeader{1, 0, true, 48}
	attributes := make([]sstpAttribute, 1)
	attributes[0] = sstpAttribute{0, AttributeIDCryptoBindingReq, 40, nil}
	controlHeader := sstpControlHeader{header, MessageTypeCallConnectAck, uint16(len(attributes)), attributes}

	log.Printf("write: %v\n", controlHeader)
	outputBytes := make([]byte, 48)
	packControlHeader(controlHeader, outputBytes)
	conn.Write(outputBytes)
}

func sendDisconnectAckPacket(conn net.Conn) {
	header := sstpHeader{1, 0, true, 8}
	attributes := make([]sstpAttribute, 0)
	controlHeader := sstpControlHeader{header, MessageTypeCallDisconnectAck, 0, attributes}

	log.Printf("write: %v\n", controlHeader)
	outputBytes := make([]byte, 8)
	packControlHeader(controlHeader, outputBytes)
	conn.Write(outputBytes)
}

func sendEchoResponsePacket(conn net.Conn) {
	header := sstpHeader{1, 0, true, 8}
	attributes := make([]sstpAttribute, 0)
	controlHeader := sstpControlHeader{header, MessageTypeEchoResponse, 0, attributes}

	log.Printf("write: %v\n", controlHeader)
	outputBytes := make([]byte, 8)
	packControlHeader(controlHeader, outputBytes)
	conn.Write(outputBytes)
}

func packDataHeader(header sstpDataHeader, outputBytes []byte) {
	packHeader(header.sstpHeader, outputBytes[0:4])
	copy(header.Data, outputBytes[4:(len(header.Data)-4)])
}

func sendDataPacket(inputBytes []byte, conn net.Conn) {
	length := 8 + len(inputBytes)
	header := sstpHeader{1, 0, false, uint16(length)}
	dataHeader := sstpDataHeader{header, inputBytes}

	log.Printf("write: %v\n", dataHeader)
	packetBytes := make([]byte, length)
	packDataHeader(dataHeader, packetBytes)
	conn.Write(packetBytes)
}

func createPPPD() *exec.Cmd {
	pppdCmd := exec.Command("pppd")
	err := pppdCmd.Start()
	handleErr(err)
	return pppdCmd
}

func addPPPDResponder(pppdInstance *exec.Cmd, conn net.Conn) {
	go func(pppdInstance *exec.Cmd, conn net.Conn) {
		defer pppdInstance.Process.Kill()

		ch := make(chan []byte)
		eCh := make(chan error)
		pppdOut, err := pppdInstance.StdoutPipe()
		handleErr(err)

		// Start a goroutine to read from our net connection
		go func(ch chan []byte, eCh chan error, pppdOut io.ReadCloser) {
			for {
				// try to read the data
				data := make([]byte, 512)
				n, err := pppdOut.Read(data)
				fmt.Printf("pppd: %v bytes read", n)

				if err != nil {
					// send an error if it's encountered
					eCh <- err
					return
				}
				// send data if we read some.
				ch <- data
			}
		}(ch, eCh, pppdOut)

		//ticker := time.Tick(time.Second)
		// continuously read from the connection
		for {
			select {
			case data := <-ch: // This case means we recieved data on the connection
				// Do something with the data
				//log.Printf("%s\n", hex.Dump(data))
				sendDataPacket(data, conn)
			case err := <-eCh: // This case means we got an error and the goroutine has finished
				if err == io.EOF {
					log.Print("pppd disconnected")
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
	}(pppdInstance, conn)
}

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Listening on port 8080")
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
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 400 Bad Request",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 15",
					"400 Bad Request")
				handleErr(err)
				log.Printf("%v HTTP bytes written (400)", n)
				return
			}
			if method != "SSTP_DUPLEX_POST" {
				log.Printf("Wrong method (%s)", method)
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 405 Method Not Allowed",
					"Allow: SSTP_DUPLEX_POST",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 22",
					"405 Method Not Allowed")
				handleErr(err)
				log.Printf("%v HTTP bytes written (405)", n)
				return
			}
			if path != "/sra_{BA195980-CD49-458b-9E23-C84EE0ADCD75}/" {
				log.Printf("Wrong path (%s)", path)
				n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n%s",
					"HTTP/1.1 404 File Not Found",
					"Server: sstp-go",
					"Connection: close",
					"Content-Length: 18",
					"404 File Not Found")
				handleErr(err)
				log.Printf("%v HTTP bytes written (404)", n)
				return
			}

			// digest rest of first packet
			data := make([]byte, 512)
			conn.Read(data)
			data = nil // free memory

			log.Print("HTTP request received")

			n, err = fmt.Fprintf(c, "%s\r\n%s\r\n%s\r\n%s\r\n\r\n",
				"HTTP/1.1 200 OK",
				"Date: Thu, 09 Nov 2006 00:51:09 GMT",
				"Server: Microsoft-HTTPAPI/2.0",
				"Content-Length: 18446744073709551615")
			handleErr(err)
			log.Printf("%v HTTP bytes written", n)

			ch := make(chan []byte)
			eCh := make(chan error)
			var pppdInstance **exec.Cmd // store null pointer to future pppd instance

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
					//log.Printf("%s\n", hex.Dump(data))
					handlePacket(data, conn, pppdInstance)
					if pppdInstance == nil {
						log.Fatal("pppd instance not started test2")
					}
				case err := <-eCh: // This case means we got an error and the goroutine has finished
					if err == io.EOF {
						log.Print("Client disconnected")
					} else {
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
			}
		}(conn)
	}
}
