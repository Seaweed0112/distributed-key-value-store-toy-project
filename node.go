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
	if len(parts) < 2 {
		fmt.Println("Invalid request:", request)
		return
	}

	command := parts[0]
	event := parts[1]

	switch command {
	case "update":
		n.update(event)
		conn.Write([]byte("Update successful\n"))
	case "get":
		count := n.get(event)
		response := fmt.Sprintf("Count for %s: %d\n", event, count)
		conn.Write([]byte(response))
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
