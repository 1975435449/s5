package proxy

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"ehang.io/nps/lib/file"
)

func TestSocks5AuthFallsBackToClientCredentialsWhenTaskAccountsAreEmpty(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	s := &Sock5ModeServer{}
	s.task = &file.Tunnel{
		Client: &file.Client{
			Cnf: &file.Config{
				U: "user",
				P: "pass",
			},
		},
		MultiAccount: &file.MultiAccount{AccountMap: map[string]string{}},
	}

	done := make(chan []byte, 1)
	go func() {
		_, _ = clientConn.Write([]byte{userAuthVersion, 4})
		_, _ = clientConn.Write([]byte("user"))
		_, _ = clientConn.Write([]byte{4})
		_, _ = clientConn.Write([]byte("pass"))
		resp := make([]byte, 2)
		_, _ = clientConn.Read(resp)
		done <- resp
	}()

	if err := s.Auth(serverConn); err != nil {
		t.Fatalf("expected client credentials to pass, got %v", err)
	}
	if resp := <-done; !bytes.Equal(resp, []byte{userAuthVersion, authSuccess}) {
		t.Fatalf("expected auth success response, got %#v", resp)
	}
}

func TestSocks5AuthUsesTaskMultiAccountsWhenConfigured(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	s := &Sock5ModeServer{}
	s.task = &file.Tunnel{
		Client: &file.Client{
			Cnf: &file.Config{
				U: "client-user",
				P: "client-pass",
			},
		},
		MultiAccount: &file.MultiAccount{AccountMap: map[string]string{"task-user": "task-pass"}},
	}

	done := make(chan []byte, 1)
	go func() {
		_, _ = clientConn.Write([]byte{userAuthVersion, 9})
		_, _ = clientConn.Write([]byte("task-user"))
		_, _ = clientConn.Write([]byte{9})
		_, _ = clientConn.Write([]byte("task-pass"))
		resp := make([]byte, 2)
		_, _ = clientConn.Read(resp)
		done <- resp
	}()

	if err := s.Auth(serverConn); err != nil {
		t.Fatalf("expected task credentials to pass, got %v", err)
	}
	if resp := <-done; !bytes.Equal(resp, []byte{userAuthVersion, authSuccess}) {
		t.Fatalf("expected auth success response, got %#v", resp)
	}
}

func TestSocks5HandleConnReadsFragmentedMethods(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	s := &Sock5ModeServer{}
	s.task = &file.Tunnel{
		Client:       &file.Client{Cnf: &file.Config{}},
		MultiAccount: &file.MultiAccount{AccountMap: map[string]string{}},
	}

	done := make(chan struct{})
	go func() {
		s.handleConn(serverConn)
		close(done)
	}()

	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	if _, err := clientConn.Write([]byte{5, 2}); err != nil {
		t.Fatalf("write greeting: %v", err)
	}
	if _, err := clientConn.Write([]byte{0}); err != nil {
		t.Fatalf("write first method: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if _, err := clientConn.Write([]byte{UserPassAuth}); err != nil {
		t.Fatalf("write second method: %v", err)
	}

	resp := make([]byte, 2)
	if _, err := io.ReadFull(clientConn, resp); err != nil {
		t.Fatalf("read negotiation response: %v", err)
	}
	if !bytes.Equal(resp, []byte{5, 0}) {
		t.Fatalf("expected no-auth negotiation response, got %#v", resp)
	}

	_, _ = clientConn.Write([]byte{5, 9, 0})
	reply := make([]byte, 8)
	_, _ = clientConn.Read(reply)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not finish after unsupported command")
	}
}
