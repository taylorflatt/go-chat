package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Watcher struct {
	ch        chan pb.ChatMessage
	WaitGroup *sync.WaitGroup
}

type Monitor struct {
	chatting  bool
	ch        chan os.Signal
	stream    pb.Chat_RouteChatClient
	WaitGroup *sync.WaitGroup
}

func CreateMonitor() *Monitor {
	m := &Monitor{
		chatting:  false,
		ch:        make(chan os.Signal),
		stream:    nil,
		WaitGroup: &sync.WaitGroup{},
	}

	m.WaitGroup.Add(1)
	return m
}

func CreateWatcher() *Watcher {
	s := &Watcher{
		ch:        make(chan pb.ChatMessage),
		WaitGroup: &sync.WaitGroup{},
	}
	log.Print("[CreateWatcher]: Set channel to ")
	log.Print(s.ch)
	log.Print("[CreateWatcher]: Set waitgroup to ")
	log.Print(s.WaitGroup)
	s.WaitGroup.Add(1)
	log.Print(s.WaitGroup)
	return s
}

func (s *Watcher) Stop() {
	log.Print("[Stop]: Entered Stop.")
	log.Print(s.ch)
	close(s.ch)
	log.Print("[Stop]: Waiting.")
	s.WaitGroup.Wait()
	log.Print("[Stop]: Done.")
}

// ControlExit handles any interrupts during program execution.
// Note: The routine control is dictated by the existence of a stream. If one is present, the user is in a group and needs
// to be removed. Otherwise, the user is still in the menu system.
// It doesn't return anything.
func (m *Monitor) ControlExit(c pb.ChatClient, u string, g string) {

	log.Print("[ControlExit]: Entered.")

	defer m.WaitGroup.Done()
	signal.Notify(m.ch, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-m.ch:
			log.Print("[ControlExit]: I need to quit the application!")
			if m.chatting {
				log.Print("[ControlExit]: I am chatting.")
				m.stream.Send(&pb.ChatMessage{Sender: u, Receiver: g, Message: u + " left chat!\n"})
				ExitClient(c, u, g)
				return
			}

			log.Print("[ControlExit]: I am NOT chatting.")
			ExitClient(c, u, g)
			os.Exit(1)
			return
		}
	}
}

// ExitClient handles removing the client from the server and exiting the program.
// It doesn't return anything.
func ExitClient(c pb.ChatClient, u string, g string) {

	c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
	os.Exit(1)
}

// ListenToClient listens to the client for input and adds that input to the sQueue with
// the username of the sender, group name, and the message.
// It doesn't return anything.
func ListenToClient(sQueue *Watcher, reader *bufio.Reader, uName string, gName string) {

	log.Println("[ListenToClient]: Starting.")
	defer sQueue.WaitGroup.Done()

	for {
		msg, _ := reader.ReadString('\n')
		if strings.TrimSpace(msg) == "!leave" {
			log.Println("[ListenToClient]: Stopping.")
			sQueue.ch <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
			return
		}
		log.Println("[ListenToClient]: Adding message to send queue.")
		sQueue.ch <- pb.ChatMessage{Sender: uName, Message: msg, Receiver: gName}
	}
}

// ReceiveMessages listens on the client's (NOT the client's group) stream and adds any incoming
// message to the client's inbox.
// It doesn't return anything.
func ReceiveMessages(inbox *Watcher, stream pb.Chat_RouteChatClient, u string) {

	log.Println("[ReceiveMessages]: Starting.")
	defer inbox.WaitGroup.Done()

	for {
		log.Println("[ReceiveMessages]: Listening for incoming messages.")
		msg, _ := stream.Recv()
		log.Println("[ReceiveMessages]: I see " + msg.Message)
		if msg.Message == u+" left chat!\n" {
			log.Println("[ReceiveMessages]: Found special signal!")
			return
		}

		inbox.ch <- *msg
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

	a := SetServer(r)

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
	ctx := context.Background()
	stream, serr := c.RouteChat(ctx)
	if serr != nil {
		log.Fatal(serr)
	}

	uName = SetName(c, r)
	showMenu := true // Control whether the user sees the menu or exits.
	m := CreateMonitor()
	go m.ControlExit(c, uName, gName)

	for showMenu {
		gName, err = TopMenu(c, r, uName)

		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}

		log.Print("Showing context")
		log.Print(ctx)
		showMenu = Chat(conn, stream, c, m, r, uName, gName)
	}
}

func Chat(conn *grpc.ClientConn, stream pb.Chat_RouteChatClient, c pb.ChatClient, m *Monitor, r *bufio.Reader, u string, g string) bool {

	//t, cancel := context.WithCancel(ctx)
	//stream, serr := c.RouteChat(t)
	//defer cancel()

	DisplayCurrentMembers(c, g)

	//if serr != nil {
	//	fmt.Print(serr)
	//} else {
	sQueue := CreateWatcher() // Creates the sQueue with a channel and waitgroup.
	inbox := CreateWatcher()  // Similar to sQueue

	go ListenToClient(sQueue, r, u, g)
	go ReceiveMessages(inbox, stream, u)

	// TODO: Find out why the first message is always dropped so an empty message needn't be sent.
	stream.Send(&pb.ChatMessage{Sender: u, Receiver: g, Message: ""})
	stream.Send(&pb.ChatMessage{Sender: u, Receiver: g, Message: "joined chat!\n"})
	m.chatting = true
	m.stream = stream

	AddSpacing(1)
	fmt.Println("You are now chatting in " + g + ".")
	Frame()

	for {
		select {
		case toSend := <-sQueue.ch:
			switch msg := strings.TrimSpace(toSend.Message); msg {
			case "!members":
				log.Println("[Main]: I'm in !members.")
				DisplayCurrentMembers(c, g)
			case "!leave":
				log.Println("[Main]: I'm in !leave.")
				c.LeaveRoom(context.Background(), &pb.GroupInfo{Client: u, GroupName: g})
				sQueue.Stop()
				inbox.Stop()
				//stream.CloseSend()
				log.Println("HEY LOOK")
				//cancel()
				log.Println(context.Canceled)
				return true
			case "!exit":
				log.Println("[Main]: I'm in !exit.")
				stream.Send(&pb.ChatMessage{Sender: u, Receiver: g, Message: u + " left chat!\n"})
				ExitClient(c, u, g)
				//stream.CloseSend()
				//cancel()
				conn.Close()
				return false
			case "!help":
				log.Println("[Main]: I'm in !help.")
				AddSpacing(1)
				fmt.Println("The following commands are available to you: ")
				color.New(color.FgHiYellow).Print("   !members")
				fmt.Print(": Lists the current members in the group.")

				AddSpacing(1)
				color.New(color.FgHiYellow).Print("   !exit")
				fmt.Println(": Leaves the chat server.")
				AddSpacing(1)

			default:
				log.Println("[Main]: Sending the message.")
				stream.Send(&toSend)
			}
		case received := <-inbox.ch:
			log.Println("[Main]: Receiving the message.")
			if received.Message != "!leave" {
				fmt.Printf("%s> %s", received.Sender, received.Message)
			}
		}
	}
	//}

	//return false
}
