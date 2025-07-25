//go:generate .tools/mockgen -source=./connection.go -destination=./connection_mock.go -package=nntpcli Connection
package nntpcli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/mnightingale/rapidyenc"
)

type Connection interface {
	io.Closer
	Authenticate(username, password string) (err error)
	JoinGroup(name string) error
	BodyDecoded(msgID string, w io.Writer, discard int64) (int64, error)
	BodyReader(msgID string) (io.ReadCloser, error)
	Post(r io.Reader) error
	CurrentJoinedGroup() string
	MaxAgeTime() time.Time
	Stat(msgID string) (int, error)
	Capabilities() ([]string, error)
}

type YencHeaders struct {
	FileName   string
	FileSize   int64
	PartNumber int64
	TotalParts int64
	Offset     int64
	PartSize   int64
	Hash       uint32
}

type ArticleBodyReader struct {
	decoder       *rapidyenc.Decoder
	conn          *connection
	responseID    uint
	buffer        *bytes.Buffer
	headersRead   bool
	yencHeaders   *YencHeaders
	closed        bool
}

func (r *ArticleBodyReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, io.EOF
	}

	if r.buffer != nil && r.buffer.Len() > 0 {
		n, err = r.buffer.Read(p)
		if r.buffer.Len() == 0 {
			r.buffer = nil
		}
		if n > 0 || err != nil {
			return n, err
		}
	}

	return r.decoder.Read(p)
}

func (r *ArticleBodyReader) GetYencHeaders() *YencHeaders {
	if r.yencHeaders != nil {
		return r.yencHeaders
	}

	if !r.headersRead {
		buf := make([]byte, 4096)
		n, _ := r.decoder.Read(buf)
		if n > 0 {
			r.buffer = bytes.NewBuffer(buf[:n])
		}
		r.headersRead = true
	}

	r.yencHeaders = &YencHeaders{
		FileName:   r.decoder.Meta.FileName,
		FileSize:   r.decoder.Meta.FileSize,
		PartNumber: r.decoder.Meta.PartNumber,
		TotalParts: r.decoder.Meta.TotalParts,
		Offset:     r.decoder.Meta.Offset,
		PartSize:   r.decoder.Meta.PartSize,
		Hash:       r.decoder.Meta.Hash,
	}

	return r.yencHeaders
}

func (r *ArticleBodyReader) Close() error {
	if r.closed {
		return nil
	}

	r.closed = true

	if r.decoder != nil {
		_, _ = io.Copy(io.Discard, r.decoder)
		rapidyenc.ReleaseDecoder(r.decoder)
		r.decoder = nil
	}

	if r.conn != nil {
		r.conn.conn.EndResponse(r.responseID)
	}

	return nil
}

type connection struct {
	maxAgeTime         time.Time
	netconn            net.Conn
	conn               *textproto.Conn
	currentJoinedGroup string
}

func newConnection(netconn net.Conn, maxAgeTime time.Time) (Connection, error) {
	conn := textproto.NewConn(netconn)

	_, _, err := conn.ReadCodeLine(200)
	if err != nil {
		// Download only server
		_, _, err = conn.ReadCodeLine(201)
		if err == nil {
			return &connection{
				conn:       conn,
				netconn:    netconn,
				maxAgeTime: maxAgeTime,
			}, nil
		}

		conn.Close()

		return nil, err
	}

	return &connection{
		conn:       conn,
		netconn:    netconn,
		maxAgeTime: maxAgeTime,
	}, nil
}

// Close this client.
func (c *connection) Close() error {
	_, _, err := c.sendCmd(205, "QUIT")
	e := c.conn.Close()

	if err == nil {
		return err
	}

	return e
}

// Authenticate against an NNTP server using authinfo user/pass
func (c *connection) Authenticate(username, password string) (err error) {
	code, _, err := c.sendCmd(381, "AUTHINFO USER %s", username)
	if err != nil {
		return err
	}

	switch code {
	case 481, 482, 502:
		// failed, out of sequence or command not available
		return err
	case 281:
		// accepted without password
		return nil
	case 381:
		// need password
		break
	default:
		return err
	}

	_, _, err = c.sendCmd(281, "AUTHINFO PASS %s", password)
	if err != nil {
		return err
	}

	return nil
}

func (c *connection) JoinGroup(group string) error {
	if group == c.currentJoinedGroup {
		return nil
	}

	_, _, err := c.sendCmd(211, "GROUP %s", group)
	if err != nil {
		return err
	}

	c.currentJoinedGroup = group

	return err
}

func (c *connection) CurrentJoinedGroup() string {
	return c.currentJoinedGroup
}

