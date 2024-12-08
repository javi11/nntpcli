package nntpcli

import (
	"errors"
	"io"
	"net"
	"net/textproto"
	"syscall"

	"golang.org/x/text/transform"
)

var (
	ErrCapabilitiesUnpopulated = errors.New("capabilities unpopulated")
	ErrNoSuchCapability        = errors.New("no such capability")
	ErrNilNttpConn             = errors.New("nil nntp connection")
)

const SegmentAlreadyExistsErrCode = 441
const ToManyConnectionsErrCode = 502
const CanNotJoinGroup = 411
const ArticleNotFoundErrCode = 430

var retirableErrors = []int{
	SegmentAlreadyExistsErrCode,
	ToManyConnectionsErrCode,
	CanNotJoinGroup,
	ArticleNotFoundErrCode,
}

func IsRetryableError(err error) bool {
	if errors.Is(err, ErrNilNttpConn) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, io.ErrShortWrite) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, transform.ErrShortSrc) {
		return true
	}

	var netErr net.Error
	if ok := errors.As(err, &netErr); ok {
		return true
	}

	var protocolErr textproto.ProtocolError
	if ok := errors.As(err, &protocolErr); ok {
		return true
	}

	var nntpErr *textproto.Error
	if ok := errors.As(err, &nntpErr); ok {
		for _, r := range retirableErrors {
			if nntpErr.Code == r {
				return true
			}
		}
	}

	return false
}

func IsArticleNotFoundError(err error) bool {
	var nntpErr *textproto.Error
	if ok := errors.As(err, &nntpErr); ok {
		return nntpErr.Code == 430
	}

	return false
}
