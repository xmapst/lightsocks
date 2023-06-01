package dialer

import (
	"time"

	"go.uber.org/atomic"
)

var (
	DefaultOptions     []Option
	DefaultInterface   = atomic.NewString("")
	DefaultRoutingMark = atomic.NewInt32(0)
	DefaultTimeout     = 30 * time.Second
)

type option struct {
	interfaceName string
	routingMark   int
	timeout       time.Duration
}

type Option func(opt *option)

func WithInterface(name string) Option {
	return func(opt *option) {
		opt.interfaceName = name
	}
}

func WithRoutingMark(mark int) Option {
	return func(opt *option) {
		opt.routingMark = mark
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(opt *option) {
		opt.timeout = timeout
	}
}
