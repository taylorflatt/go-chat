package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":12021"
)

func main() {
	// Read in the user's command.
	r := bufio.NewReader(os.Stdin)

	// Read the server address
	fmt.Print("Please specify the server IP: ")
	address, _ := r.ReadString('\n')
	address = strings.TrimSpace(address)
	address = address + port

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}

	// Close the connection after main returns.
	defer conn.Close()

	// Create the client
	c := pb.NewRemoteCommandClient(conn)

	fmt.Printf("\nYou have successfully connected to %s! To disconnect, hit ctrl+c or type exit.\n", address)

	for true {
		fmt.Printf("\nWould you like to group chat (yes/no)?")
		g, _ := r.ReadString('\n')

		if g == "yes" {
			// Do group chat.
		} else if g == "no" {
			// Connect to a client.
			fmt.Printf("Connectable clients: ")
			// GET connectable clients from the server.
			// ENDGET

			fmt.Printf("\nPlease enter the name of the client to whom you wish to connect as it appears above: ")
			c, _ := r.ReadString('\n')

			for true {

				// This strips off any trailing whitespace/carriage returns.
				tCmd = strings.TrimSpace(tCmd)
				tCmd2 := strings.Split(tCmd, " ")

				// Parse their input.
				cmdName := tCmd2[0]

				//cmdArgs := []string{}
				cmdArgs := tCmd2[1:]

				// Close the connection if the user enters exit.
				if cmdName == "exit" {
					break
				}

				// Gets the response of the shell comm and from the server.
				res, err := c.SendCommand(context.Background(), &pb.CommandRequest{CmdName: cmdName, CmdArgs: cmdArgs})

				if err != nil {
					log.Fatalf("Command failed: %v", err)
				}

				log.Printf("    %s", res.Output)
			}
		} else {
			fmt.Printf("Please enter either 'yes' for group chat or 'no' for single chat.\n\n")
		}
	}

	fmt.Print("\nPlease type the name of the client as it appears above if you wish to connect to it.")
	fmt.Print("\n$ ")

	fmt.Print("$ ")
	tCmd, _ := r.ReadString('\n')

	// Keep connection alive until ctrl+c or exit is entered.

}
