package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
	"strings"
	"sync"
)

// Matrix is 2d
type Matrix [][]int

// same code as worker to iniliaze our structures.
type Task struct {
	Operation string
	MatrixA   Matrix
	MatrixB   Matrix
}

// TaskResult holds the computation result.
type TaskResult struct {
	Result Matrix
	Error  string
}

// Internally queue tasks to handle multiple client requests.
type TaskJob struct {
	Task   Task
	Result chan TaskResult //built-in type used for communication between goroutines. chan means channel.
}

// WorkerInfo holds the address and current load of a worker.
type WorkerInfo struct {
	Address string
	Load    int
}

// yahan we are keeping record of the workers and a mutex for safe access.
type Coordinator struct {
	workers []WorkerInfo
	mu      sync.Mutex
}

// Global variable to hold TLS config
var workerTLSConfig *tls.Config

// Load balancing yahan implement kar rahy
// 1) Locks the worker list to prevent race conditions.
// 2) Iterates through all registered workers and finds the one with the smallest Load.
// 3) Returns the index of the least busy worker.
// 4) If no workers are available, returns -1 and an error.
func (c *Coordinator) selectLeastBusyWorker() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.workers) == 0 {
		return -1, errors.New("no workers available")
	}
	//initialize forst worker as least busy and then iterating over the workers list to find least busy
	minIndex := 0
	minLoad := c.workers[0].Load
	for i, worker := range c.workers {
		if worker.Load < minLoad {
			minLoad = worker.Load
			minIndex = i
		}
	}
	return minIndex, nil
}

// jb hum assign a new task to a worker kar rahy then we do the following:
// 1) Locks the worker list to ensure thread safety.
// 2) Increments the Load value of the worker at `index`.
// 3) Unlocks after updating.

func (c *Coordinator) incrementLoad(index int) {
	c.mu.Lock()
	c.workers[index].Load++
	c.mu.Unlock()
}

// Ab jb a worker finishes a task then uski task count bhi decrement karne hai, so we do the following:
// 1) Locks the worker list to ensure thread safety.
// 2) Ensures Load **never becomes negative** (Load should be ≥ 0).
// 3) Unlocks after updating.

func (c *Coordinator) decrementLoad(index int) {
	c.mu.Lock()
	if c.workers[index].Load > 0 {
		c.workers[index].Load--
	}
	c.mu.Unlock()
}

// yahan pr jitne registered workers hein unhe task assign ho rahy and all the functions we declared above are getting called here
// An imp thing is that if a worker’s RPC call fails, it is removed from the pool and the task is retried.
func (c *Coordinator) assignTask(task Task) TaskResult {
	var finalResult TaskResult

	for {
		index, err := c.selectLeastBusyWorker()
		if err != nil {
			finalResult.Error = "no workers available" //least busy banda dhondho
			return finalResult
		}
		// Increase the load counter for the chosen worker.
		c.incrementLoad(index)
		workerAddr := c.workers[index].Address

		// Dial the worker using TLS.
		conn, err := tls.Dial("tcp", workerAddr, workerTLSConfig)
		if err != nil { //Connection failed; decrease worker's load and remove worker from list.
			c.decrementLoad(index)
			c.mu.Lock()
			fmt.Println("Worker", workerAddr, "failed to connect; removing from list")
			c.workers = append(c.workers[:index], c.workers[index+1:]...)
			c.mu.Unlock()
			if len(c.workers) == 0 {
				finalResult.Error = "no workers available after failure"
				return finalResult
			}
			continue // try next available worker
		}

		// Create an RPC client over the TLS connection.
		rpcClient := rpc.NewClient(conn)
		var res TaskResult
		//Call the worker’s `ProcessTask` RPC method to execute the task.
		rpcCallErr := rpcClient.Call("WorkerService.ProcessTask", task, &res)
		rpcClient.Close() //after task is doen
		c.decrementLoad(index)
		//if rpc call fails tou fault tolerance yahan apply kar rahy
		if rpcCallErr != nil {
			fmt.Println("RPC call error on worker", workerAddr, ":", rpcCallErr)
			c.mu.Lock()
			fmt.Println("Removing worker", workerAddr, "from list due to RPC error") //Remove the failed worker from the worker list.
			c.workers = append(c.workers[:index], c.workers[index+1:]...)
			c.mu.Unlock()
			if len(c.workers) == 0 {
				finalResult.Error = "no workers available after RPC errors"
				return finalResult
			}
			continue
		}
		// Successfully got a result.
		return res
	}
}

