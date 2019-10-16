package grpcwrp

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	"github.com/makhov/grpcwrp/internal/resolver/dns"
)

const defaultPoolSize = 6

// ConnPool keeps pool of GRPC connections
type ConnPool struct {
	next uint32
	p    []*grpc.ClientConn
}

// Dial is predefined version of usual GRPC dial
func Dial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	cp, err := newPool(target, opts...)
	if err != nil {
		return nil, err
	}

	dnsBuilder := dns.NewBuilder()
	resolver.Register(dnsBuilder)
	resolver.SetDefaultScheme(dnsBuilder.Scheme())

	opts = append(
		opts,
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithBalancerName("roundrobin"),
		grpc.WithUnaryInterceptor(cp.ConnInterceptor),
	)

	return grpc.Dial(target, opts...)
}

// Close pool connections
func (cp *ConnPool) Close() error {
	for _, c := range cp.p {
		err := c.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ConnInterceptor intercepts unary request and get connection rom the pool
func (cp *ConnPool) ConnInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// override client conn from pool
	cc = cp.get()
	return invoker(ctx, method, req, reply, cc, opts...)
}

func (cp *ConnPool) get() *grpc.ClientConn {
	// Take the next client in the pool.
	// uint32 overflow resets to 0.
	idx := atomic.AddUint32(&cp.next, 1) % uint32(len(cp.p))

	return cp.p[idx]
}

func newPool(target string, opts ...grpc.DialOption) (*ConnPool, error) {
	cp := &ConnPool{
		p: make([]*grpc.ClientConn, defaultPoolSize),
	}

	for i := 0; i < defaultPoolSize; i++ {
		conn, err := grpc.Dial(target, opts...)
		if err != nil {
			return nil, err
		}

		cp.p[i] = conn
	}

	return cp, nil
}
