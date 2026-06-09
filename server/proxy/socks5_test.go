package proxy

import (
	"bytes"
	"net"
	"testing"

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
