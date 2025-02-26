// Package nntp provides base NNTP definitions.
package nntpcli

import (
	"fmt"
	"io"
	"net/textproto"
)

// PostingStatus type for groups.
type PostingStatus byte

// PostingStatus values.
const (
	Unknown             = PostingStatus(0)
	PostingPermitted    = PostingStatus('y')
	PostingNotPermitted = PostingStatus('n')
	PostingModerated    = PostingStatus('m')
)

func (ps PostingStatus) String() string {
	return fmt.Sprintf("%c", ps)
}

// Group represents a usenet newsgroup.
type Group struct {
	Name        string
	Description string
	Count       int64
	High        int64
	Low         int64
	Posting     PostingStatus
}

// An Article that may appear in one or more groups.
type Article struct {
	// The article's headers
	Header textproto.MIMEHeader
	// The article's body
	Body io.Reader
	// Number of bytes in the article body (used by OVER/XOVER)
	Bytes int
	// Number of lines in the article body (used by OVER/XOVER)
	Lines int
}

// MessageID provides convenient access to the article's Message ID.
func (a *Article) MessageID() string {
	return a.Header.Get("Message-Id")
}

func (a *Article) String() string {
	id, ok := a.Header["Message-Id"]
	if !ok {
		return "[NNTP article]"
	}

	return fmt.Sprintf("[NNTP article %s]", id[0])
}
