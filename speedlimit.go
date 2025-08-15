package connware

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type DataSize int

const (
	B  DataSize = 1
	KB DataSize = 1024 * B
	MB DataSize = 1024 * KB
	GB DataSize = 1024 * MB
)

type SpeedLimitConn struct {
	io.ReadWriteCloser
	readLimit  *rate.Limiter
	writeLimit *rate.Limiter
	ctx        context.Context
}

func NewSpeedLimitConn(ctx context.Context, conn io.ReadWriteCloser) *SpeedLimitConn {
	return &SpeedLimitConn{
		ReadWriteCloser: conn,
		ctx:             ctx,
	}
}

func (slc *SpeedLimitConn) Read(p []byte) (n int, err error) {
	l := len(p)
	if slc.readLimit != nil {
		l = min(len(p), slc.readLimit.Burst())
	}
	err = slc.readLimit.WaitN(slc.ctx, l)
	if err != nil {
		return
	}
	return slc.ReadWriteCloser.Read(p[:l])
}

func (slc *SpeedLimitConn) Write(p []byte) (n int, err error) {
	l := len(p)
	if slc.writeLimit != nil {
		l = min(len(p), slc.writeLimit.Burst())
	}
	err = slc.writeLimit.WaitN(slc.ctx, l)
	if err != nil {
		return
	}
	return slc.ReadWriteCloser.Write(p)
}

type SpeedLimitOption func(*SpeedLimitConn)

func WithReadLimit(size DataSize) SpeedLimitOption {
	return func(slc *SpeedLimitConn) {
		slc.readLimit = rate.NewLimiter(rate.Limit(size)-1, int(size))
	}
}

func WithWriteLimit(size DataSize) SpeedLimitOption {
	return func(slc *SpeedLimitConn) {
		slc.writeLimit = rate.NewLimiter(rate.Limit(size)-1, int(size))
	}
}

func SpeedLimitMiddleware(opts ...SpeedLimitOption) Middleware {
	return func(ctx context.Context, conn io.ReadWriteCloser) io.ReadWriteCloser {
		slc := &SpeedLimitConn{
			ReadWriteCloser: conn,
			ctx:             ctx,
		}

		for _, opt := range opts {
			opt(slc)
		}

		if slc.readLimit != nil || slc.writeLimit != nil {
			return slc
		}
		return conn
	}
}
