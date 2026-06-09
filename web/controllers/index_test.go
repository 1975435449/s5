package controllers

import "testing"

func TestParseSocks5Accounts(t *testing.T) {
	accounts, err := parseSocks5Accounts("alice:secret\r\nbob:pass\n\n carol : token ")
	if err != nil {
		t.Fatalf("parseSocks5Accounts returned error: %v", err)
	}

	if len(accounts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accounts))
	}
	if accounts["alice"] != "secret" || accounts["bob"] != "pass" || accounts["carol"] != "token" {
		t.Fatalf("unexpected accounts: %#v", accounts)
	}
}

func TestParseSocks5AccountsRejectsMalformedRows(t *testing.T) {
	tests := []string{
		"alice",
		":secret",
		"alice:",
	}

	for _, input := range tests {
		if _, err := parseSocks5Accounts(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestValidateTunnelInputRejectsUnsafeNumbers(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		flowLimit int
		rateLimit int
		maxConn   int
	}{
		{name: "negative port", port: -1},
		{name: "port too large", port: 65536},
		{name: "negative flow limit", flowLimit: -1},
		{name: "negative rate limit", rateLimit: -1},
		{name: "negative max conn", maxConn: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateTunnelInput(tt.port, tt.flowLimit, tt.rateLimit, tt.maxConn); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateTunnelInputAllowsAutoPort(t *testing.T) {
	if err := validateTunnelInput(0, 0, 0, 0); err != nil {
		t.Fatalf("expected empty port and zero limits to be valid, got %v", err)
	}
}
