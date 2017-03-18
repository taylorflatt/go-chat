package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

func MainMenu() {
	addSpacing(1)
	fmt.Println("Welcome to Go-Chat!")
	addSpacing(1)
	fmt.Println("Below is a list of menu options for the chat application.")
	addSpacing(1)
	fmt.Println("1) Create a Group")
	fmt.Println("2) View Group List")
	fmt.Println("3) Exit Chat")
	addSpacing(1)
	fmt.Print("Menu> ")
}

func GroupMenu() {
	addSpacing(1)
	fmt.Println("View Groups Menu")
	addSpacing(1)
	fmt.Println("Below is a list of menu options for groups.")
	addSpacing(1)
	fmt.Println("1) Join a Group")
	fmt.Println("2) View Group Members")
	fmt.Println("3) Refresh Group List")
	fmt.Println("4) Go back")
	addSpacing(1)
	fmt.Print("Menu> ")
}

func ViewGroupMembersMenu() {
	addSpacing(1)
	fmt.Println("Enter the group name that you would like to view! Enter !back to go back to the menu.")
	addSpacing(1)
	fmt.Print("Menu> ")
}

func CheckError(err error) {
	if err != nil {
		fmt.Print(err)
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
	address := "localhost:12021"

	// UNCOMMENT AFTER DEBUG
	//fmt.Print("Please specify the server IP: ")
	//t, _ := r.ReadString('\n')
	//t = strings.TrimSpace(t)
	//ts := strings.Split(t, ":")
	//sip := ts[0]
	//sport := ts[1]
	//address := sip + ":" + sport
	// END UNCOMMENT

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
	for {
		fmt.Printf("Enter your username: ")
		tu, err := r.ReadString('\n')
		if err != nil {
			fmt.Print(err)
		}
		uName = strings.TrimSpace(tu)

		_, err = c.Register(context.Background(), &pb.ClientInfo{Sender: uName})

		if err == nil {
			fmt.Println("Your username: " + uName)

			w := make(chan os.Signal, 1)

			signal.Notify(w, syscall.SIGINT, syscall.SIGTERM)

			go func() {
				sig := <-w
				fmt.Print(sig)
				fmt.Print(" used.")
				fmt.Println("Exiting chat application.")

				if sig == os.Interrupt {
					c.UnRegister(context.Background(), &pb.ClientInfo{Sender: uName})
					os.Exit(1)
				}
			}()
			break
		} else {
			fmt.Print(err)
		}
	}

	rMainMenu := true
	for rMainMenu {
		MainMenu()
		tc, err := r.ReadString('\n')
		tc = strings.TrimSpace(tc)
		CheckError(err)

		choice, err := strconv.Atoi(tc)
		CheckError(err)

		if choice > 3 || choice < 1 {
			fmt.Println("Please enter a valid selection between 1 and 3.")
		} else if choice == 1 {
			// Create a group

			rCGroup := true
			for rCGroup {
				addSpacing(1)
				fmt.Println("Enter the name of the group or type !back to go back to the main menu.")
				fmt.Print("Menu> ")
				gName, err = r.ReadString('\n')
				gName = strings.TrimSpace(gName)
				CheckError(err)

				if gName == "!back" {
					rCGroup = false
				} else {
					_, nerr := c.CreateGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gName})

					if nerr != nil {
						fmt.Println("That group name has already been chosen. Please select a new one.")
					} else {
						fmt.Println("Created group named " + gName)
						c.JoinGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gName})
						fmt.Println("Joined " + gName)
						addSpacing(1)
						rCGroup = false
						rMainMenu = false
					}
				}
			}

		} else if choice == 2 {
			// View groups
			// List the current groups.
			tr, _ := c.GetGroupList(context.Background(), &pb.Empty{})
			res := tr.Groups

			if len(res) == 0 {
				fmt.Println("There are currently no groups created. Go back to the main menu to create one!")
				addSpacing(1)
			} else {
				fmt.Println("List of all groups: ")
				for i, gName := range res {
					fmt.Println("  " + strconv.Itoa(i+1) + ") " + gName)
				}
			}

			rGMenu := true
			for rGMenu {
				GroupMenu()
				tgc, err := r.ReadString('\n')
				tgc = strings.TrimSpace(tgc)
				CheckError(err)

				gChoice, err := strconv.Atoi(tgc)
				CheckError(err)

				if gChoice > 4 || gChoice < 1 {
					fmt.Println("Please enter a valid selection between 1 and 4.")
				} else if gChoice == 1 {
					// Join a group
					rGName := true
					for rGName {
						fmt.Println("Enter the name of the group as it appears in the group list or enter !back to go back to the Group menu.")
						fmt.Print("menu> ")
						gName, _ = r.ReadString('\n')
						gName = strings.TrimSpace(gName)
						CheckError(err)

						if gName == "!back" {
							rGName = false
						} else {
							_, err := c.JoinGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gName})
							fmt.Println("Joined " + gName)

							if err != nil {
								fmt.Print("A group with that name doesn't exist. Please check again.")
							} else {
								rGName = false    // Leave join menu
								rGMenu = false    // Leave Group menu
								rMainMenu = false // Leave main menu
							}
						}
					}

				} else if gChoice == 2 {
					//View Group members for a group
					ViewGroupMembersMenu()

					rGMemMenu := true
					for rGMemMenu {
						gCName, err := r.ReadString('\n')
						gCName = strings.TrimSpace(gCName)
						CheckError(err)

						if gCName == "!back" {
							rGMemMenu = false
						} else {
							cList, err := c.GetGroupClientList(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gCName})
							if err != nil {
								fmt.Println("Please enter the group name exactly as it appears in the group list!")
							} else {
								fmt.Println(cList)
								rGMemMenu = false
							}
						}
					}
				} else if gChoice == 3 {
					// Refresh the group list
					tr, _ := c.GetGroupList(context.Background(), &pb.Empty{})
					res := tr.Groups

					for _, gName := range res {
						fmt.Println("Group: " + gName)
					}

				} else {
					// Go back
					addSpacing(4)
					rGMenu = false
				}
			}
		} else {
			// Exit chat client
			c.UnRegister(context.Background(), &pb.ClientInfo{Sender: uName})
			os.Exit(0)
		}
	}

	fmt.Println("You are now chatting in " + gName)
	addSpacing(1)

	stream, serr := c.RouteChat(context.Background())

	if serr != nil {
		fmt.Print(serr)
	} else {
		mailBox := make(chan pb.ChatMessage, 100)
		go receiveMessages(stream, mailBox, gName)

		sQueue := make(chan pb.ChatMessage, 100)
		go listenToClient(sQueue, r, uName, gName)

		for {
			select {
			case toSend := <-sQueue:
				stream.Send(&toSend)
			case received := <-mailBox:
				fmt.Printf("%s> %s", received.Sender, received.Message)
			}
		}

		//stream.Send(&pb.ChatMessage{Sender: uName, Receiver: gName})
	}
}

func addSpacing(n int) {
	for i := 0; i <= n; i++ {
		fmt.Println()
	}
}

func listenToClient(sQueue chan pb.ChatMessage, reader *bufio.Reader, uName string, gName string) {
	for {
		//fmt.Print("You> ")
		msg, _ := reader.ReadString('\n')
		sQueue <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
	}
}

// Check here if the msg coming in is from itself (sender == uName)
func receiveMessages(stream pb.Chat_RouteChatClient, mailbox chan pb.ChatMessage, gName string) {
	for {
		msg, _ := stream.Recv()

		mailbox <- *msg

		//		if msg.Receiver == gName {
		//			mailbox <- *msg
		//		}
	}
}
