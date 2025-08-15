package connware

import (
	"context"
	"io"
	"time"

	"golang.org/x/time/rate"
)

type DataSize float64

const (
	B  DataSize = 1
	KB DataSize = 1024 * B
	MB DataSize = 1024 * KB
	GB DataSize = 1024 * MB
)

type SpeedLimit struct {
	Amount   DataSize
	Interval time.Duration
}

func (s SpeedLimit) Limit() rate.Limit {
	return rate.Limit(float64(s.Amount) / s.Interval.Seconds())
}

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

func (slc *SpeedLimitConn) SetReadLimit(speed SpeedLimit) *SpeedLimitConn {
	slc.readLimit = rate.NewLimiter(speed.Limit(), int(speed.Amount))
	return slc
}

func (slc *SpeedLimitConn) SetWriteLimit(speed SpeedLimit) *SpeedLimitConn {
	slc.writeLimit = rate.NewLimiter(speed.Limit(), int(speed.Amount))
	return slc
}

func (slc *SpeedLimitConn) Read(p []byte) (n int, err error) {
	if slc.readLimit == nil {
		return slc.ReadWriteCloser.Read(p)
	}

	var totalRead int
	for totalRead < len(p) {
		toRead := len(p) - totalRead
		burst := slc.readLimit.Burst()
		if toRead > burst {
			toRead = burst
		}

		if err := slc.readLimit.WaitN(slc.ctx, toRead); err != nil {
			return totalRead, err
		}

		bytesRead, err := slc.ReadWriteCloser.Read(p[totalRead : totalRead+toRead])
		if bytesRead > 0 {
			totalRead += bytesRead
		}

		if err != nil {
			return totalRead, err
		}
	}
	return totalRead, nil
}

func (slc *SpeedLimitConn) Write(p []byte) (n int, err error) {
	if slc.writeLimit == nil {
		return slc.ReadWriteCloser.Write(p)
	}

	var totalWritten int
	for totalWritten < len(p) {
		toWrite := len(p) - totalWritten
		burst := slc.writeLimit.Burst()
		if toWrite > burst {
			toWrite = burst
		}

		if err := slc.writeLimit.WaitN(slc.ctx, toWrite); err != nil {
			return totalWritten, err
		}

		bytesWritten, err := slc.ReadWriteCloser.Write(p[totalWritten : totalWritten+toWrite])
		if bytesWritten > 0 {
			totalWritten += bytesWritten
		}

		if err != nil {
			return totalWritten, err
		}
	}

	return totalWritten, nil
}

type SpeedLimitOption func(*SpeedLimitConn)

func WithReadLimit(speed SpeedLimit) SpeedLimitOption {
	return func(slc *SpeedLimitConn) {
		slc.readLimit = rate.NewLimiter(speed.Limit(), int(speed.Amount))
	}
}

func WithWriteLimit(speed SpeedLimit) SpeedLimitOption {
	return func(slc *SpeedLimitConn) {
		slc.writeLimit = rate.NewLimiter(speed.Limit(), int(speed.Amount))
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
