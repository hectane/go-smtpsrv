## go-smtp

Golang provides an SMTP _client_ implementation in the [`net/smtp`](https://golang.org/pkg/net/smtp/) package, but it lacks an implementation of an SMTP server. This package is based on [RFC 5321](https://tools.ietf.org/html/rfc5321) and makes heavy use of the [`net/textproto`](https://golang.org/pkg/net/textproto/) package.

**Note:** this package is a work in progress and not suitable for production use.
