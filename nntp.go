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
		dialTimeout *time.Duration,
	) (Connection, error)
	DialTLS(
		ctx context.Context,
		host string,
		port int,
		insecureSSL bool,
		dialTimeout *time.Duration,
	) (Connection, error)
}

type client struct {
	log           *slog.Logger
	keepAliveTime time.Duration
}

// New creates a new NNTP client
//
// If no config is provided, the default config will be used
func New(
	c ...Config,
) Client {
	config := mergeWithDefault(c...)

	return &client{
		log:           config.Logger,
		keepAliveTime: config.KeepAliveTime,
	}
}

// Dial connects to an NNTP server
func (c *client) Dial(
	ctx context.Context,
	host string,
	port int,
	dialTimeout *time.Duration,
) (Connection, error) {
	var d net.Dialer
	if dialTimeout != nil {
		d = net.Dialer{Timeout: *dialTimeout}
	}

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(c.keepAliveTime)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return nil, err
	}

	maxAgeTime := time.Now().Add(c.keepAliveTime)

	return newConnection(conn, maxAgeTime)
}

func (c *client) DialTLS(
	ctx context.Context,
	host string,
	port int,
	insecureSSL bool,
	dialTimeout *time.Duration,
) (Connection, error) {
	var d net.Dialer
	if dialTimeout != nil {
		d = net.Dialer{Timeout: *dialTimeout}
	}

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlive(true)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(c.keepAliveTime)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return nil, err
	}

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: insecureSSL,
	})

	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}

	maxAgeTime := time.Now().Add(c.keepAliveTime)

	return newConnection(tlsConn, maxAgeTime)
}
