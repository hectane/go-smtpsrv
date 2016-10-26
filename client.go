package smtpsrv

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

// Client facilitates communication with an SMTP client. Each instance
// maintains state for and receives commands from a single client.
type Client struct {
	config     *Config
	conn       net.Conn
	reader     *bufio.Reader
	newMessage chan<- *Message
	finished   chan<- *Client
	mailFrom   string
	mailTo     []string
}

// reset initializes all values to their defaults.
func (c *Client) reset() {
	c.mailFrom = ""
	c.mailTo = []string{}
}

// readLine obtains the next line from the client while observing the timeout.
func (c *Client) readLine() ([]byte, error) {
	if c.config.ReadTimeout != 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout))
	}
	line, isPrefix, err := c.reader.ReadLine()
	if err != nil || isPrefix {
		return nil, err
	}
	return line, nil
}

// writeReply contructs a reply from the reply code and message. The result is
// then sent back to the client.
func (c *Client) writeReply(code int, message string) {
	c.conn.Write([]byte(strconv.Itoa(code) + " " + message + "\r\n"))
}

// writeBanner sends the initial greeting to the client. The banner supplied by
// the caller is combined with the name of this library.
func (c *Client) writeBanner() {
	c.writeReply(220, fmt.Sprintf("%s [go-smtpsrv]", c.config.Banner))
}

// processHELO responds to HELO or EHLO commands from the client. At this
// point, no extensions are supported, so the reply to both commands are
// identical. The banner used in the greeting is repeated here.
func (c *Client) processHELO() {
	c.reset()
	c.writeReply(250, c.config.Banner)
}

// processMail is invoked with the address the email is being sent *from*. This
// address might be used to indicate a failure if the message could not be sent
// for some reason.
func (c *Client) processMAIL(b []byte) {
	// Ensure that this hasn't already been invoked
	if len(c.mailFrom) != 0 {
		c.writeReply(503, "MAIL already invoked")
		return
	}
	// The next five bytes must be "FROM:"
	if !bytes.HasPrefix(bytes.ToUpper(b), []byte("FROM:")) {
		c.writeReply(501, "syntax: \"MAIL FROM:<address>\"")
		return
	}
	// Validate the address
	a, err := mail.ParseAddress(string(b[5:]))
	if err != nil {
		c.writeReply(501, err.Error())
		return
	}
	c.mailFrom = a.Address
	c.writeReply(250, "ok")
}

// processRCPT is invoked one or more times to specify the recipient(s) of the
// message. It may only be invoked *after* MAIL.
func (c *Client) processRCPT(b []byte) {
	// Ensure that MAIL has been invoked
	if len(c.mailFrom) == 0 {
		c.writeReply(503, "MAIL must be invoked first")
		return
	}
	// The next three bytes must be "TO:"
	if !bytes.HasPrefix(bytes.ToUpper(b), []byte("TO:")) {
		c.writeReply(501, "syntax: \"RCPT TO:<address>\"")
		return
	}
	// Validate the address
	a, err := mail.ParseAddress(string(b[3:]))
	if err != nil {
		c.writeReply(501, err.Error())
	}
	c.mailTo = append(c.mailTo, a.Address)
	c.writeReply(250, "ok")
}

// processDATA indicates that what follows is the message body
func (c *Client) processDATA() {
	// Ensure that there is at least one valid "to" address
	if len(c.mailTo) == 0 {
		c.writeReply(503, "RCPT must be invoked first")
		return
	}
	// Continue to read one line at a time until the "CRLF.CRLF" sequence is
	// found - put another way, continue until a line with only "." is
	// encountered
	c.writeReply(354, "continue until \\r\\n.\\r\\n")
	lines := []string{}
	for {
		l, err := c.readLine()
		if err != nil {
			break
		}
		// Check for end-of-transmission and send message if found
		if bytes.Equal(l, []byte(".")) {
			c.newMessage <- &Message{
				From: c.mailFrom,
				To:   c.mailTo,
				Body: strings.Join(lines, "\r\n"),
			}
			c.reset()
			c.writeReply(250, "message queued for delivery")
			break
		}
		lines = append(lines, string(l))
	}
}

// processRSET resets all of the state variables to their initial values.
func (c *Client) processRSET() {
	c.reset()
	c.writeReply(250, "ok")
}

// processNOOP does absolutely nothing.
func (c *Client) processNOOP() {
	c.writeReply(250, "ok")
}

// processQUIT sends a parting message to the client.
func (c *Client) processQUIT() {
	c.writeReply(221, "bye")
}

// run greets the client and processes each of the commands transmitted in
// turn until either the client disconnects or QUIT is issued.
func (c *Client) run() {
	defer func() {
		c.finished <- c
	}()
	c.writeBanner()
	for {
		l, err := c.readLine()
		if err != nil {
			return
		}
		var (
			lineParts = bytes.SplitN(l, []byte(" "), 2)
			cmd       = bytes.ToUpper(bytes.TrimSpace(lineParts[0]))
			param     []byte
		)
		if len(lineParts) > 1 {
			param = lineParts[1]
		}
		switch string(cmd) {
		case "HELO", "EHLO":
			c.processHELO()
		case "MAIL":
			c.processMAIL(param)
		case "RCPT":
			c.processRCPT(param)
		case "DATA":
			c.processDATA()
		case "RSET":
			c.processRSET()
		case "NOOP":
			c.processNOOP()
		case "QUIT":
			c.processQUIT()
			c.conn.Close()
			return
		default:
			c.writeReply(502, "unsupported command")
		}
	}
}

// NewClient creates a new Client instance for interacting with an SMTP client
// using the provided connection.
func NewClient(config *Config, newMessage chan<- *Message, finished chan<- *Client, conn net.Conn) *Client {
	c := &Client{
		config:     config,
		conn:       conn,
		reader:     bufio.NewReader(conn),
		newMessage: newMessage,
		finished:   finished,
		mailTo:     []string{},
	}
	go c.run()
	return c
}

// Close immediately disconnects the socket.
func (c *Client) Close() {
	c.conn.Close()
}
