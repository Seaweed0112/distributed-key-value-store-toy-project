package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Node struct {
	id    int
	mu    sync.Mutex
	store map[string]int
	Port  int
}

func NewNode(id int, port int) *Node {
	return &Node{
		id:    id,
		store: make(map[string]int),
		Port:  port,
	}
}

func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 1024)
	nRead, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}

	request := strings.TrimSpace(string(buf[:nRead]))
	if request == "" {
		return
	}

	parts := strings.SplitN(request, " ", 2)
	if parts[0] != "status" && len(parts) < 2 {
		fmt.Println("Invalid request:", request)
		return
	}

	command := parts[0]

	switch command {
	case "update":
		event := parts[1]
		n.update(event)
		conn.Write([]byte("Update successful\n"))
	case "get":
		event := parts[1]
		count := n.get(event)
		response := fmt.Sprintf("Count for %s: %d\n", event, count)
		conn.Write([]byte(response))
	case "status":
		conn.Write([]byte(n.status()))
	default:
		fmt.Println("Unknown command:", command)
	}
}

func (n *Node) update(event string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.store[event]++
}

func (n *Node) get(event string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.store[event]
}

func (n *Node) status() string {
	n.mu.Lock()
	defer n.mu.Unlock()

	var status strings.Builder
	for event, count := range n.store {
		status.WriteString(fmt.Sprintf("%s: %d\n", event, count))
	}
	if status.Len() == 0 {
		status.WriteString("No events\n")
	}
	return status.String()
}

func (n *Node) Start() {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(n.Port))
	if err != nil {
		fmt.Printf("Failed to listen on port %d: %v\n", n.Port, err)
		return
	}
	defer listener.Close()

	fmt.Printf("Node listening on port %d\n", n.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go n.handleConnection(conn)
	}
}

func (n *Node) Update(event string) error {
	_, err := n.sendCommand("update " + event)
	return err
}

func (n *Node) Get(event string) (string, error) {
	return n.sendCommand("get " + event)
}

func (n *Node) GetStatus() (string, error) {
	return n.sendCommand("status")
}

func (n *Node) sendCommand(command string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", n.Port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(command))
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	nRead, err := conn.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf[:nRead]), nil
}
