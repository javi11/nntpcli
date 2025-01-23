//go:generate mockgen -source=./nntp.go -destination=./nntp_mock.go -package=nntpcli Client
package nntpcli

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"time"
)

type TimeData struct {
	Milliseconds int64
	Bytes        int
}

type Client interface {
	Dial(
		ctx context.Context,
		host string,
		port int,
		maxAgeTime time.Time,
	) (Connection, error)
	DialTLS(
		ctx context.Context,
		host string,
		port int,
		insecureSSL bool,
		maxAgeTime time.Time,
	) (Connection, error)
}

type client struct {
	log     *slog.Logger
	timeout time.Duration
}

func New(
	c ...Config,
) Client {
	config := mergeWithDefault(c...)

	return &client{
		timeout: config.timeout,
		log:     config.log,
	}
}

// Dial connects to an NNTP server
func (c *client) Dial(
	ctx context.Context,
	host string,
	port int,
	maxAgeTime time.Time,
) (Connection, error) {
	var d net.Dialer

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	duration := time.Until(maxAgeTime)

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(duration)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return nil, err
	}

	return newConnection(conn, maxAgeTime)
}

func (c *client) DialTLS(
	ctx context.Context,
	host string,
	port int,
	insecureSSL bool,
	maxAgeTime time.Time,
) (Connection, error) {
	var d net.Dialer

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	duration := time.Until(maxAgeTime)

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(duration)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{ServerName: host, InsecureSkipVerify: insecureSSL})

	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}

	return newConnection(tlsConn, maxAgeTime)
}
