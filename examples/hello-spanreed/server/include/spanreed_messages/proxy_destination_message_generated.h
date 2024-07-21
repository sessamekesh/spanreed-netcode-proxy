// automatically generated by the FlatBuffers compiler, do not modify


#ifndef FLATBUFFERS_GENERATED_PROXYDESTINATIONMESSAGE_SPANREEDMESSAGE_H_
#define FLATBUFFERS_GENERATED_PROXYDESTINATIONMESSAGE_SPANREEDMESSAGE_H_

#include "flatbuffers/flatbuffers.h"

// Ensure the included flatbuffers.h is the same version as when this file was
// generated, otherwise it may not be compatible.
static_assert(FLATBUFFERS_VERSION_MAJOR == 24 &&
              FLATBUFFERS_VERSION_MINOR == 3 &&
              FLATBUFFERS_VERSION_REVISION == 25,
             "Non-compatible flatbuffers version included");

namespace SpanreedMessage {

struct ProxyDestConnectionRequest;
struct ProxyDestConnectionRequestBuilder;

struct ProxyDestClientMessage;
struct ProxyDestClientMessageBuilder;

struct ProxyDestCloseConnection;
struct ProxyDestCloseConnectionBuilder;

struct ProxyDestinationMessage;
struct ProxyDestinationMessageBuilder;

enum ProxyDestInnerMsg : uint8_t {
  ProxyDestInnerMsg_NONE = 0,
  ProxyDestInnerMsg_ProxyDestConnectionRequest = 1,
  ProxyDestInnerMsg_ProxyDestClientMessage = 2,
  ProxyDestInnerMsg_ProxyDestCloseConnection = 3,
  ProxyDestInnerMsg_MIN = ProxyDestInnerMsg_NONE,
  ProxyDestInnerMsg_MAX = ProxyDestInnerMsg_ProxyDestCloseConnection
};

inline const ProxyDestInnerMsg (&EnumValuesProxyDestInnerMsg())[4] {
  static const ProxyDestInnerMsg values[] = {
    ProxyDestInnerMsg_NONE,
    ProxyDestInnerMsg_ProxyDestConnectionRequest,
    ProxyDestInnerMsg_ProxyDestClientMessage,
    ProxyDestInnerMsg_ProxyDestCloseConnection
  };
  return values;
}

inline const char * const *EnumNamesProxyDestInnerMsg() {
  static const char * const names[5] = {
    "NONE",
    "ProxyDestConnectionRequest",
    "ProxyDestClientMessage",
    "ProxyDestCloseConnection",
    nullptr
  };
  return names;
}

inline const char *EnumNameProxyDestInnerMsg(ProxyDestInnerMsg e) {
  if (::flatbuffers::IsOutRange(e, ProxyDestInnerMsg_NONE, ProxyDestInnerMsg_ProxyDestCloseConnection)) return "";
  const size_t index = static_cast<size_t>(e);
  return EnumNamesProxyDestInnerMsg()[index];
}

template<typename T> struct ProxyDestInnerMsgTraits {
  static const ProxyDestInnerMsg enum_value = ProxyDestInnerMsg_NONE;
};

template<> struct ProxyDestInnerMsgTraits<SpanreedMessage::ProxyDestConnectionRequest> {
  static const ProxyDestInnerMsg enum_value = ProxyDestInnerMsg_ProxyDestConnectionRequest;
};

template<> struct ProxyDestInnerMsgTraits<SpanreedMessage::ProxyDestClientMessage> {
  static const ProxyDestInnerMsg enum_value = ProxyDestInnerMsg_ProxyDestClientMessage;
};

template<> struct ProxyDestInnerMsgTraits<SpanreedMessage::ProxyDestCloseConnection> {
  static const ProxyDestInnerMsg enum_value = ProxyDestInnerMsg_ProxyDestCloseConnection;
};

bool VerifyProxyDestInnerMsg(::flatbuffers::Verifier &verifier, const void *obj, ProxyDestInnerMsg type);
bool VerifyProxyDestInnerMsgVector(::flatbuffers::Verifier &verifier, const ::flatbuffers::Vector<::flatbuffers::Offset<void>> *values, const ::flatbuffers::Vector<uint8_t> *types);

struct ProxyDestConnectionRequest FLATBUFFERS_FINAL_CLASS : private ::flatbuffers::Table {
  typedef ProxyDestConnectionRequestBuilder Builder;
  enum FlatBuffersVTableOffset FLATBUFFERS_VTABLE_UNDERLYING_TYPE {
    VT_CLIENT_ID = 4
  };
  uint32_t client_id() const {
    return GetField<uint32_t>(VT_CLIENT_ID, 0);
  }
  bool Verify(::flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<uint32_t>(verifier, VT_CLIENT_ID, 4) &&
           verifier.EndTable();
  }
};

struct ProxyDestConnectionRequestBuilder {
  typedef ProxyDestConnectionRequest Table;
  ::flatbuffers::FlatBufferBuilder &fbb_;
  ::flatbuffers::uoffset_t start_;
  void add_client_id(uint32_t client_id) {
    fbb_.AddElement<uint32_t>(ProxyDestConnectionRequest::VT_CLIENT_ID, client_id, 0);
  }
  explicit ProxyDestConnectionRequestBuilder(::flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  ::flatbuffers::Offset<ProxyDestConnectionRequest> Finish() {
    const auto end = fbb_.EndTable(start_);
    auto o = ::flatbuffers::Offset<ProxyDestConnectionRequest>(end);
    return o;
  }
};

inline ::flatbuffers::Offset<ProxyDestConnectionRequest> CreateProxyDestConnectionRequest(
    ::flatbuffers::FlatBufferBuilder &_fbb,
    uint32_t client_id = 0) {
  ProxyDestConnectionRequestBuilder builder_(_fbb);
  builder_.add_client_id(client_id);
  return builder_.Finish();
}

struct ProxyDestClientMessage FLATBUFFERS_FINAL_CLASS : private ::flatbuffers::Table {
  typedef ProxyDestClientMessageBuilder Builder;
  enum FlatBuffersVTableOffset FLATBUFFERS_VTABLE_UNDERLYING_TYPE {
    VT_CLIENT_ID = 4
  };
  uint32_t client_id() const {
    return GetField<uint32_t>(VT_CLIENT_ID, 0);
  }
  bool Verify(::flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<uint32_t>(verifier, VT_CLIENT_ID, 4) &&
           verifier.EndTable();
  }
};

struct ProxyDestClientMessageBuilder {
  typedef ProxyDestClientMessage Table;
  ::flatbuffers::FlatBufferBuilder &fbb_;
  ::flatbuffers::uoffset_t start_;
  void add_client_id(uint32_t client_id) {
    fbb_.AddElement<uint32_t>(ProxyDestClientMessage::VT_CLIENT_ID, client_id, 0);
  }
  explicit ProxyDestClientMessageBuilder(::flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  ::flatbuffers::Offset<ProxyDestClientMessage> Finish() {
    const auto end = fbb_.EndTable(start_);
    auto o = ::flatbuffers::Offset<ProxyDestClientMessage>(end);
    return o;
  }
};

inline ::flatbuffers::Offset<ProxyDestClientMessage> CreateProxyDestClientMessage(
    ::flatbuffers::FlatBufferBuilder &_fbb,
    uint32_t client_id = 0) {
  ProxyDestClientMessageBuilder builder_(_fbb);
  builder_.add_client_id(client_id);
  return builder_.Finish();
}

struct ProxyDestCloseConnection FLATBUFFERS_FINAL_CLASS : private ::flatbuffers::Table {
  typedef ProxyDestCloseConnectionBuilder Builder;
  enum FlatBuffersVTableOffset FLATBUFFERS_VTABLE_UNDERLYING_TYPE {
    VT_CLIENT_ID = 4
  };
  uint32_t client_id() const {
    return GetField<uint32_t>(VT_CLIENT_ID, 0);
  }
  bool Verify(::flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<uint32_t>(verifier, VT_CLIENT_ID, 4) &&
           verifier.EndTable();
  }
};

struct ProxyDestCloseConnectionBuilder {
  typedef ProxyDestCloseConnection Table;
  ::flatbuffers::FlatBufferBuilder &fbb_;
  ::flatbuffers::uoffset_t start_;
  void add_client_id(uint32_t client_id) {
    fbb_.AddElement<uint32_t>(ProxyDestCloseConnection::VT_CLIENT_ID, client_id, 0);
  }
  explicit ProxyDestCloseConnectionBuilder(::flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  ::flatbuffers::Offset<ProxyDestCloseConnection> Finish() {
    const auto end = fbb_.EndTable(start_);
    auto o = ::flatbuffers::Offset<ProxyDestCloseConnection>(end);
    return o;
  }
};

inline ::flatbuffers::Offset<ProxyDestCloseConnection> CreateProxyDestCloseConnection(
    ::flatbuffers::FlatBufferBuilder &_fbb,
    uint32_t client_id = 0) {
  ProxyDestCloseConnectionBuilder builder_(_fbb);
  builder_.add_client_id(client_id);
  return builder_.Finish();
}

struct ProxyDestinationMessage FLATBUFFERS_FINAL_CLASS : private ::flatbuffers::Table {
  typedef ProxyDestinationMessageBuilder Builder;
  enum FlatBuffersVTableOffset FLATBUFFERS_VTABLE_UNDERLYING_TYPE {
    VT_INNER_MESSAGE_TYPE = 4,
    VT_INNER_MESSAGE = 6,
    VT_APP_DATA = 8
  };
  SpanreedMessage::ProxyDestInnerMsg inner_message_type() const {
    return static_cast<SpanreedMessage::ProxyDestInnerMsg>(GetField<uint8_t>(VT_INNER_MESSAGE_TYPE, 0));
  }
  const void *inner_message() const {
    return GetPointer<const void *>(VT_INNER_MESSAGE);
  }
  template<typename T> const T *inner_message_as() const;
  const SpanreedMessage::ProxyDestConnectionRequest *inner_message_as_ProxyDestConnectionRequest() const {
    return inner_message_type() == SpanreedMessage::ProxyDestInnerMsg_ProxyDestConnectionRequest ? static_cast<const SpanreedMessage::ProxyDestConnectionRequest *>(inner_message()) : nullptr;
  }
  const SpanreedMessage::ProxyDestClientMessage *inner_message_as_ProxyDestClientMessage() const {
    return inner_message_type() == SpanreedMessage::ProxyDestInnerMsg_ProxyDestClientMessage ? static_cast<const SpanreedMessage::ProxyDestClientMessage *>(inner_message()) : nullptr;
  }
  const SpanreedMessage::ProxyDestCloseConnection *inner_message_as_ProxyDestCloseConnection() const {
    return inner_message_type() == SpanreedMessage::ProxyDestInnerMsg_ProxyDestCloseConnection ? static_cast<const SpanreedMessage::ProxyDestCloseConnection *>(inner_message()) : nullptr;
  }
  const ::flatbuffers::Vector<uint8_t> *app_data() const {
    return GetPointer<const ::flatbuffers::Vector<uint8_t> *>(VT_APP_DATA);
  }
  bool Verify(::flatbuffers::Verifier &verifier) const {
    return VerifyTableStart(verifier) &&
           VerifyField<uint8_t>(verifier, VT_INNER_MESSAGE_TYPE, 1) &&
           VerifyOffset(verifier, VT_INNER_MESSAGE) &&
           VerifyProxyDestInnerMsg(verifier, inner_message(), inner_message_type()) &&
           VerifyOffset(verifier, VT_APP_DATA) &&
           verifier.VerifyVector(app_data()) &&
           verifier.EndTable();
  }
};

template<> inline const SpanreedMessage::ProxyDestConnectionRequest *ProxyDestinationMessage::inner_message_as<SpanreedMessage::ProxyDestConnectionRequest>() const {
  return inner_message_as_ProxyDestConnectionRequest();
}

template<> inline const SpanreedMessage::ProxyDestClientMessage *ProxyDestinationMessage::inner_message_as<SpanreedMessage::ProxyDestClientMessage>() const {
  return inner_message_as_ProxyDestClientMessage();
}

template<> inline const SpanreedMessage::ProxyDestCloseConnection *ProxyDestinationMessage::inner_message_as<SpanreedMessage::ProxyDestCloseConnection>() const {
  return inner_message_as_ProxyDestCloseConnection();
}

struct ProxyDestinationMessageBuilder {
  typedef ProxyDestinationMessage Table;
  ::flatbuffers::FlatBufferBuilder &fbb_;
  ::flatbuffers::uoffset_t start_;
  void add_inner_message_type(SpanreedMessage::ProxyDestInnerMsg inner_message_type) {
    fbb_.AddElement<uint8_t>(ProxyDestinationMessage::VT_INNER_MESSAGE_TYPE, static_cast<uint8_t>(inner_message_type), 0);
  }
  void add_inner_message(::flatbuffers::Offset<void> inner_message) {
    fbb_.AddOffset(ProxyDestinationMessage::VT_INNER_MESSAGE, inner_message);
  }
  void add_app_data(::flatbuffers::Offset<::flatbuffers::Vector<uint8_t>> app_data) {
    fbb_.AddOffset(ProxyDestinationMessage::VT_APP_DATA, app_data);
  }
  explicit ProxyDestinationMessageBuilder(::flatbuffers::FlatBufferBuilder &_fbb)
        : fbb_(_fbb) {
    start_ = fbb_.StartTable();
  }
  ::flatbuffers::Offset<ProxyDestinationMessage> Finish() {
    const auto end = fbb_.EndTable(start_);
    auto o = ::flatbuffers::Offset<ProxyDestinationMessage>(end);
    return o;
  }
};

inline ::flatbuffers::Offset<ProxyDestinationMessage> CreateProxyDestinationMessage(
    ::flatbuffers::FlatBufferBuilder &_fbb,
    SpanreedMessage::ProxyDestInnerMsg inner_message_type = SpanreedMessage::ProxyDestInnerMsg_NONE,
    ::flatbuffers::Offset<void> inner_message = 0,
    ::flatbuffers::Offset<::flatbuffers::Vector<uint8_t>> app_data = 0) {
  ProxyDestinationMessageBuilder builder_(_fbb);
  builder_.add_app_data(app_data);
  builder_.add_inner_message(inner_message);
  builder_.add_inner_message_type(inner_message_type);
  return builder_.Finish();
}

inline ::flatbuffers::Offset<ProxyDestinationMessage> CreateProxyDestinationMessageDirect(
    ::flatbuffers::FlatBufferBuilder &_fbb,
    SpanreedMessage::ProxyDestInnerMsg inner_message_type = SpanreedMessage::ProxyDestInnerMsg_NONE,
    ::flatbuffers::Offset<void> inner_message = 0,
    const std::vector<uint8_t> *app_data = nullptr) {
  auto app_data__ = app_data ? _fbb.CreateVector<uint8_t>(*app_data) : 0;
  return SpanreedMessage::CreateProxyDestinationMessage(
      _fbb,
      inner_message_type,
      inner_message,
      app_data__);
}

inline bool VerifyProxyDestInnerMsg(::flatbuffers::Verifier &verifier, const void *obj, ProxyDestInnerMsg type) {
  switch (type) {
    case ProxyDestInnerMsg_NONE: {
      return true;
    }
    case ProxyDestInnerMsg_ProxyDestConnectionRequest: {
      auto ptr = reinterpret_cast<const SpanreedMessage::ProxyDestConnectionRequest *>(obj);
      return verifier.VerifyTable(ptr);
    }
    case ProxyDestInnerMsg_ProxyDestClientMessage: {
      auto ptr = reinterpret_cast<const SpanreedMessage::ProxyDestClientMessage *>(obj);
      return verifier.VerifyTable(ptr);
    }
    case ProxyDestInnerMsg_ProxyDestCloseConnection: {
      auto ptr = reinterpret_cast<const SpanreedMessage::ProxyDestCloseConnection *>(obj);
      return verifier.VerifyTable(ptr);
    }
    default: return true;
  }
}

inline bool VerifyProxyDestInnerMsgVector(::flatbuffers::Verifier &verifier, const ::flatbuffers::Vector<::flatbuffers::Offset<void>> *values, const ::flatbuffers::Vector<uint8_t> *types) {
  if (!values || !types) return !values && !types;
  if (values->size() != types->size()) return false;
  for (::flatbuffers::uoffset_t i = 0; i < values->size(); ++i) {
    if (!VerifyProxyDestInnerMsg(
        verifier,  values->Get(i), types->GetEnum<ProxyDestInnerMsg>(i))) {
      return false;
    }
  }
  return true;
}

inline const SpanreedMessage::ProxyDestinationMessage *GetProxyDestinationMessage(const void *buf) {
  return ::flatbuffers::GetRoot<SpanreedMessage::ProxyDestinationMessage>(buf);
}

inline const SpanreedMessage::ProxyDestinationMessage *GetSizePrefixedProxyDestinationMessage(const void *buf) {
  return ::flatbuffers::GetSizePrefixedRoot<SpanreedMessage::ProxyDestinationMessage>(buf);
}

inline bool VerifyProxyDestinationMessageBuffer(
    ::flatbuffers::Verifier &verifier) {
  return verifier.VerifyBuffer<SpanreedMessage::ProxyDestinationMessage>(nullptr);
}

inline bool VerifySizePrefixedProxyDestinationMessageBuffer(
    ::flatbuffers::Verifier &verifier) {
  return verifier.VerifySizePrefixedBuffer<SpanreedMessage::ProxyDestinationMessage>(nullptr);
}

inline void FinishProxyDestinationMessageBuffer(
    ::flatbuffers::FlatBufferBuilder &fbb,
    ::flatbuffers::Offset<SpanreedMessage::ProxyDestinationMessage> root) {
  fbb.Finish(root);
}

inline void FinishSizePrefixedProxyDestinationMessageBuffer(
    ::flatbuffers::FlatBufferBuilder &fbb,
    ::flatbuffers::Offset<SpanreedMessage::ProxyDestinationMessage> root) {
  fbb.FinishSizePrefixed(root);
}

}  // namespace SpanreedMessage

#endif  // FLATBUFFERS_GENERATED_PROXYDESTINATIONMESSAGE_SPANREEDMESSAGE_H_
