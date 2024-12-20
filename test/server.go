package test

import (
	"bytes"
	"container/ring"
	"context"
	"io"
	"log"
	"net"
	"net/textproto"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-nntp"
	nntpserver "github.com/dustin/go-nntp/server"
)

const maxArticles = 100

type articleRef struct {
	msgid string
	num   int64
}

type groupStorage struct {
	group *nntp.Group
	// article refs
	articles *ring.Ring
}

type articleStorage struct {
	headers  textproto.MIMEHeader
	body     string
	refcount int
}

type testBackendType struct {
	// group name -> group storage
	groups map[string]*groupStorage
	// message ID -> article
	articles map[string]*articleStorage
}

func (tb *testBackendType) ListGroups(max int) ([]*nntp.Group, error) {
	rv := []*nntp.Group{}
	for _, g := range tb.groups {
		rv = append(rv, g.group)
	}
	return rv, nil
}

func (tb *testBackendType) GetGroup(name string) (*nntp.Group, error) {
	var group *nntp.Group

	for _, g := range tb.groups {
		if g.group.Name == name {
			group = g.group
			break
		}
	}

	if group == nil {
		return nil, nntpserver.ErrNoSuchGroup
	}

	return group, nil
}

func mkArticle(a *articleStorage) *nntp.Article {
	return &nntp.Article{
		Header: a.headers,
		Body:   strings.NewReader(a.body),
		Bytes:  len(a.body),
		Lines:  strings.Count(a.body, "\n"),
	}
}

func findInRing(in *ring.Ring, f func(r interface{}) bool) *ring.Ring {
	if f(in.Value) {
		return in
	}
	for p := in.Next(); p != in; p = p.Next() {
		if f(p.Value) {
			return p
		}
	}
	return nil
}

func (tb *testBackendType) GetArticle(group *nntp.Group, id string) (*nntp.Article, error) {

	msgID := id
	var a *articleStorage

	if intid, err := strconv.ParseInt(id, 10, 64); err == nil {
		msgID = ""
		// by int ID.  Gotta go find it.
		if groupStorage, ok := tb.groups[group.Name]; ok {
			r := findInRing(groupStorage.articles, func(v interface{}) bool {
				if v != nil {
					log.Printf("Looking at %v", v)
				}
				if aref, ok := v.(articleRef); ok && aref.num == intid {
					return true
				}
				return false
			})
			if aref, ok := r.Value.(articleRef); ok {
				msgID = aref.msgid
			}
		}
	}

	a = tb.articles[msgID]
	if a == nil {
		return nil, nntpserver.ErrInvalidMessageID
	}

	return mkArticle(a), nil
}

// Because I suck at ring, I'm going to just post-sort these.
type nalist []nntpserver.NumberedArticle

func (n nalist) Len() int {
	return len(n)
}

func (n nalist) Less(i, j int) bool {
	return n[i].Num < n[j].Num
}

func (n nalist) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (tb *testBackendType) GetArticles(group *nntp.Group,
	from, to int64) ([]nntpserver.NumberedArticle, error) {

	gs, ok := tb.groups[group.Name]
	if !ok {
		return nil, nntpserver.ErrNoSuchGroup
	}

	log.Printf("Getting articles from %d to %d", from, to)

	rv := []nntpserver.NumberedArticle{}
	gs.articles.Do(func(v interface{}) {
		if v != nil {
			if aref, ok := v.(articleRef); ok {
				if aref.num >= from && aref.num <= to {
					a, ok := tb.articles[aref.msgid]
					if ok {
						article := mkArticle(a)
						rv = append(rv,
							nntpserver.NumberedArticle{
								Num:     aref.num,
								Article: article})
					}
				}
			}
		}
	})

	sort.Sort(nalist(rv))

	return rv, nil
}

func (tb *testBackendType) AllowPost() bool {
	return true
}

func (tb *testBackendType) decr(msgid string) {
	if a, ok := tb.articles[msgid]; ok {
		a.refcount--
		if a.refcount == 0 {
			log.Printf("Getting rid of %v", msgid)
			delete(tb.articles, msgid)
		}
	}
}

func (tb *testBackendType) Post(article *nntp.Article) error {
	log.Printf("Got headers: %#v", article.Header)
	b := []byte{}
	buf := bytes.NewBuffer(b)
	n, err := io.Copy(buf, article.Body)
	if err != nil {
		return err
	}
	log.Printf("Read %d bytes of body", n)

	a := articleStorage{
		headers:  article.Header,
		body:     buf.String(),
		refcount: 0,
	}

	msgID := a.headers.Get("Message-Id")

	if _, ok := tb.articles[msgID]; ok {
		return nntpserver.ErrPostingFailed
	}

	for _, g := range article.Header["Newsgroups"] {
		if g, ok := tb.groups[g]; ok {
			g.articles = g.articles.Next()
			if g.articles.Value != nil {
				aref := g.articles.Value.(articleRef)
				tb.decr(aref.msgid)
			}
			if g.articles.Value != nil || g.group.Low == 0 {
				g.group.Low++
			}
			g.group.High++
			g.articles.Value = articleRef{
				msgID,
				g.group.High,
			}
			log.Printf("Placed %v", g.articles.Value)
			a.refcount++
			g.group.Count = int64(g.articles.Len())

			log.Printf("Stored %v in %v", msgID, g.group.Name)
		}
	}

	if a.refcount > 0 {
		tb.articles[msgID] = &a
	} else {
		return nntpserver.ErrPostingFailed
	}

	return nil
}

func (tb *testBackendType) Authorized() bool {
	return true
}

func (tb *testBackendType) Authenticate(user, pass string) (nntpserver.Backend, error) {
	return nil, nntpserver.ErrAuthRejected
}

func maybefatal(err error, f string, a ...interface{}) {
	if err != nil {
		log.Fatalf(f, a...)
	}
}

type Server struct {
	l net.Listener
	s *nntpserver.Server
}

func NewServer() (*Server, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return nil, err
	}

	testBackend := testBackendType{
		groups:   map[string]*groupStorage{},
		articles: map[string]*articleStorage{},
	}

	testBackend.groups["alt.test"] = &groupStorage{
		group: &nntp.Group{
			Name:        "alt.test",
			Description: "A test.",
			Posting:     nntp.PostingNotPermitted},
		articles: ring.New(maxArticles),
	}

	testBackend.groups["misc.test"] = &groupStorage{
		group: &nntp.Group{
			Name:        "misc.test",
			Description: "More testing.",
			Posting:     nntp.PostingPermitted},
		articles: ring.New(maxArticles),
	}

	s := nntpserver.NewServer(&testBackend)

	return &Server{
		l: l,
		s: s,
	}, nil
}

func (s *Server) Serve(ctx context.Context) {
	defer s.l.Close()
	for {
		select {
		case <-ctx.Done():
			s.l.Close()
			return
		default:
			c, err := s.l.Accept()
			maybefatal(err, "Error accepting connection: %v", err)
			go s.s.Process(c)
		}
	}
}

func (s *Server) Port() int {
	return s.l.Addr().(*net.TCPAddr).Port
}
