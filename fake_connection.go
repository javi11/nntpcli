package nntpcli

import (
	"io"
	"time"
)

type fakeConnection struct {
	tls          bool
	currentGroup string
}

func NewFakeConnection() Connection {
	return &fakeConnection{
		tls: false,
	}
}

func (c *fakeConnection) CurrentJoinedGroup() string {
	return c.currentGroup
}

func (c *fakeConnection) Authenticate(
	username,
	password string,
) error {
	return nil
}

func (c *fakeConnection) JoinGroup(group string) error {
	c.currentGroup = group
	return nil
}

func (c *fakeConnection) Close() error {
	return nil
}

func (c *fakeConnection) BodyDecoded(msgId string, w io.Writer, discard int64) (int64, error) {
	return 0, nil
}

func (c *fakeConnection) Post(r io.Reader) error {
	return nil
}

func (c *fakeConnection) MaxAgeTime() time.Time {
	return time.Now().Add(1 * time.Hour)
}

func (c *fakeConnection) Stat(msgId string) (int, error) {
	return 0, nil
}

func (c *fakeConnection) Capabilities() ([]string, error) {
	return []string{}, nil
}
