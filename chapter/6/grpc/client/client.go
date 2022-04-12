package client

import (
	"context"
	"time"

	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/6/grpc/proto"
)

// Client is a client to the Quote of the day server.
type Client struct {
	client pb.QOTDClient
	conn   *grpc.ClientConn
}

// New is the constructor for Client. addr is the server's [host]:[port].
func New(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Client{
		client: pb.NewQOTDClient(conn),
		conn:   conn,
	}, nil
}

// QOTD retrieves a quote of the day. If wantAuthor is not set, will randomly choose the author
// of a quote.
func (c *Client) QOTD(ctx context.Context, wantAuthor string) (author, quote string, err error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}

	resp, err := c.client.GetQOTD(ctx, &pb.GetReq{Author: wantAuthor})
	if err != nil {
		return "", "", err
	}
	return resp.Author, resp.Quote, nil
}
