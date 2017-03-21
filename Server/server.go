package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	pb "github.com/taylorflatt/go-chat"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// The port the server is listening on.
const (
	port = ":12021"
)

// Server is used to implement the RemoteCommandServer
type server struct{}

// Clients: A list of unique clients and a channel per client.
// Groups: A list of unique groups and a channel per group.
// GroupClients: A list of Groups and a list of all clients in that group.
// Note: Messages within a group are sent to that group's channel and then routed
//       to each client channel who is part of that group EXCEPT the sender's channel.
var clients = make(map[string]chan pb.ChatMessage, 100)
var groups = make(map[string]chan pb.ChatMessage, 100)
var groupClients = make(map[string][]string)

var lock = &sync.RWMutex{}

// ClientExists checks if a client exists on the server.
// It returns a bool value.
func ClientExists(name string) bool {

	for c := range clients {
		if c == name {
			return true
		}
	}

	return false
}

// GroupExists checks if a group exists on the server.
// It returns a bool value.
func GroupExists(gName string) bool {

	// Changed to just LOCK
	lock.Lock()
	defer lock.Unlock()
	for g := range groups {
		if g == gName {
			return true
		}
	}

	return false
}

// InGroup checks whether a client is currently in a
// specific group.
// It returns a bool value.
func InGroup(name string) bool {

	for _, c := range groupClients {
		for _, s := range c {
			if name == s {
				return true
			}
		}
	}

	return false

}

// AddClient adds a new client to the server.
// It doesn't return anything.
func AddClient(name string) {

	clients[name] = make(chan pb.ChatMessage, 100)
	log.Print("[AddClient]: Registered client " + name)
	log.Print("[AddClient]: Client's channel: ")
	log.Print(clients[name])
}

// AddGroup adds a new group to the server.
// It doesn't return anything.
func AddGroup(gName string) {

	lock.Lock()
	defer lock.Unlock()
	groups[gName] = make(chan pb.ChatMessage, 100)
	log.Print("[AddGroup]: Added group " + gName)
}

// RemoveClient will remove a client from the server as well as any
// groups that they are currently in.
// It returns an error.
func RemoveClient(name string) error {

	lock.Lock()
	defer lock.Unlock()

	if ClientExists(name) {
		delete(clients, name)
		log.Print("[RemoveClient]: Removed client " + name)
		if InGroup(name) {
			RemoveClientFromGroup(name)
		} else {
			log.Print("[RemoveClient]: " + name + " was not in any groups.")
			return nil
		}
	}

	return errors.New("[RemoveClient]: Client (" + name + ") doesn't exist")
}

// AddClientToGroup will add a client to a group.
// It doesn't return anything.
func AddClientToGroup(cName string, gName string) {

	lock.Lock()
	defer lock.Unlock()
	cList := groupClients[gName]
	cList = append(cList, cName)
	groupClients[gName] = cList

	log.Println("[AddClientToGroup] Added " + cName + " to " + gName)
}

// RemoveClientFromGroup will remove a client from a specific group. It will also
// delete a group if the client is the last one leaving it.
// It returns an error.
func RemoveClientFromGroup(name string) error {

	// Look through all the groups.
	for gName, cList := range groupClients {
		listG := cList
		// Look through all the users in the group.
		for i, cName := range listG {
			// Remove the user from the group.
			if cName == name {
				log.Println("[RemoveClientFromGroup]: Removed client " + name + " from " + gName)
				if len(listG) == 1 {
					delete(groups, gName)
					delete(groupClients, gName)
					log.Println("[RemoveClientFromGroup]: No more members in " + gName + ", removing the group.")
					log.Print("List of groups: ")
					for keys := range groups {
						log.Print(keys)
					}
				} else {
					listG[i] = listG[len(listG)-1]
					listG = listG[:len(listG)-1]
					groupClients[gName] = listG
					log.Print(listG)
				}

				return nil
			}
		}
	}

	return errors.New("no user found in the group list. Something went wrong")
}

// GetClientList will get all of the currently connected clients to the server.
// It returns a list of connected clients.
func (s *server) GetClientList(ctx context.Context, in *pb.Empty) (*pb.ClientList, error) {

	var conClients []string
	for key := range clients {
		conClients = append(conClients, key)
	}

	return &pb.ClientList{Clients: conClients}, nil
}

// GetGroupList will get all of the groups currently registered on the server.
// It returns a list of groups.
func (s *server) GetGroupList(ctx context.Context, in *pb.Empty) (*pb.GroupList, error) {

	var g []string
	for gName := range groups {
		g = append(g, gName)
	}

	log.Print("[GetGroupList]: Returned list of current groups ")
	log.Print(g)

	return &pb.GroupList{Groups: g}, nil
}

// GetGroupClientList will get all of the clients who is current part of a specific group.
// It returns a list of clients belonging to a group.
func (s *server) GetGroupClientList(ctx context.Context, in *pb.GroupInfo) (*pb.ClientList, error) {

	gName := in.GroupName

	if !GroupExists(gName) {
		return &pb.ClientList{}, errors.New("that group doesn't exist")
	}

	cList := groupClients[gName]

	log.Print("[GetGroupClientList]: For group " + gName + " returned members ")
	log.Print(cList)

	return &pb.ClientList{Clients: cList}, nil
}

