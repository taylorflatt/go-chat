# Remote shell
A client-server chat app implemented in Go using gRPC.

## Usage
* First, run the server by going to the Server directory and typing `go run server.go`.
* Next, run the client by going to the Client directory and typing `go run client.go`. 
* Then, enter in the IP:Port to the server.
* Now, choose whether you would like to group chat or single chat.
* Next, type in the client(s) you would like to connect to.
* Finally, you can type any messages that you would like to send/receive.

## Known Bugs
* If you type ctrl + c while in a chat, it will crash the server. This bug is known and I just need to implement the fix for it. To properly exit the chat, type `!exit` instead.
* The UI is clunky at the moment. I plan on spending a bit of time on UX in the near future. It goes without saying that cleaning up the code is also a top priority.

## Notes
To disconnect from the server, press ctrl+c or type `!exit` (hit enter) and the client will disconnect from the server.

This client/server assumes a 12021 server port. This can be changed in the server.go file near the top.