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

// Proto facilitates communication with an SMTP client. Each instance maintains
// state for and receives commands from a single client.
type Proto struct {
	config     *Config
	conn       net.Conn
	reader     *bufio.Reader
	newMessage chan<- *Message
	mailFrom   string
	mailTo     []string
}

// reset initializes all values to their defaults.
func (p *Proto) reset() {
	p.mailFrom = ""
	p.mailTo = []string{}
}

// readLine obtains the next line from the client while observing the timeout.
func (p *Proto) readLine() ([]byte, error) {
	p.conn.SetReadDeadline(time.Now().Add(p.config.ReadTimeout))
	line, isPrefix, err := p.reader.ReadLine()
	if err != nil || isPrefix {
		return nil, err
	}
	return line, nil
}

// writeReply contructs a reply from the reply code and message. The result is
// then sent back to the client.
func (p *Proto) writeReply(code int, message string) {
	p.conn.Write([]byte(strconv.Itoa(code) + " " + message + "\r\n"))
}

// writeBanner sends the initial greeting to the client. The banner supplied by
// the caller is combined with the name of this library.
func (p *Proto) writeBanner() {
	p.writeReply(220, fmt.Sprintf("%s [go-smtpsrv]", p.config.Banner))
}

// processHELO responds to HELO or EHLO commands from the client. At this
// point, no extensions are supported, so the reply to both commands are
// identical. The banner used in the greeting is repeated here.
func (p *Proto) processHELO() {
	p.reset()
	p.writeReply(250, p.config.Banner)
}

// processMail is invoked with the address the email is being sent *from*. This
// address might be used to indicate a failure if the message could not be sent
// for some reason.
func (p *Proto) processMAIL(b []byte) {
	// Ensure that this hasn't already been invoked
	if len(p.mailFrom) != 0 {
		p.writeReply(503, "MAIL already invoked")
		return
	}
	// The next five bytes must be "FROM:"
	if !bytes.HasPrefix(bytes.ToUpper(b), []byte("FROM:")) {
		p.writeReply(501, "syntax: \"MAIL FROM:<address>\"")
		return
	}
	// Validate the address
	a, err := mail.ParseAddress(string(b[5:]))
	if err != nil {
		p.writeReply(501, err.Error())
		return
	}
	p.mailFrom = a.Address
	p.writeReply(250, "ok")
}

// processRCPT is invoked one or more times to specify the recipient(s) of the
// message. It may only be invoked *after* MAIL.
func (p *Proto) processRCPT(b []byte) {
	// Ensure that MAIL has been invoked
	if len(p.mailFrom) == 0 {
		p.writeReply(503, "MAIL must be invoked first")
		return
	}
	// The next three bytes must be "TO:"
	if !bytes.HasPrefix(bytes.ToUpper(b), []byte("TO:")) {
		p.writeReply(501, "syntax: \"RCPT TO:<address>\"")
		return
	}
	// Validate the address
	a, err := mail.ParseAddress(string(b[3:]))
	if err != nil {
		p.writeReply(501, err.Error())
	}
	p.mailTo = append(p.mailTo, a.Address)
	p.writeReply(250, "ok")
}

// processDATA indicates that what follows is the message body
func (p *Proto) processDATA() {
	// Ensure that there is at least one valid "to" address
	if len(p.mailTo) == 0 {
		p.writeReply(503, "RCPT must be invoked first")
		return
	}
	// Continue to read one line at a time until the "CRLF.CRLF" sequence is
	// found - put another way, continue until a line with only "." is
	// encountered
	p.writeReply(354, "continue until \\r\\n.\\r\\n")
	lines := []string{}
	for {
		l, err := p.readLine()
		if err != nil {
			break
		}
		// Check for end-of-transmission and send message if found
		if bytes.Equal(l, []byte(".")) {
			p.newMessage <- &Message{
				From: p.mailFrom,
				To:   p.mailTo,
				Body: strings.Join(lines, "\r\n"),
			}
			p.reset()
			p.writeReply(250, "message queued for delivery")
			break
		}
		lines = append(lines, string(l))
	}
}

// processRSET resets all of the state variables to their initial values.
func (p *Proto) processRSET() {
	p.reset()
	p.writeReply(250, "ok")
}

// processNOOP does absolutely nothing.
func (p *Proto) processNOOP() {
	p.writeReply(250, "ok")
}

// processQUIT sends a parting message to the client.
func (p *Proto) processQUIT() {
	p.writeReply(221, "bye")
}

// run greets the client and processes each of the commands transmitted in
// turn until either the client disconnects or QUIT is issued.
func (p *Proto) run() {
	defer p.conn.Close()
	p.writeBanner()
	for {
		l, err := p.readLine()
		if err != nil {
			break
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
			p.processHELO()
		case "MAIL":
			p.processMAIL(param)
		case "RCPT":
			p.processRCPT(param)
		case "DATA":
			p.processDATA()
		case "RSET":
			p.processRSET()
		case "NOOP":
			p.processNOOP()
		case "QUIT":
			p.processQUIT()
			return
		default:
			p.writeReply(502, "unsupported command")
		}
	}
}

// NewProto creates a new protocol instance for interacting with an SMTP
// client using the provided connection.
func NewProto(config *Config, newMessage chan<- *Message, conn net.Conn) *Proto {
	p := &Proto{
		config:     config,
		conn:       conn,
		reader:     bufio.NewReader(conn),
		newMessage: newMessage,
		mailTo:     []string{},
	}
	go p.run()
	return p
}
