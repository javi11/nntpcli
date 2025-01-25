//go:generate .tools/mockgen -source=./nntp.go -destination=./nntp_mock.go -package=nntpcli Client
package nntpcli

import (
	"context"
	"crypto/tls"
	"fmt"
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
		keepAliveTime *time.Duration,
		dialTimeout *time.Duration,
	) (Connection, error)
	DialTLS(
		ctx context.Context,
		host string,
		port int,
		insecureSSL bool,
		keepAliveTime *time.Duration,
		dialTimeout *time.Duration,
	) (Connection, error)
}

type client struct {
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
		keepAliveTime: config.KeepAliveTime,
	}
}

// Dial connects to an NNTP server using a plain TCP connection.
//
// Parameters:
//   - ctx: Context for controlling the connection lifecycle
//   - host: The hostname or IP address of the NNTP server
//   - port: The port number of the NNTP server
//   - keepAliveTime: Optional duration to override the default keep-alive time
//   - dialTimeout: Optional timeout duration for the initial connection
//
// Returns:
//   - Connection: An NNTP connection interface if successful
//   - error: Any error encountered during connection
func (c *client) Dial(
	ctx context.Context,
	host string,
	port int,
	keepAliveTime *time.Duration,
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

	keepAlive := c.keepAliveTime
	if keepAliveTime != nil {
		keepAlive = *keepAliveTime
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
	if err != nil {
		return nil, err
	}

	err = conn.(*net.TCPConn).SetNoDelay(true)
	if err != nil {
		return nil, err
	}

	maxAgeTime := time.Now().Add(keepAlive)

	return newConnection(conn, maxAgeTime)
}

// DialTLS connects to an NNTP server using a TLS-encrypted connection.
//
// Parameters:
//   - ctx: Context for controlling the connection lifecycle
//   - host: The hostname or IP address of the NNTP server
//   - port: The port number of the NNTP server
//   - insecureSSL: If true, skips verification of the server's certificate chain and host name
//   - keepAliveTime: Optional duration to override the default keep-alive time
//   - dialTimeout: Optional timeout duration for the initial connection
//
// Returns:
//   - Connection: An NNTP connection interface if successful
//   - error: Any error encountered during connection
func (c *client) DialTLS(
	ctx context.Context,
	host string,
	port int,
	insecureSSL bool,
	keepAliveTime *time.Duration,
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

	keepAlive := c.keepAliveTime
	if keepAliveTime != nil {
		keepAlive = *keepAliveTime
	}

	err = conn.(*net.TCPConn).SetKeepAlivePeriod(keepAlive)
	if err != nil {
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

	maxAgeTime := time.Now().Add(keepAlive)

	return newConnection(tlsConn, maxAgeTime)
}
