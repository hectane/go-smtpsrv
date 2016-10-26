package smtpsrv

import (
	"net"
)

// Server accepts incoming SMTP connections and hands them off to Proto
// instances for processing.
type Server struct {
	// Receives new messages from clients
	NewMessage <-chan *Message
	newMessage chan *Message
	closed     chan bool
	config     *Config
	listener   net.Listener
}

// run implements the main loop for the server.
func (s *Server) run() {
	for {
		c, err := s.listener.Accept()
		if err != nil {
			break
		} else {
			NewProto(s.config, s.newMessage, c)
		}
	}
	s.closed <- true
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
			NewMessage: newMessage,
			newMessage: newMessage,
			closed:     make(chan bool),
			config:     config,
			listener:   l,
		}
	)
	go s.run()
	return s, nil
}

// Close shuts down the server.
func (s *Server) Close() {
	s.listener.Close()
	<-s.closed
}
