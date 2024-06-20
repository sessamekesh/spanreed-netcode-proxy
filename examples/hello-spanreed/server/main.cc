// Must go before winsock
#include <spanreed_messages/destination_message_generated.h>
#include <spanreed_messages/proxy_destination_message_generated.h>
// ...

#include <WinSock2.h>
#include <concurrentqueue.h>
#include <messages/client_message_generated.h>
#include <messages/server_message_generated.h>

#include <chrono>
#include <iostream>
#include <shared_mutex>
#include <thread>
#include <unordered_map>

struct ConnectedClient {
  sockaddr_in spanreed_client_addr{};  // Will probably be the same for all
                                       // connected clients...
  std::uint32_t spanreed_client_id{};

  std::uint64_t last_send_time{};

  std::string client_name{};
};

struct QueuedClientMessage {
  std::uint32_t client_id;
  std::vector<std::uint8_t> _buffer;
  const HelloSpanreed::ClientMessage* msg;
};

std::shared_mutex gMutConnectedClients;
std::unordered_map<std::uint32_t, ConnectedClient> gConnectedClients{};

moodycamel::ConcurrentQueue<QueuedClientMessage> gClientMessageQueue;

long long (*gGetTimestamp)() = nullptr;

struct DrawnDot {
  float x;
  float y;
  float t;
};

struct WorldState {
  std::vector<DrawnDot> dots;
};

void handle_recv_message(SOCKET sock, char* msg, int msglen, sockaddr_in src);
void connection_request(
    SOCKET sock, const SpanreedMessage::ProxyDestConnectionRequest* request,
    const std::uint8_t* app_data, size_t app_data_len, sockaddr_in src);
void close_connection_request(
    SOCKET sock, const SpanreedMessage::ProxyDestCloseConnection* request,
    sockaddr_in src);
void handle_client_message(SOCKET sock,
                           const SpanreedMessage::ProxyDestClientMessage* msg,
                           const std::uint8_t* app_data_bytes,
                           size_t app_data_len, sockaddr_in src);

static bool cmp_addr(const sockaddr_in& a, const sockaddr_in& b) {
  if (a.sin_addr.S_un.S_addr != b.sin_addr.S_un.S_addr) return false;
  if (a.sin_port != b.sin_port) return false;

  return true;
}

