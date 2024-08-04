package bmproxy

import "encoding/binary"

type DestinationMessageType byte

const (
	DestinationMessageType_Unknown           DestinationMessageType = 0
	DestinationMessageType_ConnectionVerdict DestinationMessageType = 1
	DestinationMessageType_DisconnectClient  DestinationMessageType = 2
	DestinationMessageType_Pong              DestinationMessageType = 3
	DestinationMessageType_Stats             DestinationMessageType = 4
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

func GetDestinationClientId(payload []byte) uint32 {
	if len(payload) < 8 {
		return 0xFFFFFFFF
	}

	return binary.LittleEndian.Uint32(payload[4:])
}

func SetClientId(payload []byte, clientId uint32) {
	if len(payload) < 8 {
		return
	}

	binary.LittleEndian.PutUint32(payload[4:], clientId)
}

func GetVerdict(payload []byte) bool {
	if len(payload) < 18 {
		return false
	}

	return payload[17] > 0
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
