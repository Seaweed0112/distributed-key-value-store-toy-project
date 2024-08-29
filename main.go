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
	// log.Printf("hashRing: %v\n", ch.hashRing)

	sort.Ints(ch.hashRing)
}

func (ch *ConsistentHashing) GetNode(event string) *Node {
	ch.RLock()
	defer ch.RUnlock()

	var node *Node
	hash := int(hashFunction(event))
	idx := ch.search(hash)
	node = ch.nodeMap[ch.hashRing[idx]]

	return node
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

func (ch *ConsistentHashing) HandleRequest(command, event string) (string, error) {
	node := ch.GetNode(event)
	var response string

	switch command {
	case "update":
		err := node.Update(event)
		if err != nil {
			return "", err
		}
		response = "Update successful\n"
	case "get":
		response, err := node.Get(event)
		if err == nil {
			return response, nil
		}
		response = "Event not found\n"
	default:
		response = "Unknown command\n"
	}

	return response, nil
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

	response, err := ch.HandleRequest(command, event)
	if err != nil {
		log.Println("Error handling request:", err)
		response = "Error handling request\n"
	}

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Println("Error writing response to connection:", err)
	}
}

func findAvailablePort(start int) int {
	for port := start; port <= 65535; port++ {
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

func sendCommandToNode(port int, request string) (string, error) {
	address := fmt.Sprintf("127.0.0.1:%d", port)
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

func showAllCounters(ports []int) {
	fmt.Println("Current counters on all nodes:")
	for _, port := range ports {
		fmt.Printf("Node on port %d:\n", port)
		response, err := sendCommandToNode(port, "status")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println(response)
		}
	}
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
			// wait until all nodes are started
			var wg = &sync.WaitGroup{}
			start := 5000
			for i := 0; i < nodeCount; i++ {
				port := findAvailablePort(start)
				start = port + 1
				if port == -1 {
					log.Fatalf("Failed to find available port for node %d", i+1)
				}

				wg.Add(1)
				go func(port int) {
					n := NewNode(nodeId, port)
					go n.Start()
					ch.AddNode(n)
					wg.Done()
				}(port)

				availablePorts = append(availablePorts, port)

				log.Printf("Started node %d on port %d", nodeId, port)
				nodeId++
			}
			wg.Wait()
		}
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			fmt.Println("Exiting...")
			break
		} else if input == "status" {
			fmt.Printf("Nodes running on ports: %v\n", availablePorts)
			showAllCounters(availablePorts)

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
			fmt.Println("Unknown command. Available commands: get {event}, update {event}, status, exit")
		}
	}
}
