// Package server contains our gRPC server implementation for the pet store.
package server

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/proto"
	"github.com/PacktPublishing/Go-for-DevOps/chapter/8/petstore/server/storage"
	"github.com/google/uuid"
)

// API implements our gRPC server's API.
type API struct {
	pb.UnimplementedPetStoreServer

	addr  string
	store storage.Data

	grpcServer *grpc.Server
	mu         sync.Mutex
}

// New is the constructore for API.
func New(addr string, store storage.Data) (*API, error) {
	var opts []grpc.ServerOption

	a := &API{
		addr:       addr,
		store:      store,
		grpcServer: grpc.NewServer(opts...),
	}
	a.grpcServer.RegisterService(&pb.PetStore_ServiceDesc, a)

	return a, nil
}

// Start starts the server. This blocks until Stop() is called.
func (a *API) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	lis, err := net.Listen("tcp", a.addr)
	if err != nil {
		return err
	}

	return a.grpcServer.Serve(lis)
}

// Stop stops the server.
func (a *API) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.grpcServer.Stop()
}

// AddPets adds pets to the pet store.
func (a *API) AddPets(ctx context.Context, req *pb.AddPetsReq) (*pb.AddPetsResp, error) {
	for _, p := range req.Pets {
		if err := storage.ValidatePet(p); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		p.Id = uuid.New().String()
	}

	if err := a.store.AddPets(ctx, req.Pets); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.AddPetsResp{}, nil
}

// DeletePets deletes pets from the pet store.
func (a *API) DeletePets(ctx context.Context, req *pb.DeletePetsReq) (*pb.DeletePetsResp, error) {
	if err := a.store.DeletePets(ctx, req.Ids); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.DeletePetsResp{}, nil
}

// SearchPets finds pets in the pet store.
func (a *API) SearchPets(req *pb.SearchPetsReq, stream pb.PetStore_SearchPetsServer) error {
	if err := validateSearch(req); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ch := a.store.SearchPets(stream.Context(), req)
	for item := range ch {
		if item.Error != nil {
			return status.Error(codes.Internal, item.Error.Error())
		}
		if err := stream.Send(item.Pet); err != nil {
			return err
		}
	}
	if stream.Context().Err() != nil {
		return status.Error(codes.DeadlineExceeded, stream.Context().Err().Error())
	}
	return nil
}

func validateSearch(r *pb.SearchPetsReq) error {
	for _, t := range r.Types {
		if t == pb.PetType_PTUnknown {
			return fmt.Errorf("cannot search for PetType_Unkonwn")
		}
	}

	if r.BirthdateRange != nil {
		if r.BirthdateRange.Start == nil {
			return fmt.Errorf("cannot have a BirthdateRange.Start that is nil")
		}
		if r.BirthdateRange.End == nil {
			return fmt.Errorf("cannot have a BirthdateRange.End that is nil")
		}
		if _, err := storage.BirthdayToTime(r.BirthdateRange.Start); err != nil {
			return fmt.Errorf("r.BirthdateRange.Start had error: %s", err)
		}
		if _, err := storage.BirthdayToTime(r.BirthdateRange.End); err != nil {
			return fmt.Errorf("r.BirthdateRange.End had error: %s", err)
		}
	}
	return nil
}
