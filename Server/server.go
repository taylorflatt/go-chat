package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	pb "github.com/taylorflatt/go-chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":12021"
)

// Server is used to implement the RemoteCommandServer
type server struct{}

var clients = make(map[string]chan pb.ChatMessage, 100)
var groups = make(map[int]chan pb.ChatMessage, 100)
var groupClients = make(map[int][]string)

var lock = &sync.RWMutex{}

func RandInt(min int, max int) int {
	rand.Seed(time.Now().Unix())
	return min + rand.Intn(max-min)
}

func addListener(name string, queue chan pb.ChatMessage) {

	lock.Lock()
	defer lock.Unlock()
	groups[name] = queue
}

func removeListener(name string) {

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

func broadcast(client string, msg pb.ChatMessage) {

	lock.Lock()
	defer lock.Unlock()
	for sender, queue := range clients {
		if sender != client {
			queue <- msg
		}
	}
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
	for n, c := range groupClients {
		for _, s := range c {
			if name == s {
				return true
			}
		}
	}

	return false

}

func genGroupName() int {

	exists := false

	for {
		val := RandInt(1, 1000)
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

func (s *server) EstablishConnection(ctx context.Context, in *pb.InviteRequest) (*pb.InviteResponse, error) {

	requester := in.Requester
	buddies := in.Clients
	numClients := len(buddies)

	// Check that all the clients are currently on the server and not already in groups.
	for _, name := range buddies {
		if !clientExists(name) {
			return &pb.InviteResponse{Response: false}, errors.New("connection failed: the client (" + name + ") isn't registered on the server anymore")
		} else if inGroup(name) {
			return &pb.InviteResponse{Response: false}, errors.New("connection failed: the client (" + name + ") is already in another group")
		}
	}

	// Now we need to create a group that contains all of the buddies with a single channel.
	name := genGroupName()
	groups[name] = make(chan pb.ChatMessage, 100)
	groupClients[name] = buddies

	return &pb.InviteResponse{Response: true}, nil
}

func (s *server) RouteChat(stream pb.Chat_RouteChatServer) error {

	msg, err := stream.Recv()

	if err != nil {
		return err
	}

	in := make(chan pb.ChatMessage, 100)
	var client string

	// Register the client with the server.
	if msg.Register {
		client = msg.Sender

		if hasListener(client) {
			return fmt.Errorf("this client already exists")
		}

		addListener(client, in)
	} else {
		return fmt.Errorf("you need to register prior to sending messages")
	}

	// Send/Receive messages.
	out := make(chan pb.Message, 100)
	go listenToClient(stream, out)

	for {
		select {
		case outbox := <-out:
			broadcast(client, outbox)
		case inbox := <-in:
			stream.Send(&inbox)
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
