package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestService(t *testing.T) {
	eps := make([]Endpoint, 0)
	eps = append(eps, Endpoint{
		Name:   "getRoot",
		Method: http.MethodGet,
		URI:    "/",
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("World"))
		},
		Decorators: []EndpointDecorator{
			BeforeDecorator(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Hello "))
			}),
			AfterDecorator(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("!"))
				w.WriteHeader(200)
			}),
		},
	})
	s, err := NewService("test", "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}

	optionCount := 0
	of := func() error {
		optionCount++
		return nil
	}

	var trace = os.Stdout
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(trace),
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		t.Fatal(err)
	}

	s.Init(
		BeforeStart(of),
		AfterStart(of),
		BeforeStop(of),
		AfterStop(of),
		Tracing(sdktrace.AlwaysSample(), exporter))

	err = s.Endpoints(eps...)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Start(true); err != nil {
		t.Fatal(err)
	}

	rsp, err := http.Get(fmt.Sprintf("http://%s/", s.Server().Address()))
	if err != nil {
		t.Fatal(err)
	}
	defer rsp.Body.Close()

	expectedStatus := 200
	if rsp.StatusCode != expectedStatus {
		t.Fatalf("Unexpected statusCode, got '%d', expected '%d'", rsp.StatusCode, expectedStatus)
	}

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedBody := "Hello World!"
	if string(b) != expectedBody {
		t.Fatalf("Unexpected response, got '%s', expected '%s'", string(b), expectedBody)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("Unexpected error stopping sevice. %v", err)
	}

	expectedOptionCount := 4
	if optionCount != expectedOptionCount {
		t.Fatalf("Unexpected option count, got '%v', expected '%v'", optionCount, expectedOptionCount)
	}

	fmt.Println(trace)
}
