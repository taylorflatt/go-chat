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

// The port the server is listening on.
const (
	port = ":12021"
)

type server struct{}

type Group struct {
	name      string
	ch        chan pb.ChatMessage
	clients   []string
	WaitGroup *sync.WaitGroup
}

type Client struct {
	name      string
	groups    []string
	ch        chan pb.ChatMessage
	WaitGroup *sync.WaitGroup
}

var lock = &sync.RWMutex{}
var clients = make(map[string]*Client)
var groups = make(map[string]*Group)

// AddClient adds a new client n to the server.
// It doesn't return anything.
func AddClient(n string) {

	lock.Lock()
	defer lock.Unlock()

	c := &Client{
		name:      n,
		ch:        make(chan pb.ChatMessage, 100),
		WaitGroup: &sync.WaitGroup{},
	}

	log.Print("[AddClient]: Registered client " + n)
	clients[n] = c
}

// AddGroup adds a new group to the server.
// It doesn't return anything.
func AddGroup(n string) {

	lock.Lock()
	defer lock.Unlock()

	g := &Group{
		name:      n,
		ch:        make(chan pb.ChatMessage, 100),
		WaitGroup: &sync.WaitGroup{},
	}

	log.Print("[AddGroup]: Added group " + g.name)
	groups[n] = g
	groups[n].WaitGroup.Add(1)
}

// ClientExists checks if a client exists on the server.
// It returns a bool value.
func ClientExists(n string) bool {

	lock.RLock()
	defer lock.RUnlock()
	for c := range clients {
		if c == n {
			return true
		}
	}

	return false
}

// GroupExists checks if a group exists on the server.
// It returns a bool value.
func GroupExists(gName string) bool {

	lock.RLock()
	defer lock.RUnlock()
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
func InGroup(n string) bool {

	for _, g := range groups {
		for _, c := range g.clients {
			if n == c {
				return true
			}
		}
	}

	return false
}

// RemoveClient will remove a client from the server as well as any
// groups that they are currently in.
// It returns an error.
func RemoveClient(name string) error {

	// TODO: There is some deadlock here when a user attempts to quit
	// 		 the chat app with !exit.

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
func AddClientToGroup(c string, g string) {

	//lock.Lock()
	//defer lock.Unlock()

	groups[g].WaitGroup.Add(1)
	defer groups[g].WaitGroup.Done()

	groups[g].clients = append(groups[g].clients, c)
	clients[c].groups = append(clients[c].groups, g)

	log.Println("[AddClientToGroup] Added " + c + " to " + g)
}

// RemoveClientFromGroup will remove a client from a specific group. It will also
// delete a group if the client is the last one leaving it.
// It returns an error.
func RemoveClientFromGroup(n string) error {

	for _, g := range groups {
		for i, c := range g.clients {
			if n == c {
				c := clients[n].groups
				// Remove the group from the user.
				for i, _ := range c {
					if n == g.name {
						c[i] = c[len(c)-1]
						c = c[:len(c)-1]
						clients[n].groups = c
					}
				}
				if len(g.clients) == 1 {
					delete(groups, g.name)
				} else {
					c := g.clients
					c[i] = c[len(c)-1]
					c = c[:len(c)-1]
					g.clients = c
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

	var c []string
	for key := range clients {
		c = append(c, key)
	}

	log.Print("[GetClientList]: Returned list of current groups ")
	log.Print(c)

	return &pb.ClientList{Clients: c}, nil
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

	g := in.GroupName

	if !GroupExists(g) {
		return &pb.ClientList{}, errors.New("that group doesn't exist")
	}

	lst := groups[g].clients

	log.Print("[GetGroupClientList]: For group " + g + " returned members ")
	log.Print(lst)

	return &pb.ClientList{Clients: lst}, nil
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

	u := in.Sender

	log.Print("[UnRegister]: Unregistering client " + u)

	err := RemoveClient(u)

	log.Println("[UnRegister]: The following are the remaining clients, ")
	keys := []string{}
	for _, c := range clients {
		keys = append(keys, c.name)
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

	c := in.Client
	g := in.GroupName

	log.Printf("[JoinGroup] Attempting to add " + c + " to " + g)

	if GroupExists(g) {
		AddClientToGroup(c, g)

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
		case inMsg := <-clients[msg.Sender].ch:
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
			for _, c := range groups[gn].clients {
				log.Printf("[Broadcast]: I found " + c + " in gName")
				if c == msg.Sender && msg.Message == msg.Sender+" left chat!\n" {
					log.Printf("[Broadcast]: ADDING THE KILL MESSAGE TO " + c)
					clients[c].ch <- msg
				} else if c != msg.Sender {
					log.Printf("[Broadcast] Adding the message to " + c + "'s channel.")
					clients[c].ch <- msg
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
