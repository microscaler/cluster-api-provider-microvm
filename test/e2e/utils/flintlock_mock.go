//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"net"
	"sync"

	flintlockv1 "github.com/liquidmetal-dev/flintlock/api/services/microvm/v1alpha1"
	flintlocktypes "github.com/liquidmetal-dev/flintlock/api/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/utils/pointer"
)

// FlintlockMock implements the flintlock MicroVM gRPC service for e2e tests so that
// CAPMVM can create "microvms" without a real flintlock/Firecracker stack.
type FlintlockMock struct {
	flintlockv1.UnimplementedMicroVMServer

	mu       sync.RWMutex
	microvms map[string]*flintlocktypes.MicroVM // keyed by UID
}

// NewFlintlockMock returns a new mock server.
func NewFlintlockMock() *FlintlockMock {
	return &FlintlockMock{
		microvms: make(map[string]*flintlocktypes.MicroVM),
	}
}

// CreateMicroVM stores the spec and returns a MicroVM with state CREATED so the controller marks the machine ready.
func (m *FlintlockMock) CreateMicroVM(ctx context.Context, req *flintlockv1.CreateMicroVMRequest) (*flintlockv1.CreateMicroVMResponse, error) {
	if req.GetMicrovm() == nil {
		return nil, fmt.Errorf("microvm spec required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	uid := req.Microvm.GetUid()
	if uid == "" {
		uid = fmt.Sprintf("mock-uid-%d", len(m.microvms)+1)
	}
	spec := *req.Microvm
	spec.Uid = pointer.String(uid)

	vm := &flintlocktypes.MicroVM{
		Spec: &spec,
		Status: &flintlocktypes.MicroVMStatus{
			State: flintlocktypes.MicroVMStatus_CREATED,
		},
	}
	m.microvms[uid] = vm
	return &flintlockv1.CreateMicroVMResponse{Microvm: vm}, nil
}

// GetMicroVM returns the stored MicroVM or empty if not found.
func (m *FlintlockMock) GetMicroVM(ctx context.Context, req *flintlockv1.GetMicroVMRequest) (*flintlockv1.GetMicroVMResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	vm, ok := m.microvms[req.GetUid()]
	if !ok {
		return &flintlockv1.GetMicroVMResponse{}, nil
	}
	return &flintlockv1.GetMicroVMResponse{Microvm: vm}, nil
}

// DeleteMicroVM removes the MicroVM from the store.
func (m *FlintlockMock) DeleteMicroVM(ctx context.Context, req *flintlockv1.DeleteMicroVMRequest) (*emptypb.Empty, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.microvms, req.GetUid())
	return &emptypb.Empty{}, nil
}

// ListMicroVMs returns all stored MicroVMs in the requested namespace.
func (m *FlintlockMock) ListMicroVMs(ctx context.Context, req *flintlockv1.ListMicroVMsRequest) (*flintlockv1.ListMicroVMsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []*flintlocktypes.MicroVM
	ns := req.GetNamespace()
	for _, vm := range m.microvms {
		if ns == "" || (vm.Spec != nil && vm.Spec.Namespace == ns) {
			list = append(list, vm)
		}
	}
	return &flintlockv1.ListMicroVMsResponse{Microvm: list}, nil
}

// ListMicroVMsStream streams the same as ListMicroVMs.
func (m *FlintlockMock) ListMicroVMsStream(req *flintlockv1.ListMicroVMsRequest, stream grpc.ServerStreamingServer[flintlockv1.ListMessage]) error {
	resp, err := m.ListMicroVMs(stream.Context(), req)
	if err != nil {
		return err
	}
	for _, vm := range resp.Microvm {
		if err := stream.Send(&flintlockv1.ListMessage{Microvm: vm}); err != nil {
			return err
		}
	}
	return nil
}

// ServeGRPC starts the mock on listener and returns the listener and a stop func.
func (m *FlintlockMock) ServeGRPC(ctx context.Context, listener net.Listener) (stop func(), err error) {
	srv := grpc.NewServer()
	flintlockv1.RegisterMicroVMServer(srv, m)
	reflection.Register(srv)

	go func() {
		_ = srv.Serve(listener)
	}()
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	stop = func() { srv.GracefulStop() }
	return stop, nil
}
