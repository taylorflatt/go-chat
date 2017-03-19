package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

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

func ExitChat(c pb.ChatClient, uName string) {
	c.UnRegister(context.Background(), &pb.ClientInfo{Sender: uName})
	os.Exit(1)
}

func ControlExitEarly(w chan os.Signal, c pb.ChatClient, uName string) {

	signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)

	sig := <-w
	fmt.Print(sig)
	fmt.Println(" used.")
	fmt.Println("Exiting chat application.")

	if sig == os.Interrupt {
		ExitChat(c, uName)
	}
}

func ControlExitLate(w chan os.Signal, c pb.ChatClient, uName string) {

	ExitChat(c, uName)
}

func main() {

	// Read in the user's command.
	r := bufio.NewReader(os.Stdin)

	// username, groupname
	var uName string
	var gName string

	// Read the server address
	// DEBUG ONLY:
	//a := "localhost:12021"
	// UNCOMMENT AFTER DEBUG
	a := SetServer(r)
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

	info, err := TopMenu(c, r)

	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	} else {
		uName = info[0]
		gName = info[1]
	}

	AddSpacing(1)
	fmt.Println("You are now chatting in " + gName)

	stream, serr := c.RouteChat(context.Background())

	// First message always gets dropped. TODO: Figure out why.
	// Need second message to establish itself with the server and do an announce within the group.
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: ""})
	stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName, Message: uName + " joined chat!\n"})

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
				stream.Send(&toSend)
			case received := <-inbox:
				fmt.Printf("%s> %s", received.Sender, received.Message)
			}
		}

		//stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName})
	}
}

func listenToClient(sQueue chan pb.ChatMessage, reader *bufio.Reader, uName string, gName string) {
	for {
		msg, _ := reader.ReadString('\n')
		sQueue <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
	}
}

// Check here if the msg coming in is from itself (sender == uName)
func receiveMessages(stream pb.Chat_RouteChatClient, inbox chan pb.ChatMessage) {
	for {
		msg, _ := stream.Recv()
		inbox <- *msg
	}
}
