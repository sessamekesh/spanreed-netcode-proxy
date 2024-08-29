package bmproxy

import "encoding/binary"

type ClientMessageType byte

const (
	ClientMessageType_Unknown          ClientMessageType = 0
	ClientMessageType_ConnectClient    ClientMessageType = 1
	ClientMessageType_DisconnectClient ClientMessageType = 2
	ClientMessageType_Ping             ClientMessageType = 3
	ClientMessageType_GetStats         ClientMessageType = 4
)

func GetClientMessageType(payload []byte) ClientMessageType {
	if len(payload) <= 16 {
		return ClientMessageType_Unknown
	}

	typeByte := payload[16]
	if typeByte <= byte(ClientMessageType_GetStats) && typeByte >= byte(ClientMessageType_Unknown) {
		return ClientMessageType(typeByte)
	}

	return ClientMessageType_Unknown
}

func SetPingRecvTimestamp(payload []byte, timestamp uint64) []byte {
	offset := 17 + 8
	if len(payload) < offset+8 {
		return payload
	}
	binary.LittleEndian.PutUint64(payload[offset:], timestamp)
	return payload
}

func SetPingForwardTimestamp(payload []byte, timestamp uint64) []byte {
	offset := 17 + 16
	if len(payload) < offset+8 {
		return payload
	}
	binary.LittleEndian.PutUint64(payload[offset:], timestamp)
	return payload
}

func GetDestAddr(payload []byte) string {
	if len(payload) < 17 {
		return ""
	}

	dest_url_len := payload[17]
	return string(payload[18 : 18+dest_url_len])
}
