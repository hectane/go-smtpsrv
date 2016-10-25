## go-smtpsrv

[![GoDoc](https://godoc.org/github.com/hectane/go-smtpsrv?status.svg)](https://godoc.org/github.com/hectane/go-smtpsrv)
[![MIT License](http://img.shields.io/badge/license-MIT-9370d8.svg?style=flat)](http://opensource.org/licenses/MIT)

Golang provides an SMTP _client_ implementation in the [`net/smtp`](https://golang.org/pkg/net/smtp/) package, but it lacks an implementation of an SMTP server. This package is based on [RFC 5321](https://tools.ietf.org/html/rfc5321) and makes heavy use of [`bufio.Reader`](https://golang.org/pkg/bufio/) package.

**Note:** this package is a _work in progress_ and not suitable for production use. Interfaces and functions are subject to change until the first release is finalized.

### Usage

In order to receive connections, an instance of `Server` must be created and the `Start()` method invoked. By default, the server binds to `:smtp`.

    import "github.com/hectane/go-smtpsrv"

    var server smtpsrv.Server
    err := server.Start()
    if err != nil {
        panic(err)
    }

To shut down the server, the `Stop()` method must be invoked.

    server.Stop()
