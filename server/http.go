package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Option func(o *Options)

type Options struct {
	Hostname        string
	Port            int
	CertificateFile string
	KeyFile         string
	ShutdownTimeout int
}

func Hostname(h string) Option {
	return func(o *Options) {
		o.Hostname = h
	}
}

func Port(p int) Option {
	return func(o *Options) {
		o.Port = p
	}
}

func TLS(certFile string, keyFile string) Option {
	return func(o *Options) {
		o.CertificateFile = certFile
		o.KeyFile = keyFile
	}
}

func ShutdownTimeout(t int) Option {
	return func(o *Options) {
		o.ShutdownTimeout = t
	}
}

type HttpServer struct {
	mtx     sync.RWMutex
	srv     *http.Server
	handler http.Handler
	address string
	options Options
	exit    chan chan error
}

func NewHttpServer(handler http.Handler, opts ...Option) *HttpServer {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Hostname == "" {
		options.Hostname = "localhost"
	}

	if options.Port < 0 {
		options.Port = 0
	}

	if options.ShutdownTimeout <= 0 {
		options.ShutdownTimeout = 5
	}

	return &HttpServer{
		address: fmt.Sprintf("%s:%d", options.Hostname, options.Port),
		handler: handler,
		options: options,
		exit:    make(chan chan error),
	}
}

func (hs *HttpServer) Address() string {
	hs.mtx.RLock()
	defer hs.mtx.RUnlock()
	return hs.address
}

func (hs *HttpServer) Start() error {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", hs.address)
	if err != nil {
		return err
	}

	hs.mtx.Lock()
	hs.address = l.Addr().String()
	hs.mtx.Unlock()

	log.Info().Msgf("Starting server. Listening at %s", hs.String())

	hs.srv = &http.Server{Handler: hs.handler}

	go func() {
		if hs.options.CertificateFile != "" && hs.options.KeyFile != "" {
			if err := hs.srv.ServeTLS(l, hs.options.CertificateFile, hs.options.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Error().Msgf("%v", err)
			}
		} else if err := hs.srv.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Error().Msgf("%v", err)
		}
	}()

	go func() {
		ch := <-hs.exit

		ctxShutDown, cancel := context.WithTimeout(context.Background(), time.Duration(hs.options.ShutdownTimeout)*time.Second)
		defer func() {
			cancel()
		}()
		if err = hs.srv.Shutdown(ctxShutDown); err != nil {
			log.Error().Msgf("Server Shutdown Failed:%+s", err)
		}

		ch <- nil
	}()

	return nil
}

func (hs *HttpServer) Stop() error {
	log.Info().Msg("Stopping server")
	ch := make(chan error)
	hs.exit <- ch
	var err error = <-ch
	if err == nil {
		log.Info().Msg("Stopped server.")
	}
	return err
}

func (s *HttpServer) String() string {
	return fmt.Sprintf("%s://%v", "http", s.Address())
}
