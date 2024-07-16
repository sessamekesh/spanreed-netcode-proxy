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
#include <random>
#include <shared_mutex>
#include <thread>
#include <unordered_map>

struct ConnectedClient {
  sockaddr_in spanreed_client_addr{};  // Will probably be the same for all
                                       // connected clients...
  std::uint32_t spanreed_client_id{};

  std::uint64_t last_send_time{};

  std::uint64_t last_rotate_colors_time{};

  std::string client_name{};

  HelloSpanreed::Color dot_color{};
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
std::chrono::high_resolution_clock::time_point gTpStart;

struct DrawnDot {
  float x;
  float y;
  float t;
  float radius;
  HelloSpanreed::Color color;
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

static HelloSpanreed::Color get_client_color() {
  static int next_color = 0;
  next_color++;

  switch (next_color % 5) {
    case 0:
      return HelloSpanreed::Color_Blue;
    case 1:
      return HelloSpanreed::Color_FireOrange;
    case 2:
      return HelloSpanreed::Color_Green;
    case 3:
      return HelloSpanreed::Color_Indigo;
    case 4:
    default:
      return HelloSpanreed::Color_Red;
  }
}

static bool cmp_addr(const sockaddr_in& a, const sockaddr_in& b) {
  if (a.sin_addr.S_un.S_addr != b.sin_addr.S_un.S_addr) return false;
  if (a.sin_port != b.sin_port) return false;

  return true;
}

constexpr long long kClientSendFrequencyUs = 50000;
constexpr long long kClientRotateColorFrequencyUs = 8000000;

int main() {
  std::cout << "-------- Hello Spanreed UDP Server --------" << std::endl;

  gTpStart = std::chrono::high_resolution_clock::now();
  gGetTimestamp = []() {
    auto now = std::chrono::high_resolution_clock::now();
    return std::chrono::duration_cast<std::chrono::microseconds>(now - gTpStart)
        .count();
  };

  WSADATA wsa{};
  if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
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
  WorldState world_state{};

  std::random_device rd;
  std::mt19937 rng(rd());
  std::normal_distribution<float> radius_dist(25.f, 8.f);

  auto last_frame_time = gGetTimestamp();
  while (bRunning) {
    const auto this_frame_time = gGetTimestamp();
    auto dt_s = (this_frame_time - last_frame_time) / 1000000.f;
    last_frame_time = this_frame_time;

    for (auto& dot : world_state.dots) {
      dot.t -= dt_s;
    }

    world_state.dots.erase(
        std::remove_if(world_state.dots.begin(), world_state.dots.end(),
                       [](const DrawnDot& d) { return d.t <= 0.f; }),
        world_state.dots.end());

    for (int i = 0;
         i < 25 && gClientMessageQueue.try_dequeue(msg) && msg.msg != nullptr;
         i++) {
      ConnectedClient client{};
      {
        std::shared_lock l(gMutConnectedClients);
        auto it = gConnectedClients.find(msg.client_id);
        if (it == gConnectedClients.end()) continue;
        client = it->second;
      }

      if (auto* click_msg = msg.msg->user_message_as_UserClickMessage()) {
        std::cout << "UserClickMsg: (" << msg.client_id
                  << "): " << click_msg->x() << ", " << click_msg->y()
                  << std::endl;
        DrawnDot new_dot{};
        new_dot.t = 5.f;
        new_dot.radius = radius_dist(rng);
        new_dot.x = click_msg->x();
        new_dot.y = click_msg->y();
        new_dot.color = client.dot_color;
        world_state.dots.push_back(new_dot);
      } else if (auto* broadcast_msg =
                     msg.msg->user_message_as_UserChatMessage()) {
        std::cout << "UserChatMsg: (" << msg.client_id
                  << "): " << broadcast_msg->text()->str() << std::endl;
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
          flatbuffers::FlatBufferBuilder sfbb{};
          auto proxy_msg_ptr = SpanreedMessage::CreateProxyMessage(
              sfbb, target.spanreed_client_id);
          auto app_data = sfbb.CreateVector(inner_msg_data, inner_msg_len);
          auto dest_msg = SpanreedMessage::CreateDestinationMessage(
              sfbb, SpanreedMessage::InnerMsg_ProxyMessage,
              proxy_msg_ptr.Union(), app_data);

          sfbb.Finish(dest_msg);

          uint8_t* dest_msg_data = sfbb.GetBufferPointer();
          size_t dest_msg_len = sfbb.GetSize();

          sendto(
              server_socket, reinterpret_cast<const char*>(dest_msg_data),
              dest_msg_len, 0x00,
              reinterpret_cast<const sockaddr*>(&target.spanreed_client_addr),
              sizeof(target.spanreed_client_addr));
        }
      }
    }

    flatbuffers::FlatBufferBuilder app_data_builder{};
    std::vector<HelloSpanreed::Dot> msg_dots;
    for (const auto& logical_dot : world_state.dots) {
      // TODO (sessamekesh): Assign color on connect, round-robin starting at
      //  random color (or instead of random, based on first connected client
      //  name?)
      msg_dots.push_back(HelloSpanreed::Dot(
          logical_dot.x, logical_dot.y, logical_dot.radius, logical_dot.color));
    }
    auto dots_v = app_data_builder.CreateVectorOfStructs(msg_dots);
    auto game_state_msg =
        HelloSpanreed::CreateServerGameStateMessage(app_data_builder, dots_v);
    auto app_data_v = HelloSpanreed::CreateServerMessage(
        app_data_builder, HelloSpanreed::Message_ServerGameStateMessage,
        game_state_msg.Union());
    app_data_builder.Finish(app_data_v);
    const uint8_t* app_data = app_data_builder.GetBufferPointer();
    size_t app_data_len = app_data_builder.GetSize();

    {
      std::lock_guard l(gMutConnectedClients);
      for (auto& client : gConnectedClients) {
        if (this_frame_time - client.second.last_rotate_colors_time >=
            ::kClientRotateColorFrequencyUs) {
          client.second.last_rotate_colors_time = this_frame_time;
          client.second.dot_color = get_client_color();
        }

        if (this_frame_time - client.second.last_send_time >=
            ::kClientSendFrequencyUs) {
          client.second.last_send_time = this_frame_time;

          flatbuffers::FlatBufferBuilder fbb;
          auto proxy_msg = SpanreedMessage::CreateProxyMessage(
              fbb, client.second.spanreed_client_id);
          auto app_data_v = fbb.CreateVector(app_data, app_data_len);
          auto dest_ptr = SpanreedMessage::CreateDestinationMessage(
              fbb, SpanreedMessage::InnerMsg_ProxyMessage, proxy_msg.Union(),
              app_data_v);
          SpanreedMessage::FinishDestinationMessageBuffer(fbb, dest_ptr);

          uint8_t* dest_msg_data = fbb.GetBufferPointer();
          size_t dest_msg_len = fbb.GetSize();

          sendto(server_socket, reinterpret_cast<const char*>(dest_msg_data),
                 dest_msg_len, 0x00,
                 reinterpret_cast<const sockaddr*>(
                     &client.second.spanreed_client_addr),
                 sizeof(client.second.spanreed_client_addr));
        }
      }
    }
  }

