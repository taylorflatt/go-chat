package main

import (
	"errors"
	"io"
	"log"
	"net"
	"strconv"

	pb "github.com/taylorflatt/go-chat"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":12021"
)

// Server is used to implement the RemoteCommandServer
type server struct{}

type Tuple struct {
	a, b interface{}
}

// List of clients connected to the server. Here ip is key and port is value.
var clients []Tuple

// RemoveElement swaps the element to delete with the one at the end of the array then resizes the
// array to the new size. It returns a tuple without the element at index i.
func RemoveElement(a []Tuple, i int) []Tuple {
	a[len(a)-1], a[i] = a[i], a[len(a)-1]
	return a[:len(a)-1]
}

// Registers the client with the server as available to talk.
func (s *server) RegisterClient(ctx context.Context, in *pb.ClientInfo) (*pb.Response, error) {

	var ip = in.Ip
	var port = in.Port
	var exists = false

	// Check if the ip/port are already registered.
	for _, value := range clients {
		if value.a == ip && value.b == port {
			exists = true
		}
	}

	if exists == false {
		pair := Tuple{ip, port}
		clients = append(clients, pair)
	} else {
		log.Println("Duplicate host attempted to register. Disallowing registration...")
		return &pb.Response{}, errors.New("ip: " + ip + " and port: " + strconv.Itoa(int(port)) + " are already registered on the server")
	}

	log.Println("Registered " + ip + ":" + strconv.Itoa(int(port)))
	log.Print("Current Clients: ")
	log.Print(clients)

	return &pb.Response{}, nil
}

// Unregisters the client with the server by deleting it from the global list.
func (s *server) UnRegisterClient(ctx context.Context, in *pb.ClientInfo) (*pb.Response, error) {

	var ip = in.Ip
	var port = in.Port

	for i, value := range clients {
		if value.a == ip && value.b == port {
			log.Println("Removing " + ip + ":" + strconv.Itoa(int(port)))
			clients = RemoveElement(clients, i)
			break
		}
	}

	return &pb.Response{}, nil
}

// Sends the list of currently registered IPs to the client.
func (s *server) GetClientList(ctx context.Context, in *pb.List) (*pb.ClientList, error) {

	// List of just the IPs.
	var cIps []string
	var cPorts []int32

	for _, value := range clients {
		cIps = append(cIps, value.a.(string))
		cPorts = append(cPorts, value.b.(int32))
	}

	return &pb.ClientList{Ip: cIps, Port: cPorts}, nil
}

func (s *server) RouteChat(stream pb.Chat_RouteChatServer) error {

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		log.Print(in.Ip + " sent " + in.Message)

		stream.Send(in)
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
