package configuration

import (
	"fmt"
	"os"
	"os/exec"
	"github.com/senseyeio/roger"
)

// Attribute localHostPort uint - TCP port on which R server is listening for queries.
// Attribute rClient *roger.RClient - Instance of the connection to R server. See roger.RClient.
type RServer struct {
	localHostPort 	uint
	rClient 		*roger.RClient
}

// Creating of the new R server connection manager.
// Returning *RServer - object of RServer struct.
func NewRServer(localHostPort uint) *RServer {
	rServer := RServer{localHostPort: localHostPort}
	return &rServer
}

// Starting of R server (Rserve() command).
func (RServer *RServer) StartRServer() {
	go func() {
		err := exec.Command("/usr/lib/R/bin/R", "CMD", "/usr/local/lib/R/site-library/Rserve/libs//Rserve",
			"--RS-enable-remote").Run()
		if err != nil {
			message := fmt.Sprint(os.Stderr, err)
			Error.Panic(message)
		}
	}()
}

// Connecting to R server as client.
// Returning *roger.RClient - R server connection manager. See roger.RClient.
func (RServer *RServer) ConnectToServer() *roger.RClient {
	rClient, err01 := roger.NewRClient("127.0.0.1", int64(RServer.localHostPort))
	if err01 != nil {
		message := fmt.Sprintf("An error occured during connecting to R server: %v", err01)
		Error.Panic(message)
	}
	_, err02 := rClient.Eval("library(forecast)")
	if err02 != nil {
		message := fmt.Sprintf("An error while enabling of forecast library: %v", err02)
		Error.Panic(message)
	}
	RServer.rClient = &rClient
	return &rClient
}

// Closing of R server connection.
func (RServer *RServer) CloseRServerConnection() {
	RServer.rClient = nil
}