// Client provides an API client to the petstore service.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/server/storage"

	"google.golang.org/grpc"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/proto"
)

// Client is a client to the petstore service.
type Client struct {
	client pb.PetStoreClient
	conn   *grpc.ClientConn
}

// New is the constructor for Client. addr is the server's [host]:[port].
func New(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	return &Client{
		client: pb.NewPetStoreClient(conn),
		conn:   conn,
	}, nil
}

// Pet is a wrapper around a *pb.Pet that can return Go versions of
// fields and errors if the returned stream has an error.
type Pet struct {
	*pb.Pet
	err error
}

// Proto will give the Pet's proto representation.
func (p Pet) Proto() *pb.Pet {
	return p.Pet
}

// Birthday returns the Pet's birthday as a time.Time.
func (p Pet) Birthday() time.Time {
	// We are ignoring the error as we will either get a zero time
	// anyways and the server should be preventing this problem.
	t, _ := storage.BirthdayToTime(p.Pet.Birthday)
	return t
}

// Error indicates if there was an error in the Pet output stream.
func (p Pet) Error() error {
	return p.err
}

// AddPets adds pets to the service and returns their unique identities in the
// same order as being added.
func (c *Client) AddPets(ctx context.Context, pets []*pb.Pet) ([]string, error) {
	if len(pets) == 0 {
		return nil, nil
	}

	for _, p := range pets {
		if err := storage.ValidatePet(p); err != nil {
			return nil, err
		}
	}

	resp, err := c.client.AddPets(ctx, &pb.AddPetsReq{Pets: pets})
	if err != nil {
		return nil, err
	}
	return resp.Ids, nil
}

// DeletePets deletes pets with the IDs passed. If the ID doesn't exist, the
// system ignores it.
func (c *Client) DeletePets(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	_, err := c.client.DeletePets(ctx, &pb.DeletePetsReq{Ids: ids})
	if err != nil {
		return err
	}
	return nil
}

// SearchPets searches the pet store for pets matching the filter. If the filter contains
// no entries, then all pets will be returned.
func (c *Client) SearchPets(ctx context.Context, filter *pb.SearchPetsReq) (chan Pet, error) {
	if filter == nil {
		return nil, fmt.Errorf("the filter cannot be nil")
	}

	stream, err := c.client.SearchPets(ctx, filter)
	if err != nil {
		return nil, err
	}
	ch := make(chan Pet, 1)
	go func() {
		defer close(ch)
		for {
			p, err := stream.Recv()
			if err == io.EOF {
				close(ch)
				return
			}
			if err != nil {
				ch <- Pet{err: err}
				return
			}
			ch <- Pet{Pet: p}
		}
	}()
	return ch, nil
}
