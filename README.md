## go-smtp

Golang provides an SMTP _client_ implementation in the [`net/smtp`](https://golang.org/pkg/net/smtp/) package, but it lacks an implementation of an SMTP server. This package is based on [RFC 5321](https://tools.ietf.org/html/rfc5321) and makes heavy use of the [`net/textproto`](https://golang.org/pkg/net/textproto/) package.

**Note:** this package is a _work in progress_ and not suitable for production use. Interfaces and functions are subject to change until the first release is finalized.

### Usage

In order to receive connections, an instance of `Server` must be created and the `Listen()` method invoked. By default, port 25 is used for SMTP.

    import "github.com/hectane/go-smtpsrv"

    var server smtpsrv.Server
    err := server.Listen()
    if err != nil {
        panic(err)
    }

If successful, the `Listen()` call will block until the `Close()` method is invoked. Therefore, the call to `Listen()` and `Close()` must be made from separate goroutines.

This example runs the server until the `SIGINT` signal is received.

    import (
        "github.com/hectane/go-smtpsrv"

        "os"
        "os/signal"
        "syscall"
    )

    var server smtpsrv.Server

    go func() {
        c := make(chan os.Signal)
        signal.Notify(c, syscall.SIGINT)
        <-c
        server.Close()
    }

    err := server.Listen()
    if err != nil {
        panic(err)
    }
