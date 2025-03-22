// client.go
package main

/* the below are packages for the following as they are not taught so let me explain:
TLS encryption and certificate handling (crypto/tls), x509 for certfificates
Command-line flag parsing (flag)
I/O operations fmt
RPC functionality (net / rpc)
System operations (operating systems)
*/
import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
)

type Matrix [][]int //2d represnatated as slice of slices where each inner slice is essentialla row and outer obvioulsy a column

// matrixes and the the command we will send with them of addition, subtraction ya transpose
type Task struct {
	Operation string
	MatrixA   Matrix
	MatrixB   Matrix
}

// the specific answer in them of resultant matrix and any if error found in the form of strings forrobust error chrcking
type TaskResult struct {
	Result Matrix
	Error  string
}

func printMatrix(m Matrix) {
	for _, row := range m {
		fmt.Println(row)
	}
}

func main() {

	serverPtr := flag.String("server", "localhost:8000", "Coordinator server address (host:port)")               // for connection with the server/ cootrdinator
	caCertFile := flag.String("cacert", "server.crt", "Path to CA certificate file (to verify the coordinator)") //specifuy the flag setup fpor certificate (in our case the server.crt) to ensure that TLS	 is implemented
	flag.Parse()

	// tls vertificate setup with error checkinhg

	caCert, err := ioutil.ReadFile(*caCertFile)
	if err != nil {
		fmt.Println("Error reading CA certificate:", err)
		os.Exit(1)
	}
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		fmt.Println("Failed to append CA certificate")
		os.Exit(1)
	}
	// tls configuration
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}
	// this here is rpc or remote procedure call basically the setup for communication with the coordinator over the tcp connection by listening to the poirt we defined on the coordinator (8000) in our case
	conn, err := tls.Dial("tcp", *serverPtr, tlsConfig)
	if err != nil {
		fmt.Println("Error connecting to coordinator:", err)
		os.Exit(1)
	}
	client := rpc.NewClient(conn)
	defer client.Close() // as we are working with 2darrays so resource cleaning is required

	a := Matrix{
		{1, 2},
		{3, 4},
	}
	b := Matrix{
		{5, 6},
		{7, 8},
	}
	// below is simply the tasks of multiplication addition and transpose whwre we simply pass the params for task structure , create a task add struct object called result that we pass as pointer to the seerver using the rpc client.call method

	taskAdd := Task{
		Operation: "add",
		MatrixA:   a,
		MatrixB:   b,
	}
	var result TaskResult
	err = client.Call("CoordinatorService.Compute", taskAdd, &result)
	if err != nil {
		fmt.Println("RPC error:", err)
	} else if result.Error != "" {
		fmt.Println("Task error:", result.Error)
	} else {
		fmt.Println("Addition Result:")
		printMatrix(result.Result)
	}

	////// multiply ///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	taskMultiply := Task{
		Operation: "multiply",
		MatrixA:   a,
		MatrixB:   b,
	}
	err = client.Call("CoordinatorService.Compute", taskMultiply, &result)
	if err != nil {
		fmt.Println("RPC error:", err)
	} else if result.Error != "" {
		fmt.Println("Task error:", result.Error)
	} else {
		fmt.Println("Multiplication Result:")
		printMatrix(result.Result)
	}

	////// tranpose  ////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	taskTranspose := Task{
		Operation: "transpose",
		MatrixA:   a,
	}
	err = client.Call("CoordinatorService.Compute", taskTranspose, &result)
	if err != nil {
		fmt.Println("RPC error:", err)
	} else if result.Error != "" {
		fmt.Println("Task error:", result.Error)
	} else {
		fmt.Println("Transpose Result:")
		printMatrix(result.Result)
	}
}

// go run client.go --server=172.17.18.45:8000 --cacert=server.crt
// ip address will change and cert name too if needed
