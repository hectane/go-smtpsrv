package smtpsrv

import (
	"time"
)

// Config stores configuration for an SMTP server.
type Config struct {
	// Address to listen on for new connections
	Addr string
	// Banner to display to new clients
	Banner string
	// Timeout for calls to Read()
	ReadTimeout time.Duration
}
