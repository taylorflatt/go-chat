# Remote shell
A chat client-server based remote shell implemented in Go using gRPC.

## Usage
* First, run the server by going to the Server directory and typing `go run server.go`.
* Next, run the client by going to the Client directory and typing `go run client.go`. 
* Then, enter in the IP:Port to the server.
* Now, choose whether you would like to group chat or single chat.
* Next, type in the client(s) you would like to connect to.
* Finally, you can type any messages that you would like to send/receive.

## Notes
To disconnect from the server, press ctrl+c or type !exit (hit enter) and the client will disconnect from the server.

This client/server assumes a 12021 server port. This can be changed in the server.go file near the top.