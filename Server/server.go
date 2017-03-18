package main

import (
	"errors"
	"io"
	"log"
	"net"
	"sync"

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
var groups = make(map[string]chan pb.ChatMessage, 100)
var groupClients = make(map[string][]string)

var lock = &sync.RWMutex{}

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

func addClientToGroup(cName string, gName string) {

	lock.Lock()
	defer lock.Unlock()
	cList := groupClients[gName]
	cList = append(cList, cName)
	groupClients[gName] = cList

	log.Println("[addClientToGroup] Added " + cName + " to " + gName)
}

func addGroup(gName string) {

	lock.Lock()
	defer lock.Unlock()
	groups[gName] = make(chan pb.ChatMessage, 100)
	log.Print("[addGroup]: Added group " + gName)
}

func addClient(name string) error {

	if !clientExists(name) {
		clients[name] = make(chan pb.ChatMessage, 100)
		log.Print("[addClient]: Registered client " + name)
		return nil
	}

	return errors.New("client (" + name + ") already exists")
}

func groupExists(gName string) bool {

	lock.RLock()
	defer lock.RUnlock()
	for g := range groups {
		if g == gName {
			return true
		}
	}

	return false
}

func removeClient(name string) error {

	lock.Lock()
	defer lock.Unlock()

	if clientExists(name) {
		delete(clients, name)
		log.Print("[removeClient]: Removed client " + name)

		if inGroup(name) {
			// Look through all the groups.
			for gName := range groupClients {
				listG := groupClients[gName]
				// Look through all the users in the group.
				for i, cName := range listG {
					// Remove the user from the group.
					if cName == name {
						listG[i] = listG[len(listG)-1]
						listG = listG[:len(listG)-1]
						groupClients[name] = listG
						log.Print("[removeClient]: Removed client " + name + " from group " + gName)

						break
					}
				}
			}
		}
	}

	return errors.New("[removeClient]: Client (" + name + ") doesn't exist")
}

func (s *server) UnRegister(ctx context.Context, in *pb.ClientInfo) (*pb.Empty, error) {

	uName := in.Sender
	err := removeClient(uName)

	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil

}

func (s *server) GetGroupClientList(ctx context.Context, in *pb.GroupInfo) (*pb.ClientList, error) {

	gName := in.GroupName

	if !groupExists(gName) {
		return &pb.ClientList{}, errors.New("that group doesn't exist")
	}

	cList := groupClients[gName]

	log.Print("[GetGroupClientList]: For group " + gName + " returned members ")
	log.Print(cList)

	return &pb.ClientList{Clients: cList}, nil
}

func (s *server) GetGroupList(ctx context.Context, in *pb.Empty) (*pb.GroupList, error) {

	var g []string
	for gName := range groups {
		g = append(g, gName)
	}

	log.Print("[GetGroupList]: Returned list of current groups ")
	log.Print(g)

	return &pb.GroupList{Groups: g}, nil
}

func (s *server) JoinGroup(ctx context.Context, in *pb.GroupInfo) (*pb.Empty, error) {

	cName := in.Client
	gName := in.GroupName

	log.Printf("[JoinGroup] Attempting to add " + cName + " to group" + gName)

	if groupExists(gName) {
		addClientToGroup(cName, gName)

		return &pb.Empty{}, nil
	}

	return &pb.Empty{}, errors.New("a group with that name doesn't exist")
}

func (s *server) CreateGroup(ctx context.Context, in *pb.GroupInfo) (*pb.Empty, error) {

	cName := in.Client
	gName := in.GroupName

	if !groupExists(gName) {
		addGroup(gName)
		addClientToGroup(cName, gName)

		return &pb.Empty{}, nil
	}

	return &pb.Empty{}, errors.New("a group with that name already exists")
}

func (s *server) Register(ctx context.Context, in *pb.ClientInfo) (*pb.Empty, error) {

	err := addClient(in.Sender)

	if err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil

}

func (s *server) GetClientList(ctx context.Context, in *pb.Empty) (*pb.ClientList, error) {

	var conClients []string
	for key := range clients {
		conClients = append(conClients, key)
	}

	return &pb.ClientList{Clients: conClients}, nil

}

func (s *server) RouteChat(stream pb.Chat_RouteChatServer) error {

	msg, err := stream.Recv()

	if err != nil {
		return err
	}

	log.Printf("[RouteChat]: Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)

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

func broadcast(guy string, gName string, msg pb.ChatMessage) {

	lock.Lock()
	defer lock.Unlock()

	for gn, gChan := range groups {
		if gn == gName {
			gChan <- msg
		}
	}

	//	gChan := groups[gName]
	//	gChan <- msg

	log.Printf("[broadcast] Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)
	//	for _, buddy := range groupClients[gName] {
	//		if buddy != guy {
	//			log.Printf("Friend " + guy + " sent " + gName + " a message: " + msg.Message)
	//			gChan <- msg
	//		}
	//	}
}

func listenToClient(stream pb.Chat_RouteChatServer, messages chan<- pb.ChatMessage) {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
		}
		if err != nil {
		}
		log.Printf("[listenToClient] Client " + msg.Sender + " sent " + msg.Receiver + " a message: " + msg.Message)
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
