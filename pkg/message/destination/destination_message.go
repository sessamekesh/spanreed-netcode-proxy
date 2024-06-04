package destination

import (
	"encoding/binary"

	"github.com/sessamekesh/spanreed-netcode-proxy/pkg/errors"
)

type DestinationMessageType uint8

const (
	DestinationMessageType_ConnectionResponse DestinationMessageType = iota
	DestinationMessageType_AppMessage

	DestinationMessageType_NONE
)

func clientHeaderIdToMessageType(headerId uint8) DestinationMessageType {
	switch headerId {
	case 0x0:
		return DestinationMessageType_ConnectionResponse
	case 0x1:
		return DestinationMessageType_AppMessage
	}

	return DestinationMessageType_NONE
}

type ConnectionResponse struct {
	Verdict bool
}

type DestinationMessage struct {
	MagicNumber        uint32
	Verison            uint8
	MessageType        DestinationMessageType
	ConnectionResponse *ConnectionResponse
	AppData            []byte
}

type DestinationMessageSerializer struct {
	MagicNumber uint32
	Version     uint8
}

func (s DestinationMessageSerializer) parseConnectionResponse(msg []byte, readPtr int) (int, *ConnectionResponse, error) {
	if len(msg) < readPtr+1 {
		return readPtr, nil, &errors.Underflow{
			MessageName: "Destination::ConnectionResponse",
			MsgSize:     len(msg),
			MinimumSize: readPtr + 1,
		}
	}

	return readPtr + 1, &ConnectionResponse{
		Verdict: msg[readPtr] > 0,
	}, nil
}

func (s DestinationMessageSerializer) Parse(msg []byte) (*DestinationMessage, error) {
	if len(msg) < 5 {
		return nil, &errors.Underflow{
			MessageName: "DestinationMessage",
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
	var connectionResponse *ConnectionResponse
	var pErr error
	appData := []byte{}

	switch msgType {
	case DestinationMessageType_ConnectionResponse:
		readPtr, connectionResponse, pErr = s.parseConnectionResponse(msg, readPtr)
		if pErr != nil {
			return nil, pErr
		}
	case DestinationMessageType_AppMessage: // no-op
		break
	case DestinationMessageType_NONE:
	default:
		return nil, &errors.InvalidEnumValue{
			EnumName: "DestinationMessageType",
			IntValue: msgTypeNum,
		}
	}

	if len(msg) < readPtr {
		appData = msg[readPtr:]
	}

	return &DestinationMessage{
		MagicNumber:        magicNumber,
		Verison:            version,
		MessageType:        msgType,
		ConnectionResponse: connectionResponse,
		AppData:            appData,
	}, nil
}
