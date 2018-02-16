package main

import (
	"fmt"
	"io"
	"os/exec"
)

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

type pppdInstance struct {
	commandInst *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
}

type parseReturn struct {
	isControl bool
	Data      []byte
}