  t.join();

  return 0;
}

void handle_recv_message(SOCKET sock, char* msg, int msglen, sockaddr_in src) {
  flatbuffers::Verifier::Options opts{};
  opts.check_alignment = true;
  opts.check_nested_flatbuffers = true;
  opts.max_depth = 5;
  flatbuffers::Verifier v(reinterpret_cast<std::uint8_t*>(msg), msglen, opts);
  if (!SpanreedMessage::VerifyProxyDestinationMessageBuffer(v)) {
    std::cerr << "Invalid message received from client at "
              << inet_ntoa(src.sin_addr) << " port " << src.sin_port
              << std::endl;
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
  flatbuffers::Verifier::Options opts{};
  opts.check_alignment = true;
  opts.check_nested_flatbuffers = true;
  opts.max_depth = 5;
  flatbuffers::Verifier v(app_data, app_data_len, opts);
  if (!HelloSpanreed::VerifyClientMessageBuffer(v)) {
    std::cout << "--- Not a client message buffer, aborting connection request"
              << std::endl;
    return;
  }
  auto* client_message = HelloSpanreed::GetClientMessage(app_data)
                             ->user_message_as_UserConnectMessage();
  if (client_message == nullptr) {
    std::cout << "--- Does not have UserConnectMessate, aborting" << std::endl;
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

      SpanreedMessage::ConnectionVerdictBuilder cbb(fbb);
      cbb.add_accepted(false);
      cbb.add_client_id(request->client_id());
      auto cv_offset = cbb.Finish();

      SpanreedMessage::DestinationMessageBuilder dmb(fbb);
      dmb.add_msg_type(SpanreedMessage::InnerMsg_ConnectionVerdict);
      dmb.add_msg(cv_offset.Union());

      auto msg = dmb.Finish();
      fbb.Finish(msg);

      const std::uint8_t* buf = fbb.GetBufferPointer();
      size_t buf_len = fbb.GetSize();

      sendto(sock, reinterpret_cast<const char*>(buf), buf_len, 0x00,
             reinterpret_cast<sockaddr*>(&src), sizeof(src));
      std::cout << "Rejecting client that's already connected" << std::endl;

      return;
    }
  }

  std::lock_guard l(gMutConnectedClients);
  ConnectedClient client_data{};
  client_data.last_send_time = gGetTimestamp();
  client_data.spanreed_client_addr = src;
  client_data.spanreed_client_id = request->client_id();
  client_data.client_name = client_message->name()->str();
  client_data.dot_color = get_client_color();

  gConnectedClients.insert({request->client_id(), client_data});

  flatbuffers::FlatBufferBuilder fbb;

  SpanreedMessage::ConnectionVerdictBuilder cbb(fbb);
  cbb.add_accepted(true);
  cbb.add_client_id(request->client_id());
  auto cv_offset = cbb.Finish();

  SpanreedMessage::DestinationMessageBuilder dmb(fbb);
  dmb.add_msg_type(SpanreedMessage::InnerMsg_ConnectionVerdict);
  dmb.add_msg(cv_offset.Union());

  auto msg = dmb.Finish();
  fbb.Finish(msg);

  const std::uint8_t* buf = fbb.GetBufferPointer();
  size_t buf_len = fbb.GetSize();

  sendto(sock, reinterpret_cast<const char*>(buf), buf_len, 0x00,
         reinterpret_cast<sockaddr*>(&src), sizeof(src));
  std::cout << "Accepting client with ID " << request->client_id()
            << ", nickname=" << client_name << std::endl;
}

void close_connection_request(
    SOCKET sock, const SpanreedMessage::ProxyDestCloseConnection* request,
    sockaddr_in src) {
  ConnectedClient client{};

  std::cout << "Received close connection request for "
            << inet_ntoa(src.sin_addr) << " at port " << src.sin_port
            << ", for client " << request->client_id() << std::endl;

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

  flatbuffers::Verifier::Options opts{};
  opts.check_alignment = true;
  opts.check_nested_flatbuffers = true;
  opts.max_depth = 5;
  flatbuffers::Verifier v(app_data_bytes, app_data_len, opts);
  if (!HelloSpanreed::VerifyClientMessageBuffer(v)) {
    return;
  }

  auto* client_msg = HelloSpanreed::GetClientMessage(app_data_bytes);
  std::vector<std::uint8_t> cpy(app_data_len);
  memcpy(&cpy[0], app_data_bytes, app_data_len);
  QueuedClientMessage qm{msg->client_id(), std::move(cpy), nullptr};
  qm.msg = HelloSpanreed::GetClientMessage(&qm._buffer[0]);
  gClientMessageQueue.enqueue(std::move(qm));
}
