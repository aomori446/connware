package connware

import (
	"context"
	"io"
)

type Middleware func(ctx context.Context, conn io.ReadWriteCloser) io.ReadWriteCloser

func Chain(ctx context.Context, conn io.ReadWriteCloser, middlewares ...Middleware) io.ReadWriteCloser {
	for _, m := range middlewares {
		conn = m(ctx, conn)
	}
	return conn
}
