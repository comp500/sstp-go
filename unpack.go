package main

import (
	"encoding/binary"
	"log"
	"net"
)

func handlePacket(input []byte, conn net.Conn, pppdInstance *pppdInstance) {
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
	dataHeader.Data = input[4:header.Length]

	handleDataPacket(dataHeader, conn, pppdInstance)
}

func handleDataPacket(dataHeader sstpDataHeader, conn net.Conn, pppdInstance *pppdInstance) {
	log.Printf("read: %v\n", dataHeader)
	if pppdInstance.commandInst == nil {
		log.Fatal("pppd instance not started")
	} else {
		escaped := pppEscape(dataHeader.Data)
		n, err := pppdInstance.stdin.Write(escaped)
		handleErr(err)
		log.Printf("escaped: %v", escaped)
		log.Printf("%v bytes written to pppd", n)
	}
}

func handleControlPacket(controlHeader sstpControlHeader, conn net.Conn, pppdInstance *pppdInstance) {
	log.Printf("read: %v\n", controlHeader)

	if controlHeader.MessageType == MessageTypeCallConnectRequest {
		sendConnectionAckPacket(conn)
		// TODO: implement Nak?
		// -> if protocols specified by req not supported
		// however there is only PPP currently, so not a problem
		createPPPD(pppdInstance)
		log.Print("pppd instance created")
		go addPPPDResponder(pppdInstance, conn)
	} else if controlHeader.MessageType == MessageTypeCallDisconnect {
		sendDisconnectAckPacket(conn)
		if pppdInstance.commandInst != nil {
			// kill pppd if disconnect
			err := pppdInstance.commandInst.Process.Kill()
			handleErr(err)
			pppdInstance.commandInst = nil
		}
	} else if controlHeader.MessageType == MessageTypeEchoRequest {
		// TODO: implement hello timer and echo request?
		sendEchoResponsePacket(conn)
	} else if controlHeader.MessageType == MessageTypeCallAbort {
		// TODO: parse error
		log.Fatal("error encountered, connection aborted")
	}
	// TODO: implement connected
}
