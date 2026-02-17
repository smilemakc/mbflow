// Package grpcclient provides the gRPC transport implementation for the MBFlow SDK.
package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Conn holds a gRPC client connection.
type Conn struct {
	cc *grpc.ClientConn
}

// Dial establishes a gRPC connection to the given target address.
func Dial(target string) (*Conn, error) {
	cc, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Conn{cc: cc}, nil
}

// Close closes the underlying gRPC connection.
func (c *Conn) Close() error {
	return c.cc.Close()
}