// RPC service that clients call.
type CoordinatorService struct {
	taskQueue chan TaskJob
}

// yahan pr we are not not processing tasks directly; we implemented asynchronous handling.
// yahan  we made a result channel to receive the computed result.
// 1) The task is sent into the queue
// 2) The function blocj until the result is received
// 3) Once processes the task, the result is returned to the client.
func (cs *CoordinatorService) Compute(task Task, reply *TaskResult) error {
	job := TaskJob{
		Task:   task,
		Result: make(chan TaskResult),
	}
	cs.taskQueue <- job
	res := <-job.Result
	*reply = res
	return nil
}

// dispatcher reads tasks from the queue and assigns them to workers.
func dispatcher(coordinator *Coordinator, taskQueue chan TaskJob) {
	for job := range taskQueue {
		res := coordinator.assignTask(job.Task)
		job.Result <- res
	}
}

func main() {
	// these flags are the same as worker
	portPtr := flag.String("port", "8000", "Port for coordinator to listen on")
	workersPtr := flag.String("workers", "", "Comma-separated list of worker addresses (host:port)")
	certFile := flag.String("cert", "server.crt", "Path to TLS certificate file")
	keyFile := flag.String("key", "server.key", "Path to TLS key file")
	caCertFile := flag.String("cacert", "server.crt", "Path to CA certificate file (used for verifying workers)")
	flag.Parse()

	//atleast ek worker hamesha ho
	if *workersPtr == "" {
		fmt.Println("Please provide a comma-separated list of worker addresses using --workers")
		os.Exit(1)
	}

	// Build the list of workers.
	workerAddrs := strings.Split(*workersPtr, ",")
	workers := make([]WorkerInfo, 0, len(workerAddrs)) //// Split worker list by commas
	for _, addr := range workerAddrs {
		trimmed := strings.TrimSpace(addr)
		if trimmed != "" {
			workers = append(workers, WorkerInfo{Address: trimmed, Load: 0})
		}
	}

	//coordinator bana rahy with the workers lsit jo humenin upar milein hein through cmd
	coordinator := &Coordinator{
		workers: workers,
	}

	// Prepare the task queue.
	taskQueue := make(chan TaskJob, 100)
	coordService := &CoordinatorService{
		taskQueue: taskQueue,
	}

	//rpc comm initialize kar rahy
	err := rpc.Register(coordService)
	if err != nil {
		fmt.Println("Error registering CoordinatorService:", err)
		os.Exit(1)
	}

	// start processing task using the go routine
	go dispatcher(coordinator, taskQueue)

	// Load the TLS info and keys and certs
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		fmt.Println("Error loading TLS certificate/key:", err)
		os.Exit(1)
	}
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// ye code tb use karte when worker and coordinator are on different laptops
	//tb hum ek CA banaty jo sign karta worker and coordinator ka cert
	caCert, err := ioutil.ReadFile(*caCertFile)
	if err != nil {
		fmt.Println("Error reading CA certificate file:", err)
		os.Exit(1)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		fmt.Println("Failed to append CA certificate")
		os.Exit(1)
	}
	workerTLSConfig = &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	// Listen with TLS.
	listener, err := tls.Listen("tcp", ":"+*portPtr, serverTLSConfig)
	if err != nil {
		fmt.Println("Listener error:", err)
		os.Exit(1)
	}
	fmt.Println("Coordinator (TLS) listening on port", *portPtr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection accept error:", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

/*
openssl req -x509 -nodes -newkey rsa:4096 -keyout server.key -out server.crt -days 365 -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:172.17.18.45"
yahan pr mera current ip ise hoga
and new cert generate hoga. wou cert copy karon ge worker directory mein and server.crt will be shared with client
uske baad worker run karna hoga using this
cd worker
PS M:\Blockchain & Crypto\Assignment 1\worker> go run worker.go --port=9001 --cert=server.crt --key=server.key
agar dosra worker banaya then port will be 9002 and so on
jb worker run ho jaye ga then coordinator ho ga

go run coordinator.go --port=8000 --workers=172.17.18.45:9001,172.17.18.45:9002 --cert=server.crt --key=server.key --cacert=server.crt
and then client run hoga


*/
