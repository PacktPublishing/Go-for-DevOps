// Client provides an API client to the petstore service.
package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/client/internal/server/storage"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/14/petstore-operator/client/proto"
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
	t, _ := storage.BirthdayToTime(context.Background(), p.Pet.Birthday)
	return t
}

// Error indicates if there was an error in the Pet output stream.
func (p Pet) Error() error {
	return p.err
}

// CallOptions are optional options for an RPC call.
type CallOption func(co *callOptions)

type callOptions struct {
	trace *string
}

// TraceID will cause the RPC call to execute a trace on the service and return "s" to the ID.
// If s == nil, this will ignore the option. If "s" is not set after the call finishes, then
// no trace was made.
func TraceID(s *string) CallOption {
	return func(co *callOptions) {
		if s == nil {
			return
		}
		co.trace = s
	}
}

// AddPets adds pets to the service and returns their unique identities in the
// same order as being added.
func (c *Client) AddPets(ctx context.Context, pets []*pb.Pet, options ...CallOption) ([]string, error) {
	if len(pets) == 0 {
		return nil, nil
	}

	for _, p := range pets {
		if err := storage.ValidatePet(ctx, p, false); err != nil {
			return nil, err
		}
	}

	var header metadata.MD
	ctx, gOpts, f := handleCallOptions(ctx, &header, options)
	defer f()

	resp, err := c.client.AddPets(ctx, &pb.AddPetsReq{Pets: pets}, gOpts...)
	if err != nil {
		return nil, err
	}
	return resp.Ids, nil
}

// UpdatePets updates pets that already exist in the system.
func (c *Client) UpdatePets(ctx context.Context, pets []*pb.Pet, options ...CallOption) error {
	if len(pets) == 0 {
		return nil
	}

	for _, p := range pets {
		if err := storage.ValidatePet(ctx, p, true); err != nil {
			return err
		}
	}

	var header metadata.MD
	ctx, gOpts, f := handleCallOptions(ctx, &header, options)
	defer f()

	_, err := c.client.UpdatePets(ctx, &pb.UpdatePetsReq{Pets: pets}, gOpts...)
	if err != nil {
		return err
	}
	return nil
}

// DeletePets deletes pets with the IDs passed. If the ID doesn't exist, the
// system ignores it.
func (c *Client) DeletePets(ctx context.Context, ids []string, options ...CallOption) error {
	if len(ids) == 0 {
		return nil
	}

	var header metadata.MD
	ctx, gOpts, f := handleCallOptions(ctx, &header, options)
	defer f()

	_, err := c.client.DeletePets(ctx, &pb.DeletePetsReq{Ids: ids}, gOpts...)
	if err != nil {
		return err
	}
	return nil
}

// SearchPets searches the pet store for pets matching the filter. If the filter contains
// no entries, then all pets will be returned.
func (c *Client) SearchPets(ctx context.Context, filter *pb.SearchPetsReq, options ...CallOption) (chan Pet, error) {
	if filter == nil {
		return nil, fmt.Errorf("the filter cannot be nil")
	}

	var header metadata.MD
	ctx, gOpts, f := handleCallOptions(ctx, &header, options)

	stream, err := c.client.SearchPets(ctx, filter, gOpts...)
	if err != nil {
		return nil, err
	}
	ch := make(chan Pet, 1)
	go func() {
		defer close(ch)
		defer f()

		for {
			p, err := stream.Recv()
			if err == io.EOF {
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

// SamplerType is the type of OTEL sampling to do.
type SamplerType int32

const (
	STUnknown SamplerType = 0
	Never     SamplerType = 1
	Always    SamplerType = 2
	Float     SamplerType = 3
)

var validTypes = map[SamplerType]bool{
	Never:  true,
	Always: true,
	Float:  true,
}

type Sampler struct {
	// Type is the type of sampling to use.
	Type SamplerType
	// Rate is the sampling rate, only used if type is Float.
	Rate float64
}

func (s *Sampler) validate() error {
	if !validTypes[s.Type] {
		return fmt.Errorf("type %v is not a supported type", s.Type)
	}
	if s.Type == Float {
		if s.Rate <= 0 || s.Rate > 1 {
			return fmt.Errorf("Rate must be > 0 && <= 1.0, was %v", s.Rate)
		}
	}
	return nil
}

func (s *Sampler) proto() *pb.Sampler {
	return &pb.Sampler{
		Type:       pb.SamplerType(s.Type),
		FloatValue: s.Rate,
	}
}

func (s *Sampler) fromProto(p *pb.Sampler) {
	s.Type = SamplerType(p.Type)
	s.Rate = p.FloatValue
}

// ChangeSampler changes the sampling type and rate on the server. This is
// and admin function that in production should be restricted.
func (c *Client) ChangeSampler(ctx context.Context, sc Sampler, options ...CallOption) error {
	if err := sc.validate(); err != nil {
		return err
	}

	var header metadata.MD
	ctx, gOpts, f := handleCallOptions(ctx, &header, options)
	defer f()

	_, err := c.client.ChangeSampler(ctx, &pb.ChangeSamplerReq{Sampler: sc.proto()}, gOpts...)
	if err != nil {
		return err
	}
	return nil
}

func handleCallOptions(ctx context.Context, header *metadata.MD, options []CallOption) (context.Context, []grpc.CallOption, func()) {
	opts := callOptions{}
	for _, o := range options {
		o(&opts)
	}
	var gOpts []grpc.CallOption

	if opts.trace != nil {
		(*header)["trace"] = nil
		gOpts = append(gOpts, grpc.Header(header))
	}

	f := func() {
		if opts.trace != nil {
			if len((*header)["otel.traceID"]) != 0 {
				*opts.trace = (*header)["otel.traceID"][0]
			}
		}
	}

	return ctx, gOpts, f
}
