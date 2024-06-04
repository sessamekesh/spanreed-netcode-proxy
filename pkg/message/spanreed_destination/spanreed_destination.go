package spanreeddestination

import (
	"encoding/binary"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
)

type SpanreedDestinationMessageType uint8

const (
	SpanreedDestinationMessageType_ConnectionResponse SpanreedDestinationMessageType = iota
	SpanreedDestinationMessageType_AppData

	SpanreedDestinationMessageType_NONE
)

func messageTypeToClientHeaderId(msgType SpanreedDestinationMessageType) uint8 {
	switch msgType {
	case SpanreedDestinationMessageType_ConnectionResponse:
		return 0x0
	case SpanreedDestinationMessageType_AppData:
		return 0x1
	}

	return 0xFF
}

type ConnectionResponse struct {
	Verdict   bool
	IsTimeout bool
}

type SpanreedDestinationMessage struct {
	MagicNumber        uint32
	Version            uint8
	MessageType        SpanreedDestinationMessageType
	ConnectionResponse *ConnectionResponse
	AppData            []byte
}

type SpanreedDestinationMessageSerializer struct {
	MagicNumber uint32
	Version     uint8
}

func (s SpanreedDestinationMessageSerializer) serializeConnectionResponse(out []byte, response *ConnectionResponse) ([]byte, error) {
	var verdict uint8
	var isTimeout uint8

	if response.Verdict {
		verdict = 0b1
	}
	if response.IsTimeout {
		isTimeout = 0b10
	}

	return append(out, verdict|isTimeout), nil
}

func (s SpanreedDestinationMessageSerializer) Serialize(msg *SpanreedDestinationMessage) ([]byte, error) {
	out := []byte{}
	var err error

	out = binary.LittleEndian.AppendUint32(out, s.MagicNumber)
	versionTypeByte := s.Version<<4 | (messageTypeToClientHeaderId(msg.MessageType) & 0xF)
	out = append(out, versionTypeByte)

	switch msg.MessageType {
	case SpanreedDestinationMessageType_ConnectionResponse:
		if msg.ConnectionResponse == nil {
			return nil, &errors.MissingFieldError{
				MessageName: "SpanreedDestinationMessage",
				FieldName:   "ConnectionResponse",
			}
		}
		out, err = s.serializeConnectionResponse(out, msg.ConnectionResponse)
		if err != nil {
			return nil, err
		}
	case SpanreedDestinationMessageType_AppData: // no-op
		break
	case SpanreedDestinationMessageType_NONE:
	default:
		return nil, &errors.InvalidEnumValue{
			EnumName: "SpanreedDestinationMessage::MessageType",
			IntValue: messageTypeToClientHeaderId(msg.MessageType),
		}
	}

	return append(out, msg.AppData...), nil
}
