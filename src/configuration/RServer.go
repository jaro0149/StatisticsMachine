package configuration

import (
	"fmt"
	"github.com/senseyeio/roger"
	"sync"
	"golang.org/x/sync/semaphore"
	"context"
)

// Attribute remotePort uint - listening TCP port (HTTP communication).
// Attribute remoteIpAddress string - IP address on which server resides.
// Attribute sessionsCapacity uint - This variable defines how many session can be concurrently active.
// Attribute rClient *roger.RClient - Instance of the connection to R server. See roger.RClient.
// Attribute semaphore *semaphore.Weighted - Controlling of R sessions allocation process.
// Attribute context *context.Context - Unique semaphore context.
// Attribute sessionsBuffer	*[]SessionStructure - List of allocated sessions.
type RServer struct {
	remoteIpAddress		string
	remotePort			uint
	sessionsCapacity	uint
	rClient 			*roger.RClient
	semaphore			*semaphore.Weighted
	context				*context.Context
	sessionsBuffer		*[]*SessionStructure
}

// Attribute availability bool - Availability state - if the session get be reused by next thread.
// Attribute lock *sync.Mutex - Availability locker. See sync.Mutex.
// Attribute session *roger.Session - Built R session. See roger.Session.
type SessionStructure struct {
	availability	bool
	lock			*sync.Mutex
	session			*roger.Session
}

// Creating of the new R server connection manager.
// Parameter remotePort uint - listening TCP port (HTTP communication).
// Parameter remoteIpAddress string - IP address on which server resides.
// Parameter sessionsCapacity uint - This variable defines how many session can be concurrently active.
// Returning *RServer - object of RServer struct.
func NewRServer(remotePort uint, remoteIpAddress string, sessionsCapacity uint) *RServer {
	semaphore0 := semaphore.NewWeighted(int64(sessionsCapacity))
	context0 := context.TODO()
	rServer := RServer{
		remotePort: remotePort,
		remoteIpAddress: remoteIpAddress,
		sessionsCapacity: sessionsCapacity,
		semaphore: semaphore0,
		context: &context0,
	}
	return &rServer
}

// Starting of R server (Rserve() shell command).
//func (RServer *RServer) StartRServer() {
//	Info.Println("Initilization of R server started.")
//	err := exec.Command( "/bin/sh", "rserve.sh").Run()
//	if err != nil {
//		message := fmt.Sprintf("An error occured during building of R server: %v", err)
//		Error.Panic(message)
//	}
//	Info.Println("R server process has been successfully forked.")
//}

// Connecting to R server as client.
func (RServer *RServer) ConnectToServer() {
	Info.Println("Building of connection to R server.")
	rClient, err01 := roger.NewRClient(RServer.remoteIpAddress, int64(RServer.remotePort))
	if err01 != nil {
		message := fmt.Sprintf("An error occured during connecting to R server: %v", err01)
		Error.Panic(message)
	}
	RServer.rClient = &rClient
	RServer.sessionsBuffer = RServer.buildSessions()
	Info.Println("Connection to R server has been successfully established with prepared sessions.")
}

// Building of R sessions.
func (RServer *RServer) buildSessions() *[]*SessionStructure {
	var sessions []*SessionStructure
	for i := uint(0); i < RServer.sessionsCapacity; i++ {
		session, err := (*RServer.rClient).GetSession()
		if err != nil {
			message := fmt.Sprintf("A new session cannot be created: %v", err)
			Error.Panic(message)
		}
		session.Eval("library(forecast)")
		lock := sync.Mutex{}
		sessionStructure := SessionStructure{
			session: &session,
			availability: true,
			lock: &lock,
		}
		sessions = append(sessions, &sessionStructure)
	}
	return &sessions
}

// Acquiring of a new R session.
// Returning *roger.Session - Acquired R session. See roger.Session.
func (RServer *RServer) GetSession() *roger.Session {
	RServer.semaphore.Acquire(*RServer.context, 1)
	sessionsBuffer := *RServer.sessionsBuffer
	for _, sessionStructure := range sessionsBuffer {
		foundSession := analyseSessionStructure(sessionStructure)
		if foundSession != nil {
			return foundSession
		}
	}
	return nil
}

// Analysis of availability of session.
// Parameter sessionStructure *SessionStructure - input session that is analysed.
// Returning *roger.Session - session if it is free or nil if it is not free.
func analyseSessionStructure(sessionStructure *SessionStructure) *roger.Session {
	sessionStructure.lock.Lock()
	defer sessionStructure.lock.Unlock()
	if sessionStructure.availability {
		sessionStructure.availability = false
		return sessionStructure.session
	}
	return nil
}

// Releasing of leased session.
// Parameter session0 *roger.Session - Acquired session that is going to be released. See roger.session.
func (RServer *RServer) ReleaseSession(session0 *roger.Session) {
	sessionsBuffer := *RServer.sessionsBuffer
	foundEntry := false
	for _, sessionStructure := range sessionsBuffer {
		sessionStructure.lock.Lock()
		if *sessionStructure.session == *session0 {
			sessionStructure.availability = true
			foundEntry = true
		}
		sessionStructure.lock.Unlock()
		if foundEntry {
			RServer.semaphore.Release(1)
			break
		}
	}
}

func (RServer *RServer) CloseAllSessions() {
	Info.Println("Closing of R sessions.")
	if RServer.sessionsBuffer != nil {
		sessionsBuffer := *RServer.sessionsBuffer
		for _, sessionStructure := range sessionsBuffer {
			sessionStructure.lock.Lock()
			(*(sessionStructure.session)).Close()
			sessionStructure.lock.Unlock()
		}
	}
	Info.Println("R sessions have been successfully closed.")
}