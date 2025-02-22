// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package SpanreedMessage

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ProxyDestClientMessage struct {
	_tab flatbuffers.Table
}

func GetRootAsProxyDestClientMessage(buf []byte, offset flatbuffers.UOffsetT) *ProxyDestClientMessage {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &ProxyDestClientMessage{}
	x.Init(buf, n+offset)
	return x
}

func FinishProxyDestClientMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsProxyDestClientMessage(buf []byte, offset flatbuffers.UOffsetT) *ProxyDestClientMessage {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &ProxyDestClientMessage{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedProxyDestClientMessageBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *ProxyDestClientMessage) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *ProxyDestClientMessage) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *ProxyDestClientMessage) ClientId() uint32 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetUint32(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *ProxyDestClientMessage) MutateClientId(n uint32) bool {
	return rcv._tab.MutateUint32Slot(4, n)
}

func ProxyDestClientMessageStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func ProxyDestClientMessageAddClientId(builder *flatbuffers.Builder, clientId uint32) {
	builder.PrependUint32Slot(0, clientId, 0)
}
func ProxyDestClientMessageEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
