package transport

import (
	"context"
	"net"
)

type connectionKey struct{}

// GetConnection gets the connection from the context.
func GetConnection(ctx context.Context) net.Conn {
	conn, _ := ctx.Value(connectionKey{}).(net.Conn)
	return conn
}

// SetConnection adds the connection to the context to be able to get
// information about the destination ip and port for an incoming RPC. This also
// allows any unary or streaming interceptors to see the connection.
func setConnection(ctx context.Context, conn net.Conn) context.Context {
	return context.WithValue(ctx, connectionKey{}, conn)
}
