/*
* Author: Taylor Flatt
*
* Menu handling system for the chat client.
*
 */

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	pb "github.com/taylorflatt/go-chat"
)

// Stores the main color for all the entry dialogs.
const (
	promptColor = color.FgHiMagenta
)

// RandColor picks a random from a stored list of colors.
// It returns a single color attribute.
func RandColor() color.Attribute {

	c := []color.Attribute{color.FgHiCyan, color.FgHiGreen, color.FgHiRed, color.FgHiWhite, color.FgHiYellow, color.FgHiMagenta}
	return c[(len(c)*rand.Intn(20)+rand.Intn(10)+2)%len(c)]
}

// AddSpacing adds n-new lines to console.
// It doesn't return anything.
func AddSpacing(n int) {

	for i := 0; i < n; i++ {
		fmt.Println()
	}
}

// StartMessage displays the menu text for after a user connects to the server.
// It doesn't return anything.
func StartMessage() {

	AddSpacing(1)
	fmt.Println("Welcome to Go-Chat!")
	Frame()
	fmt.Println("In order to begin chatting, you must first chose a server and username. It cannot be one")
	fmt.Println("that is already in user on the server. Remember, your username only lasts for as long as")
	fmt.Println("you are logged into the server!")
	AddSpacing(1)
}

// WelcomeMessage displays a colored string welcoming the user to the server.
// It doesn't return anything.
func WelcomeMessage(c pb.ChatClient, u string) {

	AddSpacing(1)
	u = "Welcome " + u + "!"
	for _, l := range u {
		color.New(RandColor()).Print(string(l))
	}
	n, _ := c.GetClientList(context.Background(), &pb.Empty{})
	g, _ := c.GetGroupList(context.Background(), &pb.Empty{})

	fmt.Print(" There are currently " + strconv.Itoa(len(n.Clients)) + " member(s) logged in and " + strconv.Itoa(len(g.Groups)) + " group(s).")
	AddSpacing(1)
}

// TopMenuText displays the option text for the main menu.
// It doesn't return anything.
func TopMenuText() {

	fmt.Println("Main Menu")
	AddSpacing(1)
	fmt.Println("1) Create a Group")
	fmt.Println("2) View Group Options")
	fmt.Println("3) Exit Chat")
	AddSpacing(1)
	color.New(promptColor).Print("Main> ")
}

// GroupMenuText displays the option text for the group menu.
// It doesn't return anything.
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
	color.New(promptColor).Print("Groups> ")
}

// ViewGroupMemMenuText displays option text to view a group.
// It doesn't return anything.
func ViewGroupMemMenuText() {

	AddSpacing(1)
	fmt.Println("Enter the group name that you would like to view! Enter !back to go back to the menu.")
	AddSpacing(1)
}

// Frame gives some nice formatting structure to the output.
func Frame() {

	fmt.Println("------------------------------------------")
}

// SetServer handles the input for the chat server address.
// It returns a string which contains the ip:port of the chat server.
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

// SetName sets the username for the user.
// It returns a string containing the username of the client.
func SetName(c pb.ChatClient, r *bufio.Reader) string {
	for {
		fmt.Printf("Enter your username: ")
		n, err := r.ReadString('\n')
		if err != nil {
			fmt.Print(err)
		} else {
			uName := strings.TrimSpace(n)
			if len(uName) < 3 {
				AddSpacing(1)
				color.New(color.FgHiRed).Println("Your username must be at least 3 characters long.")
			} else {
				_, err = c.Register(context.Background(), &pb.ClientInfo{Sender: uName})

				if err != nil {
					AddSpacing(1)
					color.New(color.FgHiRed).Println("That username already exists. Please choose a new one! ")
				} else {
					WelcomeMessage(c, uName)
					return uName
				}
			}
		}
	}
}

