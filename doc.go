package goChat

/*
Go-Chat is a straight forward client-server chat app implemented using gRPC with protocol buffers. The code is well documented to help individuals learn as well as contribute.package go-chat


To begin, start the server:
	cd [PATH_TO_SRC]/Server
	go run server.go

	// Alternatively:
	go build

Now, you can start spinning up as many clients as you wish:
	cd [PATH_TO_SRC]/client
	go run client.go menu.go

	// Alternatively:
	go build

At this point, you will be greated with a prompt asking for the server. Below is a sample run:

	Welcome to Go-Chat!
	------------------------------------------
	In order to begin chatting, you must first chose a server and username. It cannot be one
	that is already in user on the server. Remember, your username only lasts for as long as
	you are logged into the server!

	Please specify the server IP: localhost:12021

	You have successfully connected to localhost:12021! To disconnect, hit ctrl+c or type !exit.

	Enter your username: user1

	Welcome user1! There are currently 1 member(s) logged in and 0 group(s).
	------------------------------------------
	Main Menu

	1) Create a Group
	2) View Group Options
	3) Exit Chat

	3) Exit Chat

	Main> 1

	Enter the name of the group or type !back to go back to the main menu.
	Join> group1

	Created and joined group named group1

	You are now chatting in group1.
	------------------------------------------
	Current Members: user1

	Hello! Anyone here?

	!exit
	exit status 1
	PS C:\Users\Taylor\Work\src\github.com\taylorflatt\go-chat\Client>

I set my username to user1, created a group called group1, and sent a message "Hello! Anyone here?". I then typed !exit to quit the program.

*/
