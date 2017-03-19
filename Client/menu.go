package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	pb "github.com/taylorflatt/go-chat"
)

func AddSpacing(n int) {

	for i := 0; i <= n; i++ {
		fmt.Println()
	}
}

func StartMessage() {

	AddSpacing(1)
	fmt.Println("Welcome to Go-Chat!")
	Frame()
	fmt.Println("In order to begin chatting, you must first chose a server and username. It cannot be one")
	fmt.Println("that is already in user on the server. Remember, your username only lasts for as long as")
	fmt.Println("you are logged into the server!")
	AddSpacing(1)
}

func WelcomeMessage(u string) {

	AddSpacing(1)
	fmt.Println("Welcome, " + u + "!")
}

func MainMenuText() {

	fmt.Println("Main Menu")
	AddSpacing(1)
	fmt.Println("1) Create a Group")
	fmt.Println("2) View Group Options")
	fmt.Println("3) Exit Chat")
	AddSpacing(1)
	fmt.Print("Main> ")
}

func GroupMenuText() {

	fmt.Println("View Groups Menu")
	AddSpacing(1)
	fmt.Println("Below is a list of menu options for groups.")
	AddSpacing(1)
	fmt.Println("1) View a Group's Members")
	fmt.Println("2) Refresh List of Groups")
	fmt.Println("3) Join a Group")
	fmt.Println("4) Go back")
	AddSpacing(1)
	fmt.Print("Groups> ")
}

func ViewGroupMemMenuText() {

	AddSpacing(1)
	fmt.Println("Enter the group name that you would like to view! Enter !back to go back to the menu.")
	AddSpacing(1)
	fmt.Print("View> ")
}

func Frame() {

	fmt.Println("------------------------------------------")
}

func SetServer(r *bufio.Reader) string {

	StartMessage()

	fmt.Print("Please specify the server IP: ")
	t, _ := r.ReadString('\n')
	t = strings.TrimSpace(t)
	s := strings.Split(t, ":")
	ip := s[0]
	p := s[1]
	address := ip + ":" + p

	return address

}

func SetName(r *bufio.Reader) string {
	for {
		fmt.Printf("Enter your username: ")
		n, err := r.ReadString('\n')
		if err != nil {
			fmt.Print(err)
		} else {
			uName := strings.TrimSpace(n)
			WelcomeMessage(uName)
			return uName
		}
	}
}

func CreateGroup(c pb.ChatClient, r *bufio.Reader, uName string) (string, error) {

	for {
		AddSpacing(1)
		fmt.Println("Enter the name of the group or type !back to go back to the main menu.")
		fmt.Print("Group Name> ")
		gName, err := r.ReadString('\n')
		gName = strings.TrimSpace(gName)

		if err != nil {
			return "", err
		} else if gName != "!back" {
			_, nerr := c.CreateGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gName})

			if nerr != nil {
				fmt.Println("That group name has already been chosen. Please select a new one.")
			} else {
				c.JoinGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: gName})
				fmt.Println("Created and joined group named " + gName)
				return gName, nil
			}
		} else {
			return gName, nil
		}
	}
}

func JoinGroup(c pb.ChatClient, r *bufio.Reader, u string) string {

	for {
		fmt.Println("Enter the name of the group as it appears in the group list or enter !back to go back to the Group menu.")
		fmt.Print("Group Name> ")
		g, _ := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if g == "!back" {
			return g
		}

		_, err := c.JoinGroup(context.Background(), &pb.GroupInfo{Client: u, GroupName: g})

		if err != nil {
			fmt.Print("A group with that name doesn't exist. Please check again.")
		} else {
			fmt.Println("Joined " + g)
			return g
		}
	}
}

func ListGroups(c pb.ChatClient, r *bufio.Reader) {

	t, _ := c.GetGroupList(context.Background(), &pb.Empty{})
	l := t.Groups

	if len(l) == 0 {
		fmt.Println("There are no groups created yet!")
	} else {
		for i, g := range l {
			fmt.Println("  " + strconv.Itoa(i+1) + ") " + g)
		}
	}

}

func ListGroupMembers(c pb.ChatClient, r *bufio.Reader, u string) error {

	for {
		t, _ := c.GetGroupList(context.Background(), &pb.Empty{})
		n := len(t.Groups)

		if n == 0 {
			fmt.Println("There are currently no groups created!")
			return nil
		}

		g, err := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if err != nil {
			return err
		} else if g == "!back" {
			return nil
		} else {
			ls, err := c.GetGroupClientList(context.Background(), &pb.GroupInfo{Client: u, GroupName: g})
			if err != nil {
				fmt.Println("Please double check that the group name you entered actually exists.")
			} else {
				fmt.Println("Members of " + g)
				for i, c := range ls.Clients {
					fmt.Println("  " + strconv.Itoa(i+1) + ") " + c)
				}

				return nil
			}
		}
	}
}

func TopMenu(c pb.ChatClient, r *bufio.Reader) ([2]string, error) {

	var o [2]string
	u := SetName(r)
	o[0] = u

	for {
		Frame()
		MainMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {
		case "1": // Create group
			g, err := CreateGroup(c, r, u)

			if err != nil {
				return o, err
			} else if g != "!back" {
				o[1] = g
				return o, nil
			}
		case "2": // View Group Menu
			g, err := DisplayGroupMenu(c, r, u)

			if err != nil {
				return o, err
			} else if g != "!back" {
				o[1] = g
				return o, nil
			}
		case "3": // Exit Client
			c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
			os.Exit(0)
		default: // Error
			fmt.Println("Please enter a valid selection between 1 and 3.")
		}
	}
}

func DisplayGroupMenu(c pb.ChatClient, r *bufio.Reader, u string) (string, error) {

	ListGroups(c, r)

	for {
		Frame()
		GroupMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {
		case "1": // View Group Members
			ViewGroupMemMenuText()
			err := ListGroupMembers(c, r, u)
			if err != nil {
				return "", err
			}
		case "2": // Refresh Group List
			ListGroups(c, r)
			break
		case "3": // Join Group
			g := JoinGroup(c, r, u)
			return g, nil
		case "4": // Go Back
			return "!back", nil
		default: // Error
			fmt.Println("Please enter a valid selection between 1 and 4.")
		}
	}
}