// BodyDecoded gets the decoded body of an article
// If discard is provided the body will be discarded until the discard line,
// this is useful if you don't want to start the writer from the beginning
// Body retrieves the body of a message with the given message ID from the NNTP server,
// writes it to the provided io.Writer, and optionally discards the first 'discard' lines.
//
// Parameters:
//   - msgID: The message ID of the article to retrieve.
//   - w: The io.Writer to which the message body will be written.
//   - discard: The number of lines to discard from the beginning of the message body.
//
// Returns:
//   - int64: The number of bytes written to the io.Writer.
//   - error: Any error encountered during the operation.
//
// The function sends the "BODY" command to the NNTP server, starts the response,
// and reads the response code. It uses a decoder to read the message body and
// optionally discards the specified number of lines before writing the remaining
// body to the provided io.Writer. If an error occurs during reading or writing,
// the function ensures that the decoder is fully read to avoid connection issues.
func (c *connection) BodyDecoded(msgID string, w io.Writer, discard int64) (int64, error) {
	id, err := c.conn.Cmd("BODY <%s>", msgID)
	if err != nil {
		return 0, formatError(err)
	}

	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)

	_, _, err = c.conn.ReadCodeLine(222)
	if err != nil {
		return 0, err
	}

	dec := rapidyenc.AcquireDecoder(c.conn.R)
	defer rapidyenc.ReleaseDecoder(dec)

	// Discard the first n lines
	if discard > 0 {
		if _, err = io.CopyN(io.Discard, dec, discard); err != nil {
			_, _ = io.Copy(io.Discard, dec)

			return 0, err
		}
	}

	n, err := io.Copy(w, dec)
	if err != nil {
		_, _ = io.Copy(io.Discard, dec)

		return n, err
	}

	return n, nil
}

func (c *connection) BodyReader(msgID string) (io.ReadCloser, error) {
	id, err := c.conn.Cmd("BODY <%s>", msgID)
	if err != nil {
		return nil, formatError(err)
	}

	c.conn.StartResponse(id)

	_, _, err = c.conn.ReadCodeLine(222)
	if err != nil {
		c.conn.EndResponse(id)
		return nil, err
	}

	dec := rapidyenc.AcquireDecoder(c.conn.R)

	return &ArticleBodyReader{
		decoder:    dec,
		conn:       c,
		responseID: id,
		closed:     false,
	}, nil
}

// Post a new article
//
// The reader should contain the entire article, headers and body in
// RFC822ish format.
func (c *connection) Post(r io.Reader) error {
	_, _, err := c.sendCmd(340, "POST")
	if err != nil {
		return err
	}

	w := c.conn.DotWriter()

	_, err = io.Copy(w, r)
	if err != nil {
		// This seems really bad
		return err
	}

	w.Close()

	_, _, err = c.conn.ReadCodeLine(240)

	return err
}

const NumberOfStatResParams = 3

// Stat sends a STAT command to the NNTP server to check the status of a message
// with the given message ID. It returns the message number if the message exists.
//
// Parameters:
//
//	msgID - The message ID to check.
//
// Returns:
//
//	int - The message number if the message exists.
//	error - An error if the command fails or the response is invalid.
func (c *connection) Stat(msgID string) (int, error) {
	id, err := c.conn.Cmd("STAT <%s>", msgID)
	if err != nil {
		return 0, err
	}

	c.conn.StartResponse(id)
	defer c.conn.EndResponse(id)

	_, line, err := c.conn.ReadCodeLine(223)
	if err != nil {
		return 0, err
	}

	ss := strings.SplitN(line, " ", NumberOfStatResParams) // optional comment ignored
	if len(ss) < NumberOfStatResParams-1 {
		return 0, fmt.Errorf("bad response to STAT: %s", line)
	}

	number, err := strconv.Atoi(ss[0])
	if err != nil {
		return 0, err
	}

	return number, err
}

func (c *connection) MaxAgeTime() time.Time {
	return c.maxAgeTime
}

// Capabilities returns a list of features this server performs.
// Not all servers support capabilities.
func (c *connection) Capabilities() ([]string, error) {
	_, _, err := c.sendCmd(101, "CAPABILITIES")
	if err != nil {
		return nil, err
	}

	return c.readStrings()
}

// readStrings reads a list of strings from the NNTP connection,
// stopping at a line containing only a . (Convenience method for
// LIST, etc.)
func (c *connection) readStrings() ([]string, error) {
	var sv []string

	for {
		line, err := c.conn.ReadLine()
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(line, "\r\n") {
			line = line[0 : len(line)-2]
		} else if strings.HasSuffix(line, "\n") {
			line = line[0 : len(line)-1]
		}

		if line == "." {
			break
		}

		sv = append(sv, line)
	}

	return sv, nil
}

func (c *connection) sendCmd(expectCode int, cmd string, args ...any) (int, string, error) {
	id, err := c.conn.Cmd(cmd, args...)
	if err != nil {
		return 0, "", err
	}

	c.conn.StartResponse(id)

	defer c.conn.EndResponse(id)

	return c.conn.ReadCodeLine(expectCode)
}

func formatError(err error) error {
	if IsArticleNotFoundError(err) {
		return errors.Join(err, ErrArticleNotFound)
	}

	return err
}
