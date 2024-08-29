package main

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ConsistentHashing struct {
	sync.RWMutex
	hashRing      []int
	nodeMap       map[int]*Node
	replicaFactor int
	nodeCount     int
}

func NewConsistentHashing(replicaFactor int) *ConsistentHashing {
	return &ConsistentHashing{
		nodeMap:       make(map[int]*Node),
		replicaFactor: replicaFactor,
	}
}

func (ch *ConsistentHashing) AddNode(node *Node) {
	ch.Lock()
	defer ch.Unlock()

	ch.nodeCount++
	for i := 0; i < ch.replicaFactor; i++ {
		hash := int(hashFunction(strconv.Itoa(node.id) + "-" + strconv.Itoa(i)))
		ch.hashRing = append(ch.hashRing, hash)
		ch.nodeMap[hash] = node
	}

	sort.Ints(ch.hashRing)
}

func (ch *ConsistentHashing) GetNodes(event string) []*Node {
	ch.RLock()
	defer ch.RUnlock()

	var nodes []*Node
	hash := int(hashFunction(event))
	for i := 0; i < ch.replicaFactor; i++ {
		idx := ch.search(hash)
		nodes = append(nodes, ch.nodeMap[ch.hashRing[idx]])
		hash = ch.hashRing[(idx+1)%len(ch.hashRing)]
	}

	return nodes
}

func hashFunction(event string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(event))
	return hash.Sum32()
}

func (ch *ConsistentHashing) search(hash int) int {
	idx := sort.Search(len(ch.hashRing), func(i int) bool {
		return ch.hashRing[i] >= hash
	})
	if idx == len(ch.hashRing) {
		return 0
	}
	return idx
}

func startMainServer(port int, ch *ConsistentHashing, ready chan<- bool) {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	defer listener.Close()

	log.Printf("Main server listening on port %d\n", port)
	ready <- true

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn, ch)
	}
}

func handleConnection(conn net.Conn, ch *ConsistentHashing) {
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("Error reading from connection:", err)
		return
	}

	request := string(buf[:n])
	if request == "" {
		return
	}

	parts := strings.SplitN(request, " ", 2)
	if len(parts) < 2 {
		log.Println("Invalid request:", request)
		return
	}

	command := parts[0]
	event := parts[1]
	nodes := ch.GetNodes(event)

	for _, node := range nodes {
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", node.Port))
		if err != nil {
			log.Println("Error connecting to node:", err)
		} else {
			// Handle update/get command with node
		}
	}

	if command == "get" {
		_, err := conn.Write([]byte("Response: OK\n"))
		if err != nil {
			log.Println("Error writing response to connection:", err)
		}
	}
}

func findAvailablePort() int {
	for port := 5000; port <= 65535; port++ {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
		if err != nil {
			return port
		}
		conn.Close()
	}
	return -1
}

func sendCommandToMainServer(address, request string) (string, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(request))
	if err != nil {
		return "", err
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf[:n]), nil
}

func main() {
	fmt.Println("Welcome to Consistent Hashing based Distributed Key-Value Store")

	ch := NewConsistentHashing(3)
	availablePorts := []int{}
	nodeId := 0

	ready := make(chan bool)

	// Start the main server in a separate goroutine
	go startMainServer(4000, ch, ready)

	// Wait for the server to be ready
	<-ready

	reader := bufio.NewReader(os.Stdin)
	for {
		// check node count
		// if node count is 0, ask user to input node count
		if ch.nodeCount == 0 {
			fmt.Print("Enter number of nodes to start: ")
			fmt.Print("> ")
			input, _ := reader.ReadString('\n')
			nodeCount, err := strconv.Atoi(strings.TrimSpace(input))
			if err != nil {
				log.Println("Invalid input. Please enter a valid number.")
				continue
			}

			// start nodes
			for i := 0; i < nodeCount; i++ {
				port := findAvailablePort()
				if port == -1 {
					log.Fatalf("Failed to find available port for node %d", i+1)
				}

				n := NewNode(nodeId, port)
				go n.Start()

				availablePorts = append(availablePorts, port)
				ch.AddNode(n)

				log.Printf("Started node %d on port %d", nodeId, port)
				nodeId++
			}
		}
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			fmt.Println("Exiting...")
			break
		} else if input == "status" {
			fmt.Printf("Nodes running on ports: %v\n", availablePorts)
		} else if strings.HasPrefix(input, "get ") {
			event := strings.TrimPrefix(input, "get ")
			response, err := sendCommandToMainServer("127.0.0.1:4000", "get "+event)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println(response)
			}
		} else if strings.HasPrefix(input, "update ") {
			event := strings.TrimPrefix(input, "update ")
			response, err := sendCommandToMainServer("127.0.0.1:4000", "update "+event)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println(response)
			}
		} else {
			fmt.Println("Unknown command. Available commands: status, exit")
		}
	}
}
