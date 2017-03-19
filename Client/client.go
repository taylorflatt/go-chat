package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// ControlExitEarly handles any interrupts prior to joining a group.
// Note: This thread will die once the client joins a group.
// It doesn't return anything.
func ControlExitEarly(w chan os.Signal, c pb.ChatClient, q chan bool, u string) {

	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-w:
			if sig == os.Interrupt {
				c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
				os.Exit(1)
			}

			return
		case <-q:
			return
		}
	}
}

// ControlExitLate handles any interrupts after to joining a group.
// It doesn't return anything.
func ControlExitLate(w chan os.Signal, c pb.ChatClient, stream pb.Chat_RouteChatClient, u string, g string) {

	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)
	sig := <-w

	if sig == os.Interrupt {
		stream.Send(&pb.ChatMessage{Sender: u, Receiver: g, Message: u + " left chat!\n"})
		ExitChat(c, stream, u, g)
	}
}

// ExitChat handles removing the client from the server and exiting the program.
// It doesn't return anything.
func ExitChat(c pb.ChatClient, stream pb.Chat_RouteChatClient, u string, g string) {

	c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
	os.Exit(1)
}

// ListenToClient listens to the client for input and adds that input to the sQueue with
// the username of the sender, group name, and the message.
// It doesn't return anything.
func ListenToClient(sQueue chan pb.ChatMessage, reader *bufio.Reader, uName string, gName string) {
	for {
		msg, _ := reader.ReadString('\n')
		sQueue <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
	}
}

// ReceiveMessages listens on the client's (NOT the client's group) stream and adds any incoming
// message to the client's inbox.
// It doesn't return anything.
func ReceiveMessages(stream pb.Chat_RouteChatClient, inbox chan pb.ChatMessage) {
	for {
		msg, _ := stream.Recv()
		inbox <- *msg
	}
}

// DisplayCurrentMembers displays the members who are currently in the group chat.
// It doesn't return anything.
func DisplayCurrentMembers(c pb.ChatClient, g string) {

	m, _ := c.GetGroupClientList(context.Background(), &pb.GroupInfo{GroupName: g})
	if len(m.Clients) > 0 {
		fmt.Print("Current Members: ")
		for i := 0; i < len(m.Clients); i++ {
			if i == len(m.Clients)-1 {
				fmt.Print(m.Clients[i])
			} else {
				fmt.Print(m.Clients[i] + ", ")
			}
		}
		AddSpacing(2)
	}
}

func main() {

	r := bufio.NewReader(os.Stdin)

	var uName string // Client username
	var gName string // Client's chat group

	// DEBUG ONLY:
	a := "localhost:12021"
	// UNCOMMENT AFTER DEBUG
	//a := SetServer(r)
	// END UNCOMMENT

	// Set up a connection to the server.
	conn, err := grpc.Dial(a, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	} else {
		fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type !exit.\n\n", a)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := pb.NewChatClient(conn)

	uName = SetName(c, r)
	w := make(chan os.Signal, 1) // Watch for ctrl+c
	q := make(chan bool)         // Quit sig
	go ControlExitEarly(w, c, q, uName)

	gName, err = TopMenu(c, r, uName)

	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	AddSpacing(1)
	fmt.Println("You are now chatting in " + gName + ".")
	Frame()

	stream, serr := c.RouteChat(context.Background())
	q <- true
	go ControlExitLate(w, c, stream, uName, gName)

	// TODO: Find out why the first message is always dropped so an empty message needn't be sent.
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: ""})
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: "joined chat!\n"})

	DisplayCurrentMembers(c, gName)

	if serr != nil {
		fmt.Print(serr)
	} else {

		sQueue := make(chan pb.ChatMessage, 100)
		go ListenToClient(sQueue, r, uName, gName)

		inbox := make(chan pb.ChatMessage, 100)
		go ReceiveMessages(stream, inbox)

		for {
			select {
			case toSend := <-sQueue:
				switch msg := strings.TrimSpace(toSend.Message); msg {
				case "!exit":
					stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: uName + " left chat!\n"})
					ExitChat(c, stream, uName, gName)
					stream.CloseSend()
					conn.Close()
				case "!members":
					DisplayCurrentMembers(c, gName)
				case "!help":
					AddSpacing(1)
					fmt.Println("The following commands are available to you: ")
					color.New(color.FgHiYellow).Print("   !members")
					fmt.Print(": Lists the current members in the group.")

					AddSpacing(1)
					color.New(color.FgHiYellow).Print("   !exit")
					fmt.Println(": Leaves the chat server.")
					AddSpacing(1)

				default:
					stream.Send(&toSend)
				}
			case received := <-inbox:
				fmt.Printf("%s> %s", received.Sender, received.Message)
			}
		}
	}
}
