// importing the necessary packages
package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/rpc"
	"os"
)

// for the matrix operations
type Matrix [][]int

type Task struct {
	Operation string // yahan pr op can be "add", "multiply", or "transpose"
	MatrixA   Matrix
	MatrixB   Matrix
}

// holds the result of a computation.
type TaskResult struct {
	Result Matrix
	Error  string
}

// RPC service to process matrix tasks.
type WorkerService struct{}

// here we are performing the matrix operation requested.
// humne error handling bhi ke hai
func (w *WorkerService) ProcessTask(task Task, result *TaskResult) error {
	switch task.Operation {
	case "add":
		res, err := addMatrices(task.MatrixA, task.MatrixB) //addMatrix ka alag function banaya huwa
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Result = res
		}
	case "multiply":
		res, err := multiplyMatrices(task.MatrixA, task.MatrixB) // calling the muliply matrix function
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Result = res
		}
	case "transpose":
		// For transpose, we only use MatrixA.
		res := transposeMatrix(task.MatrixA)
		result.Result = res
	default:
		result.Error = "unknown operation"
	}
	return nil
}

// addMatrices returns the element-wise sum of two matrices.
func addMatrices(a, b Matrix) (Matrix, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("empty matrix")
	}
	if len(a) != len(b) || len(a[0]) != len(b[0]) {
		return nil, errors.New("matrices dimensions do not match for addition")
	}
	rows := len(a)
	cols := len(a[0])
	result := make(Matrix, rows)
	for i := 0; i < rows; i++ {
		result[i] = make([]int, cols)
		for j := 0; j < cols; j++ {
			result[i][j] = a[i][j] + b[i][j]
		}
	}
	return result, nil
}

// multiplyMatrices returns the product of two matrices.
func multiplyMatrices(a, b Matrix) (Matrix, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("empty matrix")
	}
	if len(a[0]) != len(b) {
		return nil, errors.New("matrices dimensions do not match for multiplication")
	}
	rows := len(a)
	cols := len(b[0])
	common := len(b)
	result := make(Matrix, rows)
	for i := 0; i < rows; i++ {
		result[i] = make([]int, cols)
		for j := 0; j < cols; j++ {
			sum := 0
			for k := 0; k < common; k++ {
				sum += a[i][k] * b[k][j]
			}
			result[i][j] = sum
		}
	}
	return result, nil
}

// transposeMatrix returns the transpose of the given matrix.
func transposeMatrix(a Matrix) Matrix {
	if len(a) == 0 {
		return Matrix{}
	}
	rows := len(a)
	cols := len(a[0])
	result := make(Matrix, cols)
	for i := 0; i < cols; i++ {
		result[i] = make([]int, rows)
		for j := 0; j < rows; j++ {
			result[i][j] = a[j][i]
		}
	}
	return result
}

// main function initializes and starts the worker server
func main() {
	// Command-line flags.
	portPtr := flag.String("port", "9001", "Port for worker to listen on") //default port is 9001 for worker 1 and 9002 for worker 2 and so on
	certFile := flag.String("cert", "server.crt", "Path to TLS certificate file")
	keyFile := flag.String("key", "server.key", "Path to TLS key file")
	flag.Parse()

	//Create an instance of the worker service that will handle RPC requests, basically initializing the RPC function
	workerService := new(WorkerService)
	err := rpc.Register(workerService)
	if err != nil {
		fmt.Println("Error registering RPC service:", err)
		os.Exit(1)
	}

	// Load the TLS certificate
	cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
	if err != nil {
		fmt.Println("Error loading TLS certificate/key:", err)
		os.Exit(1)
	}

	//config TLS wali setting to fulfill the 2 marks bonus
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Listen with TLS.
	listener, err := tls.Listen("tcp", ":"+*portPtr, tlsConfig)
	if err != nil {
		fmt.Println("Listener error:", err)
		os.Exit(1)
	}
	fmt.Println("Worker (TLS) listening on port", *portPtr)

	// Accept incoming connections and process them asynchronously
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection accept error:", err)
			continue
		}

		// Handle each connection in a separate goroutine (thread) so multiple requests can be processed at once
		go rpc.ServeConn(conn)
	}
}
