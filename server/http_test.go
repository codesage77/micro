package server

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type stubHandler struct {
	message string
	status  int
}

func (sh *stubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(sh.status)
	w.Write([]byte(sh.message))
}

func NewStubHandler(message string, status int) *stubHandler {
	return &stubHandler{
		message: message,
		status:  status,
	}
}

func TestHttpServer(t *testing.T) {
	s := NewHttpServer(NewStubHandler("Hello World!", 200), Hostname("localhost"), Port(-1))

	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	rsp, err := http.Get(fmt.Sprintf("http://%s/", s.Address()))
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
		t.Fatal(err)
	}
}

func TestHttpsServer(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Dir(pwd)

	s := NewHttpServer(NewStubHandler("Hello World!", 200),
		Hostname("localhost"), Port(-1),
		TLS(
			fmt.Sprintf("%s/cert/cert.pem", certPath),
			fmt.Sprintf("%s/cert/key.pem", certPath)))

	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			// Certificate is not signed by a known Certificate Authority.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	rsp, err := client.Get(fmt.Sprintf("https://%s/", s.Address()))
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
		t.Fatal(err)
	}
}
func TestChiHttpServer(t *testing.T) {
	h := NewChiHandler()
	h.Method(http.MethodGet, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello World!"))
	}))

	s := NewHttpServer(h)

	if err := s.Start(); err != nil {
		t.Fatal(err)
	}

	rsp, err := http.Get(fmt.Sprintf("http://%s/", s.Address()))
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
		t.Fatal(err)
	}
}
