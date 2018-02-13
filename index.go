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
	Data             interface{} // dummy?
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
	Data []byte // dummy?
}

func handlePacket(input []byte, conn net.Conn) {
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

		/*for i := 0; i < int(controlHeader.AttributesLength); i++ {

		}*/
		sendAckPacket(conn)

		fmt.Printf("hdr: %v", controlHeader)
		return
	}

	dataHeader := &sstpDataHeader{}
	dataHeader.sstpHeader = *header
	copy(input[4:(len(input)-4)], dataHeader.Data)

	fmt.Printf("hdr: %v", dataHeader)
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

func sendAckPacket(conn net.Conn) {
	// Fake length = 48, we don't actually implement crypto binding?
	header := sstpHeader{1, 0, true, 48}
	attributes := make([]sstpAttribute, 1)
	attributes[0] = sstpAttribute{0, AttributeIDCryptoBindingReq, 40, nil}
	controlHeader := sstpControlHeader{header, MessageTypeCallConnectAck, uint16(len(attributes)), attributes, nil}

	outputBytes := make([]byte, 48)
	packControlHeader(controlHeader, outputBytes)
	conn.Write(outputBytes)
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
					handlePacket(data, conn)
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
