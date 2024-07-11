// automatically generated by the FlatBuffers compiler, do not modify

/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */

import * as flatbuffers from 'flatbuffers';

export class ConnectClientVerdict {
  bb: flatbuffers.ByteBuffer | null = null;
  bb_pos = 0;
  __init(i: number, bb: flatbuffers.ByteBuffer): ConnectClientVerdict {
    this.bb_pos = i;
    this.bb = bb;
    return this;
  }

  static getRootAsConnectClientVerdict(bb: flatbuffers.ByteBuffer, obj?: ConnectClientVerdict): ConnectClientVerdict {
    return (obj || new ConnectClientVerdict()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
  }

  static getSizePrefixedRootAsConnectClientVerdict(bb: flatbuffers.ByteBuffer, obj?: ConnectClientVerdict): ConnectClientVerdict {
    bb.setPosition(bb.position() + flatbuffers.SIZE_PREFIX_LENGTH);
    return (obj || new ConnectClientVerdict()).__init(bb.readInt32(bb.position()) + bb.position(), bb);
  }

  accepted(): boolean {
    const offset = this.bb!.__offset(this.bb_pos, 4);
    return offset ? !!this.bb!.readInt8(this.bb_pos + offset) : false;
  }

  errorReason(): string | null
  errorReason(optionalEncoding: flatbuffers.Encoding): string | Uint8Array | null
  errorReason(optionalEncoding?: any): string | Uint8Array | null {
    const offset = this.bb!.__offset(this.bb_pos, 6);
    return offset ? this.bb!.__string(this.bb_pos + offset, optionalEncoding) : null;
  }

  appData(index: number): number | null {
    const offset = this.bb!.__offset(this.bb_pos, 8);
    return offset ? this.bb!.readUint8(this.bb!.__vector(this.bb_pos + offset) + index) : 0;
  }

  appDataLength(): number {
    const offset = this.bb!.__offset(this.bb_pos, 8);
    return offset ? this.bb!.__vector_len(this.bb_pos + offset) : 0;
  }

  appDataArray(): Uint8Array | null {
    const offset = this.bb!.__offset(this.bb_pos, 8);
    return offset ? new Uint8Array(this.bb!.bytes().buffer, this.bb!.bytes().byteOffset + this.bb!.__vector(this.bb_pos + offset), this.bb!.__vector_len(this.bb_pos + offset)) : null;
  }

  static startConnectClientVerdict(builder: flatbuffers.Builder) {
    builder.startObject(3);
  }

  static addAccepted(builder: flatbuffers.Builder, accepted: boolean) {
    builder.addFieldInt8(0, +accepted, +false);
  }

  static addErrorReason(builder: flatbuffers.Builder, errorReasonOffset: flatbuffers.Offset) {
    builder.addFieldOffset(1, errorReasonOffset, 0);
  }

  static addAppData(builder: flatbuffers.Builder, appDataOffset: flatbuffers.Offset) {
    builder.addFieldOffset(2, appDataOffset, 0);
  }

  static createAppDataVector(builder: flatbuffers.Builder, data: number[] | Uint8Array): flatbuffers.Offset {
    builder.startVector(1, data.length, 1);
    for (let i = data.length - 1; i >= 0; i--) {
      builder.addInt8(data[i]!);
    }
    return builder.endVector();
  }

  static startAppDataVector(builder: flatbuffers.Builder, numElems: number) {
    builder.startVector(1, numElems, 1);
  }

  static endConnectClientVerdict(builder: flatbuffers.Builder): flatbuffers.Offset {
    const offset = builder.endObject();
    return offset;
  }

  static finishConnectClientVerdictBuffer(builder: flatbuffers.Builder, offset: flatbuffers.Offset) {
    builder.finish(offset);
  }

  static finishSizePrefixedConnectClientVerdictBuffer(builder: flatbuffers.Builder, offset: flatbuffers.Offset) {
    builder.finish(offset, undefined, true);
  }

  static createConnectClientVerdict(builder: flatbuffers.Builder, accepted: boolean, errorReasonOffset: flatbuffers.Offset, appDataOffset: flatbuffers.Offset): flatbuffers.Offset {
    ConnectClientVerdict.startConnectClientVerdict(builder);
    ConnectClientVerdict.addAccepted(builder, accepted);
    ConnectClientVerdict.addErrorReason(builder, errorReasonOffset);
    ConnectClientVerdict.addAppData(builder, appDataOffset);
    return ConnectClientVerdict.endConnectClientVerdict(builder);
  }
}
