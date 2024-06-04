package spanreedclient

import (
	"encoding/binary"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
)

type SpanreedClientMessageType uint8

const (
	SpanreedClientMessageType_ConnectionRequest SpanreedClientMessageType = iota
	SpanreedClientMessageType_AppMessage

	SpanreedClientMessageType_NONE
)

func messageTypeToClientHeaderId(msgType SpanreedClientMessageType) uint8 {
	switch msgType {
	case SpanreedClientMessageType_ConnectionRequest:
		return 0x0
	case SpanreedClientMessageType_AppMessage:
		return 0x1
	}

	return 0xFF
}

type ConnectionRequest struct {
	ClientId uint32
}

type SpanreedClientMessage struct {
	MagicNumber       uint32
	Version           uint8
	MessageType       SpanreedClientMessageType
	ConnectionRequest *ConnectionRequest
	AppData           []byte
}

type SpanreedClientMessageSerializer struct {
	MagicNumber uint32
	Version     uint8
}

func (s SpanreedClientMessageSerializer) serializeConnectionRequest(out []byte, request *ConnectionRequest) ([]byte, error) {
	out = binary.LittleEndian.AppendUint32(out, request.ClientId)

	return out, nil
}

func (s SpanreedClientMessageSerializer) SerializeMessage(msg *SpanreedClientMessage) ([]byte, error) {
	out := []byte{}
	var err error

	out = binary.LittleEndian.AppendUint32(out, s.MagicNumber)
	versionTypeByte := s.Version<<4 | (messageTypeToClientHeaderId(msg.MessageType) & 0xF)
	out = append(out, versionTypeByte)

	switch msg.MessageType {
	case SpanreedClientMessageType_ConnectionRequest:
		if msg.ConnectionRequest == nil {
			return nil, &errors.MissingFieldError{
				MessageName: "SpanreedClientMessage",
				FieldName:   "ConnectionRequest",
			}
		}
		out, err = s.serializeConnectionRequest(out, msg.ConnectionRequest)
		if err != nil {
			return nil, err
		}
	case SpanreedClientMessageType_AppMessage: // no-op
		break
	case SpanreedClientMessageType_NONE:
	default:
		return nil, &errors.InvalidEnumValue{
			EnumName: "SpanreedClientMessage::MessageType",
			IntValue: messageTypeToClientHeaderId(msg.MessageType),
		}
	}

	return append(out, msg.AppData...), nil
}
