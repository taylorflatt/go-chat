package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
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

func SingleChat(c pb.ChatClient, r *bufio.Reader) {

	fmt.Printf("Connectable clients: \n")
	var ips []string
	var ports []int32

	res, err := c.GetClientList(context.Background(), &pb.List{})

	if err != nil {
		log.Fatalf("Failed to get list of clients: %v", err)
	} else {
		if res.Ip != nil {
			ips = res.Ip
			ports = res.Port

			// TODO: Remove the current client's IP from the list of clients connected.
			for i := range ips {
				// TODO: Rewrite to a custom function for quicker conversion.
				fmt.Println(strconv.Itoa(i+1) + ") " + ips[i] + ":" + strconv.Itoa(int(ports[i])))
			}
		} else {
			fmt.Println("There are currently no clients.")
		}
	}

	fmt.Printf("\nPlease enter the name of the client to whom you wish to connect as it appears above: ")
	client, _ := r.ReadString('\n')
	t := strings.TrimSpace(client)
	ts := strings.Split(t, ":")
	cIP := ts[0]

	// Type cast from string to int to int32.
	// TODO: Clean this up.
	tc, _ := strconv.Atoi(ts[1])
	var cPort int32
	cPort = int32(tc)

	stream, err := c.RouteChat(context.Background())
	waitc := make(chan struct{})

	fmt.Printf("\n\nPlease enter '!exit' to exit the chat.\n")
	for {
		// Close the connection if the user enters exit.
		fmt.Printf("You: ")
		m, _ := r.ReadString('\n')
		m = strings.TrimSpace(m)
		if m == "!exit" {
			c.UnRegisterClient(context.Background(), &pb.ClientInfo{Ip: cIP, Port: cPort})
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
	t, _ := r.ReadString('\n')
	t = strings.TrimSpace(t)
	ts := strings.Split(t, ":")
	sip := ts[0]
	sport := ts[1]

	address := sip + ":" + sport

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
	var dupe = true
	for dupe == true {
		// Generate unique port.
		port := RandInt32(10000, 15000)
		_, err := c.RegisterClient(context.Background(), &pb.ClientInfo{Ip: ip, Port: port})

		if err == nil {
			dupe = false
			fmt.Println("Your port: " + strconv.Itoa(int(port)))
		}
	}

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
