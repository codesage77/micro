package service

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/codesage77/micro/server"
	"github.com/rs/zerolog/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Option func(o *Options)

type Options struct {
	BeforeStart []func() error
	BeforeStop  []func() error
	AfterStart  []func() error
	AfterStop   []func() error

	Port            int
	CertificateFile string
	KeyFile         string
	Context         context.Context
	Signal          bool
	Sampler         sdktrace.Sampler
	Exporter        sdktrace.SpanExporter
}

func BeforeStart(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

func BeforeStop(fn func() error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

func AfterStart(fn func() error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

func AfterStop(fn func() error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
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

func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

func Tracing(s sdktrace.Sampler, e sdktrace.SpanExporter) Option {
	return func(o *Options) {
		o.Sampler = s
		o.Exporter = e
	}
}

type Endpoint struct {
	Name        string
	Method      string
	URI         string
	HandlerFunc http.HandlerFunc
	Decorators  []EndpointDecorator
}

func BeforeDecorator(df func(w http.ResponseWriter, r *http.Request)) EndpointDecorator {
	return EndpointDecorator{Type: Before, DecoratorFunc: df}
}

func AfterDecorator(df func(w http.ResponseWriter, r *http.Request)) EndpointDecorator {
	return EndpointDecorator{Type: After, DecoratorFunc: df}
}

type EndpointDecoratorType uint8

const (
	Before EndpointDecoratorType = iota
	After
)

type EndpointDecorator struct {
	Type          EndpointDecoratorType
	DecoratorFunc func(w http.ResponseWriter, r *http.Request)
}

type Service struct {
	name          string
	version       string
	opts          Options
	server        *server.HttpServer
	handler       chi.Router
	traceProvider *sdktrace.TracerProvider
}

func NewService(name string, version string) (*Service, error) {
	s := &Service{
		name:    name,
		version: version,
		handler: server.NewChiHandler(),
	}
	return s, nil
}

func (s *Service) Init(opts ...Option) error {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	if options.Context == nil {
		options.Context = context.Background()
	}
	s.opts = options
	return nil
}

func (s *Service) Endpoints(endpoints ...Endpoint) error {
	for _, ep := range endpoints {
		var h http.Handler = ep.HandlerFunc
		for _, d := range ep.Decorators {
			h = decorateHandler(h, d)
		}

		if s.opts.Sampler != nil && s.opts.Exporter != nil {
			tp, err := initTracer(s.name, s.version, s.opts.Sampler, s.opts.Exporter)
			if err != nil {
				return err
			}
			s.traceProvider = tp
			h = trace(h, tp, s.name, ep.Name)
		}

		s.handler.Method(ep.Method, ep.URI, h)
	}
	return nil
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Options() Options {
	return s.opts
}

func (s *Service) Server() *server.HttpServer {
	return s.server
}

func (s *Service) Start() error {
	for _, fn := range s.opts.BeforeStart {
		if err := fn(); err != nil {
			return err
		}
	}

	s.server = server.NewHttpServer(s.handler, server.Port(s.opts.Port), server.TLS(s.opts.CertificateFile, s.opts.KeyFile))
	if err := s.server.Start(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStart {
		if err := fn(); err != nil {
			return err
		}
	}

	ch := make(chan os.Signal, 1)
	if s.opts.Signal {
		signals := []os.Signal{
			syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL,
		}
		signal.Notify(ch, signals...)
	}

	go func() {
		select {
		// wait on kill signal
		case <-ch:
		// wait on context cancel
		case <-s.opts.Context.Done():
		}

		if err := s.Stop(); err != nil {
			log.Info().Msgf("An error occurred when stopping the service %v", err)
		}
	}()

	return nil
}

func (s *Service) Stop() error {
	for _, fn := range s.opts.BeforeStop {
		if err := fn(); err != nil {
			return err
		}
	}

	if err := s.Server().Stop(); err != nil {
		return err
	}

	for _, fn := range s.opts.AfterStop {
		if err := fn(); err != nil {
			return err
		}
	}

	if s.traceProvider != nil {
		return s.traceProvider.Shutdown(s.opts.Context)
	}
	return nil
}

func (s *Service) String() string {
	return s.name
}

func decorateHandler(h http.Handler, ed EndpointDecorator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ed.Type == Before {
			ed.DecoratorFunc(w, r)
		}
		h.ServeHTTP(w, r)
		if ed.Type == After {
			ed.DecoratorFunc(w, r)
		}
	})
}
