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
	} else {
		fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type !exit.\n", address)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := pb.NewChatClient(conn)

	// Register the client with the server.
	var dupe = true
	for dupe == true {

		fmt.Printf("Enter your username: ")
		tu, err := r.ReadString('\n')
		if err != nil {
			fmt.Print(err)
		}
		uName := strings.TrimSpace(tu)

		_, err = c.Register(context.Background(), &pb.ClientInfo{Sender: uName})

		if err == nil {
			dupe = false
			fmt.Println("Your username: " + uName)
		} else {
			fmt.Print(err)
		}
	}

	fmt.Printf("Connectable clients: \n")

	var conClients []string
	res, err := c.GetClientList(context.Background(), &pb.Empty{})
	conClients = res.Clients

	if len(conClients) == 0 {
		fmt.Println("There are currently no clients.")
	} else {
		for i, name := range conClients {
			fmt.Println("  " + strconv.Itoa(i+1) + ") " + name)
		}
	}

	fmt.Println()
}
