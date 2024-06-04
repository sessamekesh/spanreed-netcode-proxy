package client

import (
	"encoding/binary"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
)

type ClientMessageType uint8

const (
	ClientMessageType_ConnectionRequest ClientMessageType = iota
	// ClientMessageType_Ping
	// ClientMessageType_CloseConnection
	ClientMessageType_AppMessage

	ClientMessageType_NONE
)

func clientHeaderIdToMessageType(headerId uint8) ClientMessageType {
	switch headerId {
	case 0x0:
		return ClientMessageType_ConnectionRequest
	case 0x1:
		return ClientMessageType_AppMessage
	}

	return ClientMessageType_NONE
}

type ConnectionRequest struct {
	ConnectionString string
}

type ClientMessage struct {
	MagicNumber       uint32
	Version           uint8
	MessageType       ClientMessageType
	ConnectionRequest *ConnectionRequest
	AppData           []byte
}

type ClientMessageSerializer struct {
	MagicNumber uint32
	Version     uint8
}

func (s ClientMessageSerializer) parseConnectionRequest(msg []byte, readPtr int) (int, *ConnectionRequest, error) {
	if len(msg) < readPtr+2 {
		return readPtr, nil, &errors.Underflow{
			MessageName: "Client::ConnectionRequest",
			MsgSize:     len(msg),
			MinimumSize: readPtr + 2,
		}
	}

	connectionStringByteLength := binary.LittleEndian.Uint16(msg[readPtr : readPtr+2])
	if connectionStringByteLength == 0 {
		return readPtr, nil, &errors.Underflow{
			MessageName: "Client::ConnectionRequest::ConnectionStringLength",
			MsgSize:     0,
			MinimumSize: 1,
		}
	}

	ptr := readPtr + 2

	if len(msg) < ptr+int(connectionStringByteLength) {
		return readPtr, nil, &errors.Underflow{
			MessageName: "Client::ConnectionRequest::ConnectionString",
			MsgSize:     len(msg) - ptr,
			MinimumSize: int(connectionStringByteLength),
		}
	}

	connectionString := string(msg[ptr : ptr+int(connectionStringByteLength)])

	connectionRequest := &ConnectionRequest{
		ConnectionString: connectionString,
	}

	return ptr + int(connectionStringByteLength), connectionRequest, nil
}

func (s ClientMessageSerializer) Parse(msg []byte) (*ClientMessage, error) {
	if len(msg) < 5 {
		// Underflow
		return nil, &errors.Underflow{
			MessageName: "ClientMessage",
			MsgSize:     len(msg),
			MinimumSize: 5,
		}
	}

	magicNumber := binary.LittleEndian.Uint32(msg[0:4])
	versionTypeByte := msg[4]
	version := versionTypeByte & 0xF0 >> 4
	msgTypeNum := versionTypeByte & 0xF
	msgType := clientHeaderIdToMessageType(msgTypeNum)

	if magicNumber != s.MagicNumber || version != s.Version {
		return nil, &errors.InvalidHeaderVersion{
			ExpectedMagicNumber: s.MagicNumber,
			ExpectedVersion:     s.Version,
			ActualMagicNumber:   magicNumber,
			ActualVersion:       version,
		}
	}

	readPtr := 5
	var connectionRequest *ConnectionRequest
	var pErr error
	appData := []byte{}

	switch msgType {
	case ClientMessageType_ConnectionRequest:
		readPtr, connectionRequest, pErr = s.parseConnectionRequest(msg, readPtr)
		if pErr != nil {
			return nil, pErr
		}
	case ClientMessageType_AppMessage: // no-op
		break
	case ClientMessageType_NONE:
	default:
		return nil, &errors.InvalidEnumValue{
			EnumName: "ClientMessageType",
			IntValue: msgTypeNum,
		}
	}

	if len(msg) < readPtr {
		appData = msg[readPtr:]
	}

	return &ClientMessage{
		MagicNumber:       magicNumber,
		Version:           version,
		MessageType:       msgType,
		ConnectionRequest: connectionRequest,
		AppData:           appData,
	}, nil
}
