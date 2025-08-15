package connware

import (
	"context"
	"io"
	"sync/atomic"
)

type Statistic struct {
	io.ReadWriteCloser
	Uploaded   atomic.Uint64
	Downloaded atomic.Uint64
}

func (s *Statistic) wrap(conn io.ReadWriteCloser) *Statistic {
	s.ReadWriteCloser = conn
	return s
}

func (s *Statistic) Write(p []byte) (n int, err error) {
	n, err = s.ReadWriteCloser.Write(p)
	if n > 0 {
		s.Uploaded.Add(uint64(n))
	}
	return
}

func (s *Statistic) Read(p []byte) (n int, err error) {
	n, err = s.ReadWriteCloser.Read(p)
	if n > 0 {
		s.Downloaded.Add(uint64(n))
	}
	return
}

func (s *Statistic) Total() uint64 {
	return s.Uploaded.Load() + s.Downloaded.Load()
}

func StatisticMiddleware(s *Statistic) Middleware {
	return func(ctx context.Context, conn io.ReadWriteCloser) io.ReadWriteCloser {
		return s.wrap(conn)
	}
}
