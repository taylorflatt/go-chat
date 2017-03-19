package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// The IP is hardcoded now. But eventually it will not be.
// Will need to reference Interfaces().
const (
	ip = "localhost"
	//port = 12021
)

// RandInt32 generates a random int32 between two values.
func RandInt32(min int32, max int32) int32 {
	rand.Seed(time.Now().Unix())
	return min + rand.Int31n(max-min)
}

func CheckError(err error) {
	if err != nil {
		fmt.Print(err)
	}
}

func ExitChat(c pb.ChatClient, stream pb.Chat_RouteChatClient, u string, g string) {

	c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
	os.Exit(1)
}

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

func ControlExitLate(w chan os.Signal, c pb.ChatClient, stream pb.Chat_RouteChatClient, u string, g string) {

	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)
	sig := <-w

	if sig == os.Interrupt {
		ExitChat(c, stream, u, g)
	}
}

func main() {

	// Read in the user's command.
	r := bufio.NewReader(os.Stdin)

	// username, groupname
	var uName string
	var gName string

	// Read the server address
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
	} else {
		fmt.Print("Group: " + gName)
	}

	AddSpacing(1)
	fmt.Println("You are now chatting in " + gName + ".")
	Frame()

	stream, serr := c.RouteChat(context.Background())
	q <- true
	go ControlExitLate(w, c, stream, uName, gName)

	// First message always gets dropped. TODO: Figure out why.
	// Need second message to establish itself with the server and do an announce within the group.
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: ""})
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: " joined chat!\n"})

	if serr != nil {
		fmt.Print(serr)
	} else {

		sQueue := make(chan pb.ChatMessage, 100)
		go listenToClient(sQueue, r, uName, gName)

		inbox := make(chan pb.ChatMessage, 100)
		go receiveMessages(stream, inbox)

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
					l, err := c.GetGroupClientList(context.Background(), &pb.GroupInfo{GroupName: gName})
					if err != nil {
						fmt.Println("There was an error grabbing the current members of the group: " + err.Error())
					} else {
						fmt.Println("The current members in the group are: ")
						fmt.Println(l.Clients)
					}
				case "!help":
					AddSpacing(1)
					fmt.Println("The following commands are available to you: ")
					color.New(color.FgHiYellow).Print("   !members")
					fmt.Print(": Lists the current members in the group.")

					fmt.Println()
					color.New(color.FgHiYellow).Print("   !exit")
					fmt.Println(": Leaves the chat server.")
					fmt.Println()

				default:
					stream.Send(&toSend)
				}

				//stream.Send(&toSend)
			case received := <-inbox:
				fmt.Printf("%s> %s", received.Sender, received.Message)
			}
		}
	}
}

func listenToClient(sQueue chan pb.ChatMessage, reader *bufio.Reader, uName string, gName string) {
	for {
		msg, _ := reader.ReadString('\n')
		sQueue <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
	}
}

func receiveMessages(stream pb.Chat_RouteChatClient, inbox chan pb.ChatMessage) {
	for {
		msg, _ := stream.Recv()
		inbox <- *msg
	}
}
