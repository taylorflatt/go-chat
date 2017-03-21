# Remote shell [![Build Status](https://travis-ci.org/taylorflatt/go-chat.svg?branch=master)](https://travis-ci.org/taylorflatt/go-chat)
A client-server chat app implemented in Go using gRPC.

## Usage
### Server
Start the server by running `go run server.go`. Alternatively, you can run `go build` while in the Server directory.
### Client
Start the client(s) by running `go run client.go menu.go`. Alternatively, you can run `go build` while in the Client directory.

To run navigate the client: 
* First enter the server ip:port exactly. It is likely `localhost:12021` unless the port at the top of server.go has changed.
* Then you'll enter a username that is yours for the session.
* Finally, you are greeted by the menu system which will allow you to create, join, or view other members and groups.

## Known Bugs
* None currently. If you run into any problems, please don't hesistate to create an issue.

## Notes
* To disconnect from the server, press ctrl+c or type `!exit` (hit enter) and the client will disconnect from the server.
* To move backwards in the menu system, you can type `!back` (hit enter).
* This client/server assumes a 12021 server port. This can be changed in the server.go file near the top.

## Future Ideas
* The !leave function to leave a chat and go back to the main menu.
* Encryption on chat channels.
* Ability to send files.
