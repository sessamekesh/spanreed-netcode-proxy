Flatbuffer files used for messaging with default WebSocket backend

Typescript generation:

flatc.exe --ts -I ../../ ..\..\connect_client_message.fbs ..\..\connect_client_verdict.fbs ..\..\connect_destination_verdict.fbs ..\..\destination_message.fbs ..\..\proxy_destination_message.fbs --ts-no-import-ext