int main() {
  std::cout << "-------- Hello Spanreed UDP Server --------" << std::endl;

  auto tp_start = std::chrono::high_resolution_clock::now();
  auto gGetTimestamp = [tp_start]() {
    auto now = std::chrono::high_resolution_clock::now();
    return std::chrono::duration_cast<std::chrono::microseconds>(now - tp_start)
        .count();
  };

  WSADATA wsa{};
  if (WSAStartup(MAKEWORD(2, 2), &wsa) == 0) {
    std::cerr << "Failed to startup WinSock - error code " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  SOCKET server_socket{};
  if ((server_socket = socket(AF_INET, SOCK_DGRAM, 0)) == INVALID_SOCKET) {
    std::cerr << "Could not create server socket: " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  sockaddr_in server_addr{};
  server_addr.sin_family = AF_INET;
  server_addr.sin_addr.s_addr = INADDR_ANY;
  server_addr.sin_port = htons(30001);

  if (bind(server_socket, (sockaddr*)&server_addr, sizeof(server_addr)) ==
      SOCKET_ERROR) {
    std::cerr << "Failed to bind socket, code: " << WSAGetLastError()
              << std::endl;
    return -1;
  }

  std::cout << "Listening on port 30001!" << std::endl;

  bool bRunning = true;
  std::thread t([&bRunning, &server_socket]() {
    while (bRunning) {
      constexpr size_t BUFLEN = 1024;
      char message[BUFLEN] = {};

      int message_len;
      int slen = sizeof(sockaddr_in);
      sockaddr_in recv_addr{};
      if ((message_len = recvfrom(server_socket, message, BUFLEN, 0x0,
                                  (sockaddr*)&recv_addr, &slen)) ==
          SOCKET_ERROR) {
        std::cout << "recvfrom() failed with error code: " << WSAGetLastError()
                  << std::endl;
        return -1;
      }

      std::cout << "Received packet (" << message_len << " bytes) from "
                << inet_ntoa(recv_addr.sin_addr) << " "
                << ntohs(recv_addr.sin_port) << std::endl;

      handle_recv_message(server_socket, message, message_len, recv_addr);
    }
  });

  QueuedClientMessage msg{};

  while (bRunning) {
    if (!gClientMessageQueue.try_dequeue(msg)) {
      continue;
    }

    if (msg.msg == nullptr) continue;

    ConnectedClient client{};
    {
      std::shared_lock l(gMutConnectedClients);
      auto it = gConnectedClients.find(msg.client_id);
      if (it == gConnectedClients.end()) continue;
      client = it->second;
    }

    if (auto* click_msg = msg.msg->user_message_as_UserClickMessage()) {
      // TODO (sessamekesh): Add new dot!
    } else if (auto* broadcast_msg =
                   msg.msg->user_message_as_UserChatMessage()) {
      // TODO (sessamekesh): Broadcast message to all clients!
      flatbuffers::FlatBufferBuilder inner_msg_fbb;
      auto user_text = inner_msg_fbb.CreateString(client.client_name);
      auto broadcast_text =
          inner_msg_fbb.CreateString(broadcast_msg->text()->str());

      auto server_chat_msg = HelloSpanreed::CreateServerChatMessage(
                                 inner_msg_fbb, user_text, broadcast_text)
                                 .Union();
      auto app_message = HelloSpanreed::CreateServerMessage(
          inner_msg_fbb, HelloSpanreed::Message_ServerChatMessage,
          server_chat_msg);
      inner_msg_fbb.Finish(app_message);

      uint8_t* inner_msg_data = inner_msg_fbb.GetBufferPointer();
      size_t inner_msg_len = inner_msg_fbb.GetSize();

      std::vector<ConnectedClient> broadcast_targets;
      {
        std::shared_lock l(gMutConnectedClients);
        for (auto& [a, b] : gConnectedClients) {
          broadcast_targets.push_back(b);
        }
      }

      for (const auto& target : broadcast_targets) {
        // TODO (sessamekesh): Assemble message
        // TODO (sessamekesh): Broadcast message
      }
    }
    // TODO (sessamekesh): Handle incoming messages
    //  - Add new dots to sim
    //  - Append list of messages to broadcast
    // TODO (sessamekesh): Lifetime expire dots
    // TODO (sessamekesh): Broadcast messages to clients
  }

  t.join();

  return 0;
}

void handle_recv_message(SOCKET sock, char* msg, int msglen, sockaddr_in src) {
  if (!SpanreedMessage::VerifyProxyDestinationMessageBuffer(
          flatbuffers::Verifier(reinterpret_cast<std::uint8_t*>(msg),
                                msglen))) {
    // TODO (sessamekesh): Error state
    return;
  }

  auto* proxy_destination_msg =
      SpanreedMessage::GetProxyDestinationMessage(msg);
  switch (proxy_destination_msg->inner_message_type()) {
    case SpanreedMessage::ProxyDestInnerMsg_ProxyDestConnectionRequest: {
      auto* req =
          proxy_destination_msg->inner_message_as_ProxyDestConnectionRequest();
      connection_request(sock, req, proxy_destination_msg->app_data()->data(),
                         proxy_destination_msg->app_data()->Length(), src);
      return;
    }
    case SpanreedMessage::ProxyDestInnerMsg_ProxyDestCloseConnection: {
      auto* req =
          proxy_destination_msg->inner_message_as_ProxyDestCloseConnection();
      close_connection_request(sock, req, src);
      return;
    }
    case SpanreedMessage::ProxyDestInnerMsg_ProxyDestClientMessage: {
      auto* req =
          proxy_destination_msg->inner_message_as_ProxyDestClientMessage();
      handle_client_message(sock, req,
                            proxy_destination_msg->app_data()->data(),
                            proxy_destination_msg->app_data()->Length(), src);
      return;
    }
    default:
      return;
  }
}

void connection_request(
    SOCKET sock, const SpanreedMessage::ProxyDestConnectionRequest* request,
    const std::uint8_t* app_data, size_t app_data_len, sockaddr_in src) {
  if (!HelloSpanreed::VerifyClientMessageBuffer(
          flatbuffers::Verifier(app_data, app_data_len))) {
    return;
  }
  auto* client_message = HelloSpanreed::GetClientMessage(app_data)
                             ->user_message_as_UserConnectMessage();
  if (client_message == nullptr) {
    return;
  }

  auto client_name = client_message->name()->str();

  {
    std::shared_lock l(gMutConnectedClients);
    if (gConnectedClients.find(request->client_id()) !=
            gConnectedClients.end() ||
        gConnectedClients.size() > 20) {
      // Client ID already exists, send a FALSE verdict
      flatbuffers::FlatBufferBuilder fbb;
      SpanreedMessage::DestinationMessageBuilder dmb(fbb);
      dmb.add_msg_type(SpanreedMessage::InnerMsg_ConnectionVerdict);

      SpanreedMessage::ConnectionVerdictBuilder cbb(fbb);
      cbb.add_accepted(false);
      cbb.add_client_id(request->client_id());
      auto cv_offset = cbb.Finish();
      dmb.add_msg(cv_offset.Union());

      auto msg = dmb.Finish();
      fbb.Finish(msg);

      const std::uint8_t* buf = fbb.GetBufferPointer();
      size_t buf_len = fbb.GetSize();

      sendto(sock, reinterpret_cast<const char*>(buf), buf_len, 0x00,
             reinterpret_cast<sockaddr*>(&src), sizeof(src));

      return;
    }
  }

  std::lock_guard l(gMutConnectedClients);
  ConnectedClient client_data{};
  client_data.last_send_time = gGetTimestamp();
  client_data.spanreed_client_addr = src;
  client_data.spanreed_client_id = request->client_id();
  client_data.client_name = client_message->name()->str();

  gConnectedClients.insert({request->client_id(), client_data});

  flatbuffers::FlatBufferBuilder fbb;
  SpanreedMessage::DestinationMessageBuilder dmb(fbb);
  dmb.add_msg_type(SpanreedMessage::InnerMsg_ConnectionVerdict);

  SpanreedMessage::ConnectionVerdictBuilder cbb(fbb);
  cbb.add_accepted(true);
  cbb.add_client_id(request->client_id());
  auto cv_offset = cbb.Finish();
  dmb.add_msg(cv_offset.Union());

  auto msg = dmb.Finish();
  fbb.Finish(msg);

  const std::uint8_t* buf = fbb.GetBufferPointer();
  size_t buf_len = fbb.GetSize();

  sendto(sock, reinterpret_cast<const char*>(buf), buf_len, 0x00,
         reinterpret_cast<sockaddr*>(&src), sizeof(src));
}

void close_connection_request(
    SOCKET sock, const SpanreedMessage::ProxyDestCloseConnection* request,
    sockaddr_in src) {
  ConnectedClient client{};

  {
    std::shared_lock l(gMutConnectedClients);
    auto it = gConnectedClients.find(request->client_id());
    if (it == gConnectedClients.end()) {
      return;
    }
    client = it->second;
  }

  if (!(cmp_addr(src, client.spanreed_client_addr))) {
    return;
  }

  std::lock_guard l(gMutConnectedClients);
  gConnectedClients.erase(request->client_id());
}

void handle_client_message(SOCKET sock,
                           const SpanreedMessage::ProxyDestClientMessage* msg,
                           const std::uint8_t* app_data_bytes,
                           size_t app_data_len, sockaddr_in src) {
  ConnectedClient client{};

  {
    std::shared_lock l(gMutConnectedClients);
    auto it = gConnectedClients.find(msg->client_id());
    if (it == gConnectedClients.end()) {
      return;
    }
    client = it->second;
  }

  if (!HelloSpanreed::VerifyClientMessageBuffer(
          flatbuffers::Verifier(app_data_bytes, app_data_len))) {
    return;
  }

  auto* client_msg = HelloSpanreed::GetClientMessage(app_data_bytes);
  std::vector<std::uint8_t> cpy(app_data_len);
  memcpy(&cpy[0], app_data_bytes, app_data_len);
  QueuedClientMessage qm{msg->client_id(), std::move(cpy), nullptr};
  qm.msg = HelloSpanreed::GetClientMessage(&qm._buffer[0]);
  gClientMessageQueue.enqueue(std::move(qm));
}
