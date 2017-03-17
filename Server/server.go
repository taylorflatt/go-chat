package main

import (
	"errors"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	pb "github.com/taylorflatt/go-chat"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":12021"
)

// Server is used to implement the RemoteCommandServer
type server struct{}

var clients = make(map[string]chan pb.ChatMessage, 100)
var groups = make(map[int32]chan pb.ChatMessage, 100)
var groupClients = make(map[int32][]string)

var lock = &sync.RWMutex{}

func RandInt32(min int32, max int32) int32 {
	rand.Seed(time.Now().Unix())
	return min + rand.Int31n(max-min)
}

func removeListener(name int32) {

	lock.Lock()
	defer lock.Unlock()
	delete(groups, name)
}

func hasListener(name string) bool {

	lock.RLock()
	defer lock.RUnlock()
	_, ok := clients[name]
	return ok
}

func clientExists(name string) bool {

	lock.RLock()
	defer lock.RUnlock()
	for c := range clients {
		if c == name {
			return true
		}
	}

	return false
}

func inGroup(name string) bool {

	lock.RLock()
	defer lock.RUnlock()
	for _, c := range groupClients {
		for _, s := range c {
			if name == s {
				return true
			}
		}
	}

	return false

}

func genGroupName() int32 {

	exists := false

	for {
		val := RandInt32(1, 1000)
		// Look through all the groups to make sure it is unique.
		for g, _ := range groups {
			if val == g {
				exists = true
				break
			}
		}

		// If it is unique, return the new group name.
		if !exists {
			return val
		}
	}
}

func addClientsToGroup(name int32, buddies []string) {

	lock.Lock()
	defer lock.Unlock()
	groupClients[name] = buddies
}

func addGroup(name int32) {

	lock.Lock()
	defer lock.Unlock()
	groups[name] = make(chan pb.ChatMessage, 100)
}

func addClient(name string) error {

	if !clientExists(name) {
		clients[name] = make(chan pb.ChatMessage, 100)
		return nil
	}

	return errors.New("client (" + name + ") already exists")

}

func (s *server) Register(ctx context.Context, in *pb.ClientInfo) (*pb.InviteResponse, error) {

	err := addClient(in.Sender)

	if err != nil {
		return &pb.InviteResponse{Response: false}, err
	}

	return &pb.InviteResponse{Response: true}, nil

}

func (s *server) GetClientList(ctx context.Context, in *pb.Empty) (*pb.ClientList, error) {

	var conClients []string
	for key := range clients {
		conClients = append(conClients, key)
	}

	return &pb.ClientList{Clients: conClients}, nil

}

func (s *server) EstablishConnection(ctx context.Context, in *pb.InviteRequest) (*pb.InviteResponse, error) {

	buddies := in.Clients
	numClients := len(buddies)

	// Check that all the clients are currently on the server and not already in groups.
	if numClients != 0 {
		for _, name := range buddies {
			if !clientExists(name) {
				return &pb.InviteResponse{Response: false}, errors.New("connection failed: the client (" + name + ") isn't registered on the server anymore")
			} else if inGroup(name) {
				return &pb.InviteResponse{Response: false}, errors.New("connection failed: the client (" + name + ") is already in another group")
			}
		}
	} else {
		return &pb.InviteResponse{Response: false}, errors.New("connection failed: there are no clients registered with the server")
	}

	// Now we need to create a group that contains all of the buddies with a single channel.
	name := genGroupName()
	addGroup(name)
	addClientsToGroup(name, buddies)

	return &pb.InviteResponse{Response: true}, nil
}

func (s *server) RouteChat(stream pb.Chat_RouteChatServer) error {

	msg, err := stream.Recv()

	if err != nil {
		return err
	}

	gChan := groups[msg.Receiver]
	go listenToClient(stream, gChan)

	for {
		select {
		case outbox := <-gChan:
			broadcast(msg.Sender, msg.Receiver, outbox)
		case inbox := <-gChan:
			stream.Send(&inbox)
		}
	}

}

func broadcast(sender string, gName int32, msg pb.ChatMessage) {

	lock.Lock()
	defer lock.Unlock()

	gChan := groups[gName]
	for _, buddy := range groupClients[gName] {
		if buddy != sender {
			gChan <- msg
		}
	}
}

func listenToClient(stream pb.Chat_RouteChatServer, messages chan<- pb.ChatMessage) {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
		}
		if err != nil {
		}
		messages <- *msg
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
