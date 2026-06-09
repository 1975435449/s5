package bridge

import (
	"testing"

	"ehang.io/nps/lib/file"
)

func TestTunnelBelongsToClientHandlesNil(t *testing.T) {
	if tunnelBelongsToClient(nil, 1) {
		t.Fatal("nil tunnel must not match")
	}
	if tunnelBelongsToClient(&file.Tunnel{}, 1) {
		t.Fatal("tunnel with nil client must not match")
	}
	if !tunnelBelongsToClient(&file.Tunnel{Client: &file.Client{Id: 1}}, 1) {
		t.Fatal("tunnel with matching client should match")
	}
}

func TestHostBelongsToClientHandlesNil(t *testing.T) {
	if hostBelongsToClient(nil, 1) {
		t.Fatal("nil host must not match")
	}
	if hostBelongsToClient(&file.Host{}, 1) {
		t.Fatal("host with nil client must not match")
	}
	if !hostBelongsToClient(&file.Host{Client: &file.Client{Id: 1}}, 1) {
		t.Fatal("host with matching client should match")
	}
}
