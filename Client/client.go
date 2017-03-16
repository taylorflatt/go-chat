package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// The IP is hardcoded now. But eventually it will not be.
// Will need to reference Interfaces().
const (
	ip   = "localhost"
	port = 12021
)

func SingleChat(c pb.ChatClient, r *bufio.Reader) {
	fmt.Printf("Connectable clients: \n")
	res, err := c.GetClientList(context.Background(), &pb.List{})
	if err != nil {
		log.Fatalf("Failed to get list of clients: %v", err)
	} else {
		// TODO: Split up into individual items so they can be formatted nicely.
		fmt.Println(res)
	}

	fmt.Printf("\nPlease enter the name of the client to whom you wish to connect as it appears above: ")
	client, _ := r.ReadString('\n')

	stream, err := c.RouteChat(context.Background())
	waitc := make(chan struct{})

	fmt.Printf("\n\nPlease enter '\\exit' to exit the chat.\n")
	for {
		// Close the connection if the user enters exit.
		fmt.Printf("You: ")
		m, _ := r.ReadString('\n')
		if m == "\\exit" {
			c.UnRegisterClient(context.Background(), &pb.ClientInfo{Ip: ip, Port: port})
			break
		} else {
			msg := &pb.RouteMessage{Ip: client, Message: m}
			stream.Send(msg)
		}

		if err != nil {
			log.Fatalf("Command failed: %v", err)
		}
	}
	<-waitc
	stream.CloseSend()
}

func main() {
	// Read in the user's command.
	r := bufio.NewReader(os.Stdin)

	// Read the server address
	fmt.Print("Please specify the server IP: ")
	address, _ := r.ReadString('\n')
	address = strings.TrimSpace(address)
	address = address + ":" + strconv.Itoa(port)

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := pb.NewChatClient(conn)

	// Register the client with the server.
	c.RegisterClient(context.Background(), &pb.ClientInfo{Ip: ip, Port: port})

	fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type exit.\n", address)

	for true {
		fmt.Printf("\nWould you like to group chat (yes/no): ")
		g, _ := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if g == "yes" {
			// Do group chat.
		} else if g == "no" {
			// Connect to a single client.
			SingleChat(c, r)

		} else {
			fmt.Printf("Please enter either 'yes' for group chat or 'no' for single chat.\n\n")
		}
	}

	fmt.Print("\nPlease type the name of the client as it appears above if you wish to connect to it.")
	fmt.Print("\n$ ")
}
