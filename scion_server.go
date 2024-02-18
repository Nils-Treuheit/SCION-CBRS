package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/netsec-ethz/scion-apps/pkg/pan"

	"github.com/netsec-ethz/scion-apps/pkg/quicutil"
)

// Server wraps a http.Server making it work with SCION
type SCIONServer struct {
	*http.Server
	rs pan.ReplySelector
}

// ListenAndServe listens for HTTP connections on the SCION address addr and calls Serve
// with handler to handle requests
func ListenAndServeRepSelect(addr string, handler http.Handler, repsel pan.ReplySelector) error {
	s := &SCIONServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		}, rs: repsel,
	}
	return s.ListenAndServe()
}

// ListenAndServe listens for HTTPS connections on the SCION address addr and calls Serve
// with handler to handle requests
func ListenAndServeTLSRepSelect(addr, certFile, keyFile string, handler http.Handler, repsel pan.ReplySelector) error {
	s := &SCIONServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		}, rs: repsel,
	}
	return s.ListenAndServeTLS(certFile, keyFile)
}

// ListenAndServe listens for HTTP connections on the SCION address addr and calls Serve
// with handler to handle requests
func ListenAndServe(addr string, handler http.Handler) error {
	s := &SCIONServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		}, rs: pan.NewDefaultReplySelector(),
	}
	return s.ListenAndServe()
}

// ListenAndServe listens for HTTPS connections on the SCION address addr and calls Serve
// with handler to handle requests
func ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	s := &SCIONServer{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		}, rs: pan.NewDefaultReplySelector(),
	}
	return s.ListenAndServeTLS(certFile, keyFile)
}

func (srv *SCIONServer) Serve(l net.Listener) error {
	// Providing a custom listener defeats the purpose of this library.
	panic("not implemented")
}

func (srv *SCIONServer) ServeTLS(l net.Listener, certFile, keyFile string) error {
	// Providing a custom listener defeats the purpose of this library.
	panic("not implemented")
}

// ListenAndServe listens for QUIC connections on srv.Addr and
// calls Serve to handle incoming requests
func (srv *SCIONServer) ListenAndServe() error {
	listener, err := listen(srv.Addr, srv.rs)
	if err != nil {
		return err
	}
	defer listener.Close()
	return srv.Server.Serve(listener)
}

func (srv *SCIONServer) ListenAndServeTLS(certFile, keyFile string) error {
	listener, err := listen(srv.Addr, srv.rs)
	if err != nil {
		return err
	}
	defer listener.Close()
	return srv.Server.ServeTLS(listener, certFile, keyFile)
}

func listen(addr string, rs pan.ReplySelector) (net.Listener, error) {
	tlsCfg := &tls.Config{
		NextProtos:   []string{quicutil.SingleStreamProto},
		Certificates: quicutil.MustGenerateSelfSignedCert(),
	}
	laddr, err := pan.ParseOptionalIPPort(addr)
	if err != nil {
		return nil, err
	}
	quicListener, err := pan.ListenQUIC(context.Background(), laddr, rs, tlsCfg, nil)
	if err != nil {
		return nil, err
	}
	return quicutil.SingleStreamListener{Listener: quicListener}, nil
}
