package smtpsrv

// Message represents a raw message received from a client.
type Message struct {
	From string
	To   []string
	Body string
}
