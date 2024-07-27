# Net Benchmark

Benchmark tool that demonstrates two things:

1. How to use Spanreed as a library in a custom proxy
2. Using custom message exchange formats

It's also useful for collecting some benchmarks around CPU+memory use under certian network conditions, packet loss, RTT between client <-> proxy and proxy <-> destination, and to form comparisons to client <-> destination direct WebSocket connections.

# Application Instructions

See each app's respective `README.MD` file for more detailed instructions

# Objectives

The main metrics I care about collecting:

1. RTT latency
   * Aggregates: slowest, 99th, 95th, 90th, 75th, median, fastest
2. Packet loss
3. Out-of-order packet receival
4. Proxy resource usage per client
   * Clock time
   * CPU load
   * RAM usage

To measure each of those, the following data must be captured:

1. Client send/recv time (for total RTT)
2. Destination send/recv time
3. Proxy send/recv for both `client -> dest` and `dest -> client` paths
4. Proxy goroutine runtime stats
   * Throughput + load (measured as pct time in processing vs. waiting)
   * Metrics graphs through whatever hosting provider actually runs the proxy
5. SequenceID / ACK fields
   * Monotonically increasing message ID scoped to sender
   * Last received client message sequence ID
   * 32-bit bitfield of previous 32 seen messages

### Experiment: `stability-and-latency`

#### Test Setup

Client fires off a stream of small messages (~50 bytes) to the server, server responds with a stream of its own.

Messages on each side contain a monotonically increasing 

Final message is sent from server to client until received, contains a few extra fields:
* IDs of OOO messages received
* IDs of duplicate messages received

Connection modes:
1. Spanreed WebTransport connection to server (browser client)
2. Direct UDP connection to server (native client)

The most important network topologies to test involve the proxy and destination servers being deployed nearby each other. Testing the client at different distances from the proxy/destination region should yield different and interesting results.

It's also worth experimenting with a proxy located further from both the client and destination, to test the case where a Spanreed deployment is run by a third-party in a separate zone for some reason.

There's quite a few potential zones, and the matrix of data that could be collected here is _massive_. Further work can be done here, but I think the following zonal combinations should be good for an initial study:

1. US West <-> US East
2. US West <-> US Central
3. US Central <-> Tokyo
4. US Central <-> Sydney
5. US Central <-> Europe
6. US Central <-> South America
7. Tokyo <-> Europe

And finally, a couple toy cases that probably won't have much difference but are fun to validate this use case with: running this from an Android, an iPhone, a Raspberry Pi, and a Tesla screen.

#### Results

TODO

### Experiment: `proxy-throughput`

Much easier test, web-only, fires simple messages. Load test online.
