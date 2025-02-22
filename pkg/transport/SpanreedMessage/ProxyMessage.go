// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package SpanreedMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ProxyMessage struct {
	_tab flatbuffers.Table
}

func GetRootAsProxyMessage(buf []byte, offset flatbuffers.UOffsetT) *ProxyMessage {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ProxyMessage{}
	x.Init(buf, n+offset)
	return x
}

func FinishProxyMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsProxyMessage(buf []byte, offset flatbuffers.UOffsetT) *ProxyMessage {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &ProxyMessage{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedProxyMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *ProxyMessage) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ProxyMessage) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ProxyMessage) ClientId() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ProxyMessage) MutateClientId(n uint32) bool {
	return rcv._tab.MutateUint32Slot(4, n)
}

func ProxyMessageStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func ProxyMessageAddClientId(builder *flatbuffers.Builder, clientId uint32) {
	builder.PrependUint32Slot(0, clientId, 0)
}
func ProxyMessageEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
