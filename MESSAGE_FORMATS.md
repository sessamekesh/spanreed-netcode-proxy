All messages to/from a Spanreed server are prefixed with the following data:

* `Magic Number` - 32-bit prefix identifying your Spanreed instance. Defaults to `0x52554259` (utf-8 encoding of "RUBY").
* `Version` - 4-bit message API version to allow changing schemas in the same namespace. Defaults to `0`.
* `MsgType` - 4-bit number identifying the message type of the rest of the payload _(context-dependent - 0x00 encodes a different message type for messages sent spanreed to client than client to spanreed)_.

## Client to Spanreed Proxy

TODO (sessamekesh): Include some client decoder files for this (C++, TypeScript, Go, etc...)

```
ClientConnectionRequest
+----------------------------+------------------+---------+
| ConnectionStringByteLength | ConnectionString | AppData |
|          16 bits           |                  |         |
+----------------------------+------------------+---------+

* MsgType = 0b0000
```

Example connection string formats supported out of the box:
* **UDP**: `udp:{{hostname}}:{{port}}`
* **TCP**: `tcp:{{hostname}}:{{port}}`

You may choose to support additional transport layers and are encouraged to use the connection string format `transport:address` to do so.

## Spanreed Proxy to Destination

```
SpanreedConnectionRequest
+--------------------+---------+
| Spanreed Client ID | AppData |
|       32 bits      |         |
+--------------------+---------+

* MsgType = 0b0100
```

## Destination to Spanreed Proxy

```
DestinationVerdict
+--------------------+---------+-----------+---------+
| Spanreed Client ID | Verdict | (padding) | AppData |
|       32 bits      |  1 bit  |   7 bits  |         |
+--------------------+---------+-----------+---------+
```

## Spanreed Proxy to Client

```
SpanreedVerdict
+--------------+-----------+-----------+-----------+---------+
| IsProxyError | IsTimeout |  Verdict  | (padding) | AppData |
|    1 bit     |   1 bit   |   1 bit   |  5 bits   |         |
+--------------+-----------+-----------+-----------+---------+
```
