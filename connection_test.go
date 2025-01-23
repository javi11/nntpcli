package nntpcli

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/javi11/nntpcli/test"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

const examplepost = `From: <nobody@example.com>
Newsgroups: misc.test
Subject: Code test
Message-Id: <1234>
Organization: usenet drive

`

func TestConnection_Body(t *testing.T) {
	conn := articleReadyToDownload(t)

	w := bytes.NewBuffer(nil)

	n, err := conn.BodyDecoded("1234", w, 0)
	assert.NoError(t, err)

	assert.Equal(t, int64(9), n)
	assert.Equal(t, "test text", w.String())
}

func TestConnection_Body_Closed_Before_Full_Read_Drains_The_Buffer(t *testing.T) {
	conn := articleReadyToDownload(t)

	_, w := io.Pipe()
	w.Close()

	n, err := conn.BodyDecoded("1234", w, 0)
	assert.ErrorIs(t, err, io.ErrClosedPipe)

	assert.Equal(t, int64(0), n)

	// The buffer should be drained
	buff := bytes.NewBuffer(nil)

	n, err = conn.BodyDecoded("1234", buff, 0)
	assert.NoError(t, err)

	assert.Equal(t, int64(9), n)
	assert.Equal(t, "test text", buff.String())
}

func TestConnection_Body_Discarding_Bytes(t *testing.T) {
	conn := articleReadyToDownload(t)

	w := bytes.NewBuffer(nil)

	n, err := conn.BodyDecoded("1234", w, 5)
	assert.NoError(t, err)

	// The article is 9 bytes long, so we should get 4 bytes since we discard 5
	assert.Equal(t, int64(4), n)
}

func articleReadyToDownload(t *testing.T) Connection {
	var conn Connection
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())

	s, err := test.NewServer()
	assert.NoError(t, err)

	t.Cleanup(func() {
		cancel()
		s.Close()

		wg.Wait()
	})

	port := s.Port()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve(ctx)
	}()

	var d net.Dialer
	netConn, err := d.DialContext(ctx, "tcp", fmt.Sprintf(":%d", port))
	assert.NoError(t, err)

	conn, err = newConnection(netConn, time.Now().Add(time.Hour))
	assert.NoError(t, err)

	t.Cleanup(func() {
		conn.Close()
	})

	err = conn.JoinGroup("misc.test")
	assert.NoError(t, err)

	buf := bytes.NewBuffer(make([]byte, 0))
	_, err = buf.WriteString(examplepost)
	assert.NoError(t, err)

	encoded, err := os.ReadFile("test/fixtures/test.yenc")
	assert.NoError(t, err)

	_, err = buf.Write(encoded)
	assert.NoError(t, err)

	err = conn.Post(buf)
	assert.NoError(t, err)

	return conn
}