// Register will add the user to the server's collection of users (and by extension restrict the username).
// It returns an empty object and an error.
func (s *server) Register(ctx context.Context, in *pb.ClientInfo) (*pb.Empty, error) {

	n := in.Sender
	if ClientExists(n) {
		return nil, errors.New("that name already exists")
	}

	AddClient(n)
	return &pb.Empty{}, nil
}

// UnRegister removes a user from the server's collection of users and any
// groups the user may have been in.
// It returns an empty object and an error.
func (s *server) UnRegister(ctx context.Context, in *pb.ClientInfo) (*pb.Empty, error) {

	uName := in.Sender

	cChan := clients[uName]
	fmt.Print(cChan)

	log.Print("[UnRegister]: Unregistering client " + uName)

	err := RemoveClient(uName)

	log.Println("[UnRegister]: The following are the remaining clients, ")
	keys := []string{}
	for key := range clients {
		keys = append(keys, key)
	}
	log.Println(keys)

	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// CreateGroup creates a new group provided it doesn't already exist.
// It returns an empty object and an error.
func (s *server) CreateGroup(ctx context.Context, in *pb.GroupInfo) (*pb.Empty, error) {

	cName := in.Client
	gName := in.GroupName

	log.Printf("[CreateGroup] " + cName + " is attempting to create " + gName)

	if !GroupExists(gName) {
		AddGroup(gName)
		return &pb.Empty{}, nil
	}

	return &pb.Empty{}, errors.New("a group with that name already exists")
}

// JoinGroup adds a user to an existing group.
// It returns an empty object and an error.
func (s *server) JoinGroup(ctx context.Context, in *pb.GroupInfo) (*pb.Empty, error) {

	cName := in.Client
	gName := in.GroupName

	log.Printf("[JoinGroup] Attempting to add " + cName + " to " + gName)

	if GroupExists(gName) {
		AddClientToGroup(cName, gName)

		return &pb.Empty{}, nil
	}

	return &pb.Empty{}, errors.New("a group with that name doesn't exist")
}

// LeaveRoom removes the user from their group.
// It returns an empty object and an error.
func (s *server) LeaveRoom(ctx context.Context, in *pb.GroupInfo) (*pb.Empty, error) {

	u := in.Client
	g := in.GroupName

	if !GroupExists(g) {
		return &pb.Empty{}, errors.New("the group " + g + " doesn't exist")
	} else if !ClientExists(u) {
		return &pb.Empty{}, errors.New("the client " + g + " doesn't exist")
	} else {
		die := pb.ChatMessage{Sender: u, Receiver: g, Message: u + " left chat!\n"}
		Broadcast(g, die)
		RemoveClientFromGroup(u)
		return &pb.Empty{}, nil
	}
}

// RouteChat handles the routing of all messages on the stream.
// It returns an error.
func (s *server) RouteChat(stream pb.Chat_RouteChatServer) error {

	msg, err := stream.Recv()

	if err != nil {
		return err
	}

	log.Printf("[RouteChat]: Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)

	outbox := make(chan pb.ChatMessage, 100)

	go ListenToClient(stream, outbox)

	for {
		select {
		case outMsg := <-outbox:
			Broadcast(msg.Receiver, outMsg)
		case inMsg := <-clients[msg.Sender]:
			log.Println("Sending message to channel: ")
			log.Println(clients[msg.Sender])
			log.Println("[LOOK HERE]: Sending message to STREAM: ")
			log.Println(stream)
			stream.Send(&inMsg)
		}
	}
}

// Broadcast takes any messages that need to be sent and sorts them by group. It then
// adds the message to the channel of each member of that group.
// It doesn't return anything.
func Broadcast(gName string, msg pb.ChatMessage) {

	lock.Lock()
	defer lock.Unlock()

	for gn := range groups {
		log.Printf("[Broadcast]: I found " + gn + ".")
		if gn == gName {
			log.Printf("[Broadcast]: Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)
			for _, cName := range groupClients[gn] {
				log.Printf("[Broadcast]: I found " + cName + " in gName")
				if cName == msg.Sender && msg.Message == msg.Sender+" left chat!\n" {
					log.Printf("[Broadcast]: ADDING THE KILL MESSAGE TO " + cName)
					clients[cName] <- msg
				} else if cName != msg.Sender {
					log.Printf("[Broadcast] Adding the message to " + cName + "'s channel.")
					clients[cName] <- msg
				}
			}
		}
	}
}

// ListenToClient listens on the incoming stream for any messages. It adds those messages to the channel.
// It doesn't return anything.
func ListenToClient(stream pb.Chat_RouteChatServer, messages chan<- pb.ChatMessage) {

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
		}
		if err != nil {
		} else {
			log.Printf("[ListenToClient] Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)
			messages <- *msg
		}

	}
}

func main() {

	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Failed to listen %v", err)
	}

	// Initializes the gRPC server.
	s := grpc.NewServer()

	// Register the server with gRPC.
	pb.RegisterChatServer(s, &server{})

	// Register reflection service on gRPC server.
	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
