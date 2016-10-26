package smtpsrv

import (
	"net"
	"sync"
)

// Server accepts incoming SMTP connections and hands them off to Client
// instances for processing.
type Server struct {
	// Receives new messages from clients
	NewMessage <-chan *Message
	newMessage chan *Message
	finished   chan bool
	config     *Config
	listener   net.Listener

	// Used for synchronizing shutdown - unfortunately, this is all necessary;
	// the list monitors which clients are active so that shutdown can be
	// performed upon request and the mutex guards access to the list
	waitGroup      sync.WaitGroup
	mutex          sync.Mutex
	clients        []*Client
	clientFinished chan *Client
}

// accept listens for new connections from clients. When one connects, a new
// Client instance is created, it is added to the list, and the wait group is
// incremented.
func (s *Server) accept() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			break
		} else {
			c := NewClient(s.config, s.newMessage, s.clientFinished, conn)
			s.waitGroup.Add(1)
			s.mutex.Lock()
			s.clients = append(s.clients, c)
			s.mutex.Unlock()
		}
	}
	s.finished <- true
}

// remove watches for clients that have signalled that they are done and
// removes them from the list of active clients. The wait group is also
// decremented.
func (s *Server) remove() {
	for p := range s.clientFinished {
		s.mutex.Lock()
		for i, v := range s.clients {
			if v == p {
				s.clients = append(s.clients[:i], s.clients[i+1:]...)
				s.waitGroup.Done()
				break
			}
		}
		s.mutex.Unlock()
	}
}

// NewServer creates a new server with the specified configuration.
func NewServer(config *Config) (*Server, error) {
	l, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return nil, err
	}
	var (
		newMessage = make(chan *Message)
		s          = &Server{
			NewMessage:     newMessage,
			newMessage:     newMessage,
			finished:       make(chan bool),
			config:         config,
			listener:       l,
			clientFinished: make(chan *Client),
		}
	)
	go s.accept()
	go s.remove()
	return s, nil
}

// Close shuts down the server and waits for all clients to disconnect. If
// the force parameter is true, clients will be immediately disconnected.
func (s *Server) Close(force bool) {
	s.listener.Close()
	<-s.finished
	if force {
		s.mutex.Lock()
		for _, v := range s.clients {
			v.Close()
		}
		s.mutex.Unlock()
	}
	s.waitGroup.Wait()
	close(s.newMessage)
}
