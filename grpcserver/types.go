package grpcserver

import (
	"google.golang.org/grpc"
)

// GRPCRegistrar is the subset of *grpc.Server we need for registration.
// It keeps parent code testable by accepting mocks/fakes if needed.
type GRPCRegistrar interface {
	RegisterService(desc *ServiceDesc, impl interface{})
}

// Alias ServiceDesc to the gRPC one so generated proto descriptors fit without conversion.
type ServiceDesc = grpc.ServiceDesc
