package connware

import (
	"context"
	"io"
	"time"
)

type Activity struct {
	io.ReadWriteCloser
	timeout time.Duration
	timer   *time.Timer
}

func NewActivity(ctx context.Context, conn io.ReadWriteCloser, timeout time.Duration) *Activity {
	a := &Activity{
		ReadWriteCloser: conn,
		timeout:         timeout,
		timer:           time.NewTimer(timeout),
	}
	go a.monitor(ctx)
	return a
}

func (a *Activity) monitor(ctx context.Context) {
	defer a.Close()
	defer a.timer.Stop()

	for {
		select {
		case <-a.timer.C:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (a *Activity) resetTimer() {
	if !a.timer.Stop() {
		select {
		case <-a.timer.C:
		default:
		}
	}
	a.timer.Reset(a.timeout)
}

func (a *Activity) Read(p []byte) (n int, err error) {
	n, err = a.ReadWriteCloser.Read(p)
	if n > 0 {
		a.resetTimer()
	}
	return
}

func (a *Activity) Write(p []byte) (n int, err error) {
	n, err = a.ReadWriteCloser.Write(p)
	if n > 0 {
		a.resetTimer()
	}
	return
}

func ActivityMiddleware(timeout time.Duration) Middleware {
	return func(ctx context.Context, conn io.ReadWriteCloser) io.ReadWriteCloser {
		return NewActivity(ctx, conn, timeout)
	}
}