// CreateGroup handles the create group menu option.
// It returns a string which contains the keyword !back allowing it to escape the input as well as an error.
func CreateGroup(c pb.ChatClient, r *bufio.Reader, uName string) (string, error) {

	for {
		AddSpacing(1)
		fmt.Println("Enter the name of the group or type !back to go back to the main menu.")
		color.New(promptColor).Print("Join> ")
		g, err := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if err != nil {
			return "", err
		} else if g != "!back" {
			_, nerr := c.CreateGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: g})

			if nerr != nil {
				AddSpacing(1)
				color.New(color.FgRed).Println("The group name \"" + g + "\" has already been chosen. Please select a new one.")
			} else {
				c.JoinGroup(context.Background(), &pb.GroupInfo{Client: uName, GroupName: g})
				AddSpacing(1)
				color.New(color.FgGreen).Println("Created and joined group named " + g)
				return g, nil
			}
		} else {
			return g, nil
		}
	}
}

// JoinGroup handles the join group menu option.
// It returns a string which contains the keyword !back allowing it to escape the input.
func JoinGroup(c pb.ChatClient, r *bufio.Reader, u string) string {

	for {
		fmt.Println("Enter the name of the group as it appears in the group list or enter !back to go back to the Group menu.")
		color.New(promptColor).Print("Group Name> ")
		g, _ := r.ReadString('\n')
		g = strings.TrimSpace(g)

		if g == "!back" {
			return g
		}

		_, err := c.JoinGroup(context.Background(), &pb.GroupInfo{Client: u, GroupName: g})

		if err != nil {
			AddSpacing(1)
			color.New(color.FgRed).Println("The group name \"" + g + "\" doesn't exist. Please check again.")
			AddSpacing(1)
		} else {
			color.New(color.FgGreen).Println("Joined " + g)
			return g
		}
	}
}

// ListGroups handles listing all of the groups stored on the server.
// It doesn't return anything.
func ListGroups(c pb.ChatClient, r *bufio.Reader) {

	t, _ := c.GetGroupList(context.Background(), &pb.Empty{})
	l := t.Groups

	if len(l) == 0 {
		AddSpacing(1)
		color.New(color.FgYellow).Println("There are no groups created yet!")
	} else {
		AddSpacing(1)
		fmt.Println("Current groups able to join:")
		for i, g := range l {
			fmt.Println("  " + strconv.Itoa(i+1) + ") " + g)
		}
	}

}

// ListGroupMembers handles listing the members of a specific group.
// It returns an error.
func ListGroupMembers(c pb.ChatClient, r *bufio.Reader, u string) error {

	for {
		color.New(promptColor).Print("View> ")
		t, _ := c.GetGroupList(context.Background(), &pb.Empty{})
		n := len(t.Groups)

		if n == 0 {
			AddSpacing(2)
			color.New(color.FgYellow).Println("There are currently no groups created!")
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
				color.New(color.FgRed).Println("Please double check that the group name you entered actually exists.")
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

// TopMenu handles displaying the menu to the client.
// It returns the group name for the user and an error.
func TopMenu(c pb.ChatClient, r *bufio.Reader, u string) (string, error) {
	//func TopMenu(c pb.ChatClient, u string) (string, error) {
	log.Println("In TopMenu")

	//r := bufio.NewReader(os.Stdin)

	for {
		Frame()
		TopMenuText()
		i, _ := r.ReadString('\n')
		i = strings.TrimSpace(i)

		switch input := i; input {
		case "1": // Create group
			g, err := CreateGroup(c, r, u)

			if err != nil {
				return g, err
			} else if g != "!back" {
				return g, nil
			}
		case "2": // View Group Menu
			g, err := DisplayGroupMenu(c, r, u)

			if err != nil {
				return g, err
			} else if g != "!back" {
				return g, nil
			}
		case "3": // Exit Client
			c.UnRegister(context.Background(), &pb.ClientInfo{Sender: u})
			os.Exit(0)
		default: // Error
			color.New(color.FgRed).Println("Please enter a valid selection between 1 and 3.")
		}
	}
}

// DisplayGroupMenu displays the menu for the group options.
// It returns either an empty string or the keyword !back to navigate to TopMenu.
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
			if g != "!back" {
				return g, nil
			}
		case "4": // Go Back
			return "!back", nil
		default: // Error
			color.New(color.FgRed).Println("Please enter a valid selection between 1 and 4.")
		}
	}
}
