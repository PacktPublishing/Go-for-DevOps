// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package lb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// LoadBalancerClient is the client API for LoadBalancer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LoadBalancerClient interface {
	AddPool(ctx context.Context, in *AddPoolReq, opts ...grpc.CallOption) (*AddPoolResp, error)
	RemovePool(ctx context.Context, in *RemovePoolReq, opts ...grpc.CallOption) (*RemovePoolResp, error)
	AddBackend(ctx context.Context, in *AddBackendReq, opts ...grpc.CallOption) (*AddBackendResp, error)
	RemoveBackend(ctx context.Context, in *RemoveBackendReq, opts ...grpc.CallOption) (*RemoveBackendResp, error)
	PoolHealth(ctx context.Context, in *PoolHealthReq, opts ...grpc.CallOption) (*PoolHealthResp, error)
}

type loadBalancerClient struct {
	cc grpc.ClientConnInterface
}

func NewLoadBalancerClient(cc grpc.ClientConnInterface) LoadBalancerClient {
	return &loadBalancerClient{cc}
}

func (c *loadBalancerClient) AddPool(ctx context.Context, in *AddPoolReq, opts ...grpc.CallOption) (*AddPoolResp, error) {
	out := new(AddPoolResp)
	err := c.cc.Invoke(ctx, "/rollout.lb.LoadBalancer/AddPool", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loadBalancerClient) RemovePool(ctx context.Context, in *RemovePoolReq, opts ...grpc.CallOption) (*RemovePoolResp, error) {
	out := new(RemovePoolResp)
	err := c.cc.Invoke(ctx, "/rollout.lb.LoadBalancer/RemovePool", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loadBalancerClient) AddBackend(ctx context.Context, in *AddBackendReq, opts ...grpc.CallOption) (*AddBackendResp, error) {
	out := new(AddBackendResp)
	err := c.cc.Invoke(ctx, "/rollout.lb.LoadBalancer/AddBackend", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loadBalancerClient) RemoveBackend(ctx context.Context, in *RemoveBackendReq, opts ...grpc.CallOption) (*RemoveBackendResp, error) {
	out := new(RemoveBackendResp)
	err := c.cc.Invoke(ctx, "/rollout.lb.LoadBalancer/RemoveBackend", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loadBalancerClient) PoolHealth(ctx context.Context, in *PoolHealthReq, opts ...grpc.CallOption) (*PoolHealthResp, error) {
	out := new(PoolHealthResp)
	err := c.cc.Invoke(ctx, "/rollout.lb.LoadBalancer/PoolHealth", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LoadBalancerServer is the server API for LoadBalancer service.
// All implementations must embed UnimplementedLoadBalancerServer
// for forward compatibility
type LoadBalancerServer interface {
	AddPool(context.Context, *AddPoolReq) (*AddPoolResp, error)
	RemovePool(context.Context, *RemovePoolReq) (*RemovePoolResp, error)
	AddBackend(context.Context, *AddBackendReq) (*AddBackendResp, error)
	RemoveBackend(context.Context, *RemoveBackendReq) (*RemoveBackendResp, error)
	PoolHealth(context.Context, *PoolHealthReq) (*PoolHealthResp, error)
	mustEmbedUnimplementedLoadBalancerServer()
}

// UnimplementedLoadBalancerServer must be embedded to have forward compatible implementations.
type UnimplementedLoadBalancerServer struct {
}

func (UnimplementedLoadBalancerServer) AddPool(context.Context, *AddPoolReq) (*AddPoolResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddPool not implemented")
}
func (UnimplementedLoadBalancerServer) RemovePool(context.Context, *RemovePoolReq) (*RemovePoolResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemovePool not implemented")
}
func (UnimplementedLoadBalancerServer) AddBackend(context.Context, *AddBackendReq) (*AddBackendResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddBackend not implemented")
}
func (UnimplementedLoadBalancerServer) RemoveBackend(context.Context, *RemoveBackendReq) (*RemoveBackendResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveBackend not implemented")
}
func (UnimplementedLoadBalancerServer) PoolHealth(context.Context, *PoolHealthReq) (*PoolHealthResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PoolHealth not implemented")
}
func (UnimplementedLoadBalancerServer) mustEmbedUnimplementedLoadBalancerServer() {}

// UnsafeLoadBalancerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LoadBalancerServer will
// result in compilation errors.
type UnsafeLoadBalancerServer interface {
	mustEmbedUnimplementedLoadBalancerServer()
}

func RegisterLoadBalancerServer(s grpc.ServiceRegistrar, srv LoadBalancerServer) {
	s.RegisterService(&LoadBalancer_ServiceDesc, srv)
}

func _LoadBalancer_AddPool_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddPoolReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoadBalancerServer).AddPool(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rollout.lb.LoadBalancer/AddPool",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoadBalancerServer).AddPool(ctx, req.(*AddPoolReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _LoadBalancer_RemovePool_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemovePoolReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoadBalancerServer).RemovePool(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rollout.lb.LoadBalancer/RemovePool",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoadBalancerServer).RemovePool(ctx, req.(*RemovePoolReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _LoadBalancer_AddBackend_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddBackendReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoadBalancerServer).AddBackend(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rollout.lb.LoadBalancer/AddBackend",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoadBalancerServer).AddBackend(ctx, req.(*AddBackendReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _LoadBalancer_RemoveBackend_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveBackendReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoadBalancerServer).RemoveBackend(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rollout.lb.LoadBalancer/RemoveBackend",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoadBalancerServer).RemoveBackend(ctx, req.(*RemoveBackendReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _LoadBalancer_PoolHealth_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PoolHealthReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoadBalancerServer).PoolHealth(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rollout.lb.LoadBalancer/PoolHealth",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoadBalancerServer).PoolHealth(ctx, req.(*PoolHealthReq))
	}
	return interceptor(ctx, in, info, handler)
}

// LoadBalancer_ServiceDesc is the grpc.ServiceDesc for LoadBalancer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LoadBalancer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rollout.lb.LoadBalancer",
	HandlerType: (*LoadBalancerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AddPool",
			Handler:    _LoadBalancer_AddPool_Handler,
		},
		{
			MethodName: "RemovePool",
			Handler:    _LoadBalancer_RemovePool_Handler,
		},
		{
			MethodName: "AddBackend",
			Handler:    _LoadBalancer_AddBackend_Handler,
		},
		{
			MethodName: "RemoveBackend",
			Handler:    _LoadBalancer_RemoveBackend_Handler,
		},
		{
			MethodName: "PoolHealth",
			Handler:    _LoadBalancer_PoolHealth_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "lb.proto",
}
