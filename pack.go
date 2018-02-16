package main

import (
	"encoding/binary"
	"log"
	"net"
)

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
	currentPosition := 7
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
	data := []byte{0, 0, 0, 3} // 3 means, supports SHA1 and SHA256
	attributes[0] = sstpAttribute{0, AttributeIDCryptoBindingReq, 40, data}
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
	copy(outputBytes[4:(len(header.Data)+4)], header.Data)
}

func sendDataPacket(inputBytes []byte, conn net.Conn) {
	length := 8 + len(inputBytes)
	header := sstpHeader{1, 0, false, uint16(length)}
	dataHeader := sstpDataHeader{header, inputBytes}

	//log.Printf("write: %v\n", dataHeader)
	packetBytes := make([]byte, length)
	packDataHeader(dataHeader, packetBytes)
	conn.Write(packetBytes)
}
