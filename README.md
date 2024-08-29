# Distributed Key-Value Store with Consistent Hashing

## Overview

This project is a simple distributed key-value store implemented in Go, using consistent hashing to manage the distribution of keys across multiple nodes. The system allows you to dynamically add nodes and handle requests such as `get` and `update` for specific keys. It is designed to run all components (main server and consistent hashing) within a single program for simplicity and ease of use.

## Features

-   **Consistent Hashing**: Efficiently distributes keys across multiple nodes, allowing for scalable and fault-tolerant key-value storage.
-   **Dynamic Node Management**: Add nodes dynamically at runtime to balance the load and ensure fault tolerance.
-   **Basic Key-Value Operations**: Supports `get` and `update` commands to retrieve and increment event counts.
-   **Node Status Reporting**: Retrieve the current status of all nodes, including the count of all keys stored on each node.

## Architecture

The system is composed of two main components:

1. **Main Server**:

    - Handles user input via a command-line interface (CLI).
    - Sends commands to the consistent hashing server to manage nodes and handle key-value operations.

2. **Consistent Hashing Server**:
    - Manages the distribution of keys across nodes using consistent hashing.
    - Handles commands to add nodes, get key values, update key values, and report the status of all nodes.

## Installation

### Prerequisites

-   Go 1.15 or higher installed on your system.

### Clone the Repository

```bash
git@github.com:Seaweed0112/distributed-key-value-store-toy-project.git
cd distributed-key-value-store-toy-project
```

### Run the Program

```bash
go install
distributed-cache-system
```

## Usage

Once the program is running, you can interact with the system using the CLI. The following commands are supported:

### Commands

-   **status**:

    -   Displays the current counters on all nodes, including the count of all events stored on each node.

    ```bash
    > status
    ```

-   **get `<event_name>`**:

    -   Retrieves the count of the specified event.

    ```bash
    > get event1
    ```

-   **update `<event_name>`**:

    -   Increments the count of the specified event by 1.

    ```bash
    > update event1
    ```

-   **exit**:

    -   Exits the program.

    ```bash
    > exit
    ```

## How It Works

1. **Main Server**:

    - Starts and initializes the consistent hashing system.
    - Listens for user input and sends commands to the consistent hashing server.

2. **Consistent Hashing Server**:

    - Manages nodes and handles key-value operations.
    - Uses consistent hashing to determine which nodes should handle specific keys.
    - Dynamically adds nodes and redistributes keys when new nodes are added.

3. **Node Operations**:
    - Each node is responsible for storing key-value pairs and can be queried or updated through the consistent hashing server.
    - Nodes can report their current status, showing all keys and their associated counts.

## Example

Here’s an example interaction:

```bash
Welcome to Consistent Hashing based Distributed Key-Value Store
Enter number of nodes to start: > 3
Node listening on port 5001
Node listening on port 5002
Node listening on port 5003
> update a
Update successful
> get a
Count for a: 1
> status
Nodes running on ports: [5001 5002 5003]
Current counters on all nodes:
Node on port 5001:
a: 1

Node on port 5002:
No events

Node on port 5003:
No events
> exit
Exiting...
```

## TODO:

-   [ ] Add tests
-   [ ] Add more options: Add node, kill node
-   [ ] Fault tolerance and replication

## Welcome for suggestions!
