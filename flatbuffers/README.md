Flatbuffer files used for messaging with default WebSocket backend

Typescript generation:

flatc.exe --ts -I ../../ ..\..\connect_client_message.fbs ..\..\connect_client_verdict.fbs ..\..\connect_destination_verdict.fbs ..\..\destination_message.fbs ..\..\proxy_destination_message.fbs --ts-no-import-ext

## Why flatbuffers?

... Honestly, I sorta want to stop using them. The use of flatbuffers in this project evolved from the experiment code I was using from other (non-released!) projects and it ended up sticking around well into development of Spanreed.

I _love_ the high performance and "simplicity" of flatbuffers, but language bindings are piss-poor for everything but C++.

Seriously. The complete and total lack of Go validation for flatbuffers makes using them a complete _minefield_ here.

Protocol buffers are a decent fit too, not excellent but not horrible.

The best approach going forward for future versions of this library is probably to use a custom message format, since really the scope of structured data is preeeeety small. Flatbuffers works for now until I can be bothered to do that.