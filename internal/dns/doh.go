package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"

	"github.com/miekg/dns"
	utls "github.com/refraction-networking/utls"
	"github.com/xmapst/lightsocks/internal/dialer"
	"github.com/xmapst/lightsocks/internal/resolver"
)

const (
	// dotMimeType is the DoH mimetype that should be used.
	dotMimeType = "application/dns-message"
)

type dohClient struct {
	url       string
	transport *http.Transport
}

func (dc *dohClient) Exchange(m *dns.Msg) (msg *dns.Msg, err error) {
	return dc.ExchangeContext(context.Background(), m)
}

func (dc *dohClient) ExchangeContext(ctx context.Context, m *dns.Msg) (msg *dns.Msg, err error) {
	// https://datatracker.ietf.org/doc/html/rfc8484#section-4.1
	// In order to maximize cache friendliness, SHOULD use a DNS ID of 0 in every DNS request.
	newM := *m
	newM.Id = 0
	req, err := dc.newRequest(&newM)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	msg, err = dc.doRequest(req)
	if err == nil {
		msg.Id = m.Id
	}
	return
}

// newRequest returns a new DoH request given a dns.Msg.
func (dc *dohClient) newRequest(m *dns.Msg) (*http.Request, error) {
	buf, err := m.Pack()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, dc.url, bytes.NewReader(buf))
	if err != nil {
		return req, err
	}

	req.Header.Set("content-type", dotMimeType)
	req.Header.Set("accept", dotMimeType)
	return req, nil
}

func (dc *dohClient) doRequest(req *http.Request) (msg *dns.Msg, err error) {
	_client := &http.Client{Transport: dc.transport}
	resp, err := _client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	msg = &dns.Msg{}
	err = msg.Unpack(buf)
	return msg, err
}

func newDoHClient(url, iface string, r *Resolver) *dohClient {
	return &dohClient{
		url: url,
		transport: &http.Transport{
			ForceAttemptHTTP2: true,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := resolver.LookupIPWithResolver(ctx, host, r)
				if err != nil {
					return nil, err
				} else if len(ips) == 0 {
					return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, host)
				}
				ip := ips[rand.Intn(len(ips))]

				var options []dialer.Option
				if iface != "" {
					options = append(options, dialer.WithInterface(iface))
				}

				conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port), options...)
				if err != nil {
					return nil, err
				}
				uTlsConn := utls.UClient(conn, &utls.Config{InsecureSkipVerify: true}, utls.HelloChrome_Auto)
				err = uTlsConn.Handshake()
				if err != nil {
					return nil, fmt.Errorf("uTlsConn.Handshake() error: %+v", err)
				}
				return uTlsConn, nil
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				ips, err := resolver.LookupIPWithResolver(ctx, host, r)
				if err != nil {
					return nil, err
				} else if len(ips) == 0 {
					return nil, fmt.Errorf("%w: %s", resolver.ErrIPNotFound, host)
				}
				ip := ips[rand.Intn(len(ips))]

				var options []dialer.Option
				if iface != "" {
					options = append(options, dialer.WithInterface(iface))
				}

				return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port), options...)
			},
			TLSClientConfig: &tls.Config{
				// alpn identifier, see https://tools.ietf.org/html/draft-hoffman-dprive-dns-tls-alpn-00#page-6
				NextProtos: []string{"dns"},
			},
		},
	}
}
