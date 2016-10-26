package smtpsrv

import (
	"errors"
	"fmt"
	"net/smtp"
	"reflect"
	"testing"
	"time"
)

var (
	testEmail1 = "a@localhost"
	testEmail2 = "b@localhost"
	testEmail3 = "c@localhost"
	content    = "this\r\nis\r\na\r\ntest"
	message    = &Message{
		From: testEmail1,
		To: []string{
			testEmail2,
			testEmail3,
		},
		Body: content,
	}
)

func TestResponse(t *testing.T) {
	var (
		m      *Message
		s, err = NewServer(&Config{
			Addr:        "127.0.0.1:0",
			Banner:      "Banner",
			ReadTimeout: 100 * time.Millisecond,
		})
	)
	if err != nil {
		t.Fatal(err)
	}
	// Spawn a goroutine to capture any new message
	go func() {
		m = <-s.NewMessage
	}()
	// Connect to the server using its address
	c, err := smtp.Dial(s.listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// Begin by saying hello
	if err := c.Hello("localhost"); err != nil {
		t.Fatal(err)
	}
	// Now make an out-of-sequence call to RCPT
	if err := c.Rcpt(""); err == nil {
		t.Fatal(errors.New("RCPT should not have succeeded"))
	}
	// Make a correct call to MAIL but with an invalid address
	if err := c.Mail(""); err == nil {
		t.Fatal(errors.New("MAIL should not have accepted malformed address"))
	}
	// Now issue a legit email
	if err := c.Mail(testEmail1); err != nil {
		t.Fatal(err)
	}
	// Call MAIL again (which should fail)
	if err := c.Mail(testEmail1); err == nil {
		t.Fatal(errors.New("MAIL should not be accepted twice"))
	}
	// Reset...
	if err := c.Reset(); err != nil {
		t.Fatal(err)
	}
	// ...and try again
	if err := c.Mail(testEmail1); err != nil {
		t.Fatal(err)
	}
	// DATA should not succeed here
	if _, err := c.Data(); err == nil {
		t.Fatal(errors.New("DATA should not have succeeded"))
	}
	// Send the recipient
	if err := c.Rcpt(testEmail2); err != nil {
		t.Fatal(err)
	}
	// Send the second recipient
	if err := c.Rcpt(testEmail3); err != nil {
		t.Fatal(err)
	}
	// Now send the data
	if w, err := c.Data(); err != nil {
		t.Fatal(err)
	} else {
		w.Write([]byte(content))
		w.Close()
	}
	// Say goodbye...
	if err := c.Quit(); err != nil {
		t.Fatal(err)
	}
	// Shut 'er down
	defer s.Close(false)
	// Ensure a message was received
	if m == nil {
		t.Fatal(errors.New("message expected"))
	}
	// Ensure it matches
	if !reflect.DeepEqual(m, message) {
		t.Fatal(fmt.Errorf("%t != %t", m, message))
	}
}
