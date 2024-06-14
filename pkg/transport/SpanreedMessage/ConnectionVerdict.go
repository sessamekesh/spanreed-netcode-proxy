// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package SpanreedMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ConnectionVerdict struct {
	_tab flatbuffers.Table
}

func GetRootAsConnectionVerdict(buf []byte, offset flatbuffers.UOffsetT) *ConnectionVerdict {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ConnectionVerdict{}
	x.Init(buf, n+offset)
	return x
}

func FinishConnectionVerdictBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsConnectionVerdict(buf []byte, offset flatbuffers.UOffsetT) *ConnectionVerdict {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &ConnectionVerdict{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedConnectionVerdictBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *ConnectionVerdict) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ConnectionVerdict) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ConnectionVerdict) ClientId() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ConnectionVerdict) MutateClientId(n uint32) bool {
	return rcv._tab.MutateUint32Slot(4, n)
}

func (rcv *ConnectionVerdict) Accepted() bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetBool(o + rcv._tab.Pos)
	}
	return false
}

func (rcv *ConnectionVerdict) MutateAccepted(n bool) bool {
	return rcv._tab.MutateBoolSlot(6, n)
}

func ConnectionVerdictStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func ConnectionVerdictAddClientId(builder *flatbuffers.Builder, clientId uint32) {
	builder.PrependUint32Slot(0, clientId, 0)
}
func ConnectionVerdictAddAccepted(builder *flatbuffers.Builder, accepted bool) {
	builder.PrependBoolSlot(1, accepted, false)
}
func ConnectionVerdictEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
