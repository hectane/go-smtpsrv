## go-smtpsrv

[![Build Status](https://travis-ci.org/hectane/go-smtpsrv.svg?branch=master)](https://travis-ci.org/hectane/go-smtpsrv)
[![GoDoc](https://godoc.org/github.com/hectane/go-smtpsrv?status.svg)](https://godoc.org/github.com/hectane/go-smtpsrv)
[![MIT License](http://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](http://opensource.org/licenses/MIT)

Golang provides an SMTP _client_ implementation in the [`net/smtp`](https://golang.org/pkg/net/smtp/) package, but it lacks an implementation of an SMTP server. This package is based on [RFC 5321](https://tools.ietf.org/html/rfc5321) and attempts to bridge that gap.

### Using go-smtpsrv

To use the package in your project, import the following package:

    import "github.com/hectane/go-smtpsrv"

To begin receiving SMTP connections from clients, create an instance of `smtpsrv.Server`:

    s, err := smtpsrv.NewServer(&smtpsrv.Config{
        Addr: ":smtp",
        Banner: "SuperAwesomeServer",
        ReadTimeout: 2 * time.Minute,
    })

The banner is used to greet clients and the read timeout determines how long the server will wait for the client to send a command before timing out and disconnecting them.

The server provides a channel that must be used for receiving messages:

    go func() {
        for m := range s.NewMessage {
            // do something with the message
        }
    }()

To close the server and wait for it to shut down:

    s.Close(false)

To shut down immediately and forcefully disconnect all clients without allowing them to finish, use `true` for the parameter passed to `Close()`.
