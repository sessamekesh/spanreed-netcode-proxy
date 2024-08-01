package main

import "encoding/binary"

type DestinationMessageType byte

const (
	DestinationMessageType_Unknown           DestinationMessageType = 0
	DestinationMessageType_ConnectionVerdict DestinationMessageType = 1
	DestinationMessageType_Pong              DestinationMessageType = 2
	DestinationMessageType_Stats             DestinationMessageType = 3
)

func GetDestinationMessageType(payload []byte) DestinationMessageType {
	if len(payload) <= 16 {
		return DestinationMessageType_Unknown
	}

	typeByte := payload[16]
	if typeByte <= byte(DestinationMessageType_Stats) && typeByte >= byte(DestinationMessageType_Unknown) {
		return DestinationMessageType(typeByte)
	}

	return DestinationMessageType_Unknown
}

func SetPongRecvTimestamp(payload []byte, timestamp uint64) []byte {
	offset := 17 + 40
	if len(payload) < offset+8 {
		return payload
	}
	binary.LittleEndian.PutUint64(payload[offset:], timestamp)
	return payload
}

func SetPongForwardTimestamp(payload []byte, timestamp uint64) []byte {
	offset := 17 + 48
	if len(payload) < offset+8 {
		return payload
	}
	binary.LittleEndian.PutUint64(payload[offset:], timestamp)
	return payload
}
