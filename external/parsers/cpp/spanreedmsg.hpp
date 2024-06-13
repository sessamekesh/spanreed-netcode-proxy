#ifndef _SPANREED_MSG_HPP_
#define _SPANREED_MSG_HPP_

#include <memory>
#include <optional>

namespace spanreed {

struct ConnectionResponse {
  bool Verdict;
};

enum class DestinationMessageType {
  ConnectionResponse,
  AppMessage,
  NONE,
};

struct DestinationMessage {
  std::uint32_t MagicNumber;
  std::uint8_t Version;

  std::uint32_t ClientId;

  DestinationMessageType MessageType;
  std::unique_ptr<ConnectionResponse> ConnectionResponseMsg;
  std::vector<std::uint8_t> AppData;
};

class SpanreedMessageParser {
 public:
  enum class ParseResult {
    OK,
    InvalidMagicNumber,
    InvalidVersion,
    InvalidMessageType,
    Underflow
  };
  enum class SerializeResult { OK, Overflow };

 public:
  SpanreedMessageParser(std::uint32_t magic_number = 0x52554259u,
                        std::uint8_t version = 0u,
                        std::size_t max_message_size = 1400u)
      : magic_number_(magic_number),
        version_(version),
        max_message_size_(max_message_size) {}

  ParseResult parse(const std::uint8_t* msg, size_t msg_len,
                    DestinationMessage* out) const {
    // TODO (sessamekesh): Author this
    if (msg_len < 9) {
      return ParseResult::Underflow;
    }

    size_t read_ptr = 0;
    std::uint32_t magic_number = 0, client_id = 0;
    ParseResult rsl = read_u32(msg, msg_len, &read_ptr, &magic_number);
    if (rsl != ParseResult::OK) {
      return rsl;
    }
    rsl = read_u32(msg, msg_len, &read_ptr, &client_id);
    if (rsl != ParseResult::OK) {
      return rsl;
    }

    std::uint8_t version_type_byte = msg[read_ptr];
    read_ptr++;

    std::uint8_t version = version_type_byte & 0xF0 >> 4;
    std::uint8_t msg_type = version_type_byte & 0xF;

    if (magic_number != magic_number_) {
      return ParseResult::InvalidMagicNumber;
    }

    if (version != version_) {
      return ParseResult::InvalidVersion;
    }

    out->ClientId = client_id;
    out->MagicNumber = magic_number;
    out->Version = version;

    switch (msg_type) {
      case 0x0:
        if (read_ptr >= msg_len) {
          return ParseResult::Underflow;
        }
        out->MessageType = DestinationMessageType::ConnectionResponse;
        out->ConnectionResponseMsg =
            std::make_unique<spanreed::ConnectionResponse>();
        out->ConnectionResponseMsg->Verdict = msg[read_ptr] > 0;
        read_ptr++;
        break;
      case 0x1:
        out->MessageType = DestinationMessageType::AppMessage;
        out->ConnectionResponseMsg = nullptr;
      default:
        return ParseResult::InvalidMessageType;
    }

    if (read_ptr >= msg_len) {
      return ParseResult::OK;
    }

    out->AppData.resize(msg_len - read_ptr);
    memcpy(&out->AppData[0], msg + read_ptr, msg_len - read_ptr);
    return ParseResult::OK;
  }

  SerializeResult serialize(const DestinationMessage& msg,
                            std::vector<uint8_t>* out) const {
    // TODO (sessamekesh): Author this
    return SerializeResult::OK;
  }

 private:
  ParseResult read_u32(const std::uint8_t* msg, size_t msg_len,
                       size_t* read_ptr, std::uint32_t* o) const {
    if (*read_ptr + 4 >= msg_len) {
      return ParseResult::Underflow;
    }

    // Little endian parsing...
    std::uint32_t a = msg[*read_ptr];
    std::uint32_t b = msg[*read_ptr + 1];
    std::uint32_t c = msg[*read_ptr + 2];
    std::uint32_t d = msg[*read_ptr + 3];

    *read_ptr += 4;
    *o = a | (b << 8) | (c << 16) | (d << 32);

    return ParseResult::OK;
  }

  SerializeResult write_u32(std::vector<uint8_t>& out, std::uint32_t val) {
    out.push_back(val & 0x000000FF);
    out.push_back((val & 0x0000FF) >> 8);
    out.push_back((val & 0x00FF) >> 16);
    out.push_back((val & 0xFF) >> 24);
    return SerializeResult::OK;
  }

 private:
  std::uint32_t magic_number_;
  std::uint8_t version_;
  std::size_t max_message_size_;
};

}  // namespace spanreed

#endif
