package smtpsrv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"
	"time"
)

var (
	banner = "Banner"
	config = &Config{
		Banner:      banner,
		ReadTimeout: 100 * time.Millisecond,
	}

	// Sample data
	fromEmail = "a@localhost"
	toEmail1  = "b@localhost"
	toEmail2  = "c@localhost"
	content   = "this\r\nis\r\na\r\ntest"

	// Sample commands to send
	cHELO  = "HELO\r\n"
	cMAIL  = "MAIL FROM:" + fromEmail + "\r\n"
	cRCPT1 = "RCPT TO:" + toEmail1 + "\r\n"
	cRCPT2 = "RCPT TO:" + toEmail2 + "\r\n"
	cDATA  = "DATA\r\n" + content + "\r\n.\r\n"
	cRSET  = "RSET\r\n"
	cNOOP  = "NOOP\r\n"
	cQUIT  = "QUIT\r\n"

	// Sample data received
	rBanner = "220 " + banner + " [go-smtpsrv]\r\n"
	rOk     = "250 ok\r\n"
	rQuit   = "221 bye\r\n"
)

// buffersToConn converts two buffers into a net.Conn.
type buffersToConn struct {
	r io.Reader
	w io.Writer
}

func (b buffersToConn) Read(p []byte) (int, error)       { return b.r.Read(p) }
func (b buffersToConn) Write(p []byte) (int, error)      { return b.w.Write(p) }
func (buffersToConn) Close() error                       { return nil }
func (buffersToConn) LocalAddr() net.Addr                { return nil }
func (buffersToConn) RemoteAddr() net.Addr               { return nil }
func (buffersToConn) SetDeadline(t time.Time) error      { return nil }
func (buffersToConn) SetReadDeadline(t time.Time) error  { return nil }
func (buffersToConn) SetWriteDeadline(t time.Time) error { return nil }

// testReponse is used to verify the response from the server for a sequence of
// commands. The first parameter is the input which will be fed into the
// client. The second parameter is the expected output. The third parameter is
// an optional message expected from the newMessage channel.
func testResponse(input, exOutput []byte, message *Message) error {
	var (
		newMessage = make(chan *Message, 1)
		finished   = make(chan *Client)
		inBuffer   = bytes.NewBuffer(input)
		outBuffer  = &bytes.Buffer{}
		_          = NewClient(config, newMessage, finished, buffersToConn{inBuffer, outBuffer})
	)
	select {
	case <-finished:
	case <-time.After(1000 * time.Millisecond):
		return errors.New("timeout exceeded")
	}
	if !bytes.Equal(outBuffer.Bytes(), exOutput) {
		return fmt.Errorf("%s != %s", outBuffer.String(), exOutput)
	}
	select {
	case m := <-newMessage:
		if !reflect.DeepEqual(m, message) {
			return fmt.Errorf("%t != %t", m, message)
		}
	default:
		if message != nil {
			return errors.New("message expected")
		}
	}
	return nil
}

func TestReset(t *testing.T) {
	if err := testResponse(
		[]byte(cMAIL+cRSET+cMAIL+cQUIT),
		[]byte(rBanner+rOk+rOk+rOk+rQuit),
		nil,
	); err != nil {
		t.Fatal(err)
	}
}
