package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func TestTCPServer_AuthHandshake(t *testing.T) {
	logger := log.New(io.Discard)

	srv, err := NewTCPServer(TCPServerOptions{
		Addr:        "127.0.0.1:0",
		RequireAuth: true,
		AllowedIPs:  []string{"127.0.0.1"},
		ValidateAuth: func(_ context.Context, sessionKey string) (bool, error) {
			return sessionKey == "good", nil
		},
	}, logger)
	if err != nil {
		t.Fatalf("NewTCPServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop() })

	addr := srv.listener.Addr().String()

	t.Run("rejects bad auth", func(t *testing.T) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
		_, _ = conn.Write([]byte(`{"auth":"bad"}` + "\n"))
		_, _ = conn.Write([]byte(`{"method":"ping","id":1}` + "\n"))

		r := bufio.NewReader(conn)
		if _, err := r.ReadBytes('\n'); err == nil {
			t.Fatalf("expected connection to be rejected")
		}
	})

	t.Run("accepts good auth", func(t *testing.T) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
		if _, err := conn.Write([]byte(`{"auth":"good"}` + "\n")); err != nil {
			t.Fatalf("write handshake: %v", err)
		}
		if _, err := conn.Write([]byte(`{"method":"ping","id":1}` + "\n")); err != nil {
			t.Fatalf("write ping: %v", err)
		}

		r := bufio.NewReader(conn)
		line, err := r.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read response: %v", err)
		}

		var resp RPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("unexpected rpc error: %s", resp.Error.Message)
		}
		m, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("unexpected result type %T", resp.Result)
		}
		if v, ok := m["pong"].(bool); !ok || !v {
			t.Fatalf("expected pong=true, got %v", m["pong"])
		}
	})
}

func TestTCPServer_IPAllowlist(t *testing.T) {
	logger := log.New(io.Discard)

	srv, err := NewTCPServer(TCPServerOptions{
		Addr:        "127.0.0.1:0",
		RequireAuth: false,
		AllowedIPs:  []string{"10.0.0.0/8"},
	}, logger)
	if err != nil {
		t.Fatalf("NewTCPServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop() })

	addr := srv.listener.Addr().String()

	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = conn.Write([]byte(`{"auth":""}` + "\n"))
	_, _ = conn.Write([]byte(`{"method":"ping","id":1}` + "\n"))

	r := bufio.NewReader(conn)
	if _, err := r.ReadBytes('\n'); err == nil {
		t.Fatalf("expected connection to be rejected by allowlist")
	}
}

func TestParseAllowedIPNets_ValidCIDR(t *testing.T) {
	nets, err := parseAllowedIPNets([]string{"10.0.0.0/8", "192.168.0.0/16"})
	if err != nil {
		t.Fatalf("parseAllowedIPNets: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("expected 2 nets, got %d", len(nets))
	}
}

func TestParseAllowedIPNets_SingleIPs(t *testing.T) {
	nets, err := parseAllowedIPNets([]string{"127.0.0.1", "192.168.1.1"})
	if err != nil {
		t.Fatalf("parseAllowedIPNets: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("expected 2 nets, got %d", len(nets))
	}
	// Single IPs should have /32 mask
	if ones, bits := nets[0].Mask.Size(); ones != 32 || bits != 32 {
		t.Errorf("expected /32 mask for single IP, got /%d", ones)
	}
}

func TestParseAllowedIPNets_IPv6(t *testing.T) {
	nets, err := parseAllowedIPNets([]string{"::1", "fe80::/10"})
	if err != nil {
		t.Fatalf("parseAllowedIPNets: %v", err)
	}
	if len(nets) != 2 {
		t.Fatalf("expected 2 nets, got %d", len(nets))
	}
}

func TestParseAllowedIPNets_EmptyString(t *testing.T) {
	nets, err := parseAllowedIPNets([]string{"", "  ", "127.0.0.1"})
	if err != nil {
		t.Fatalf("parseAllowedIPNets: %v", err)
	}
	// Empty strings should be skipped
	if len(nets) != 1 {
		t.Fatalf("expected 1 net (empty skipped), got %d", len(nets))
	}
}

func TestParseAllowedIPNets_InvalidCIDR(t *testing.T) {
	_, err := parseAllowedIPNets([]string{"not-a-cidr/8"})
	if err == nil {
		t.Fatal("expected error for invalid CIDR")
	}
}

func TestParseAllowedIPNets_InvalidIP(t *testing.T) {
	_, err := parseAllowedIPNets([]string{"not-an-ip"})
	if err == nil {
		t.Fatal("expected error for invalid IP")
	}
}

func TestExtractRemoteIP_TCPAddr(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.100"), Port: 12345}
	ip, err := extractRemoteIP(addr)
	if err != nil {
		t.Fatalf("extractRemoteIP: %v", err)
	}
	if ip.String() != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s", ip.String())
	}
}

func TestExtractRemoteIP_HostPort(t *testing.T) {
	ip, err := extractRemoteIP(stringAddr("10.0.0.1:8080"))
	if err != nil {
		t.Fatalf("extractRemoteIP: %v", err)
	}
	if ip.String() != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", ip.String())
	}
}

func TestExtractRemoteIP_NilAddr(t *testing.T) {
	_, err := extractRemoteIP(nil)
	if err == nil {
		t.Fatal("expected error for nil addr")
	}
}

func TestExtractRemoteIP_InvalidFormat(t *testing.T) {
	_, err := extractRemoteIP(stringAddr("not:valid:host:port"))
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestExtractRemoteIP_PlainIP(t *testing.T) {
	// Address without port
	ip, err := extractRemoteIP(stringAddr("192.168.1.50"))
	if err != nil {
		t.Fatalf("extractRemoteIP: %v", err)
	}
	if ip.String() != "192.168.1.50" {
		t.Errorf("expected 192.168.1.50, got %s", ip.String())
	}
}

func TestIPAllowed_Match(t *testing.T) {
	nets, _ := parseAllowedIPNets([]string{"192.168.0.0/16"})
	if !ipAllowed(net.ParseIP("192.168.1.100"), nets) {
		t.Error("expected IP to be allowed")
	}
}

func TestIPAllowed_NoMatch(t *testing.T) {
	nets, _ := parseAllowedIPNets([]string{"192.168.0.0/16"})
	if ipAllowed(net.ParseIP("10.0.0.1"), nets) {
		t.Error("expected IP to be rejected")
	}
}

func TestIPAllowed_NilIP(t *testing.T) {
	nets, _ := parseAllowedIPNets([]string{"192.168.0.0/16"})
	if ipAllowed(nil, nets) {
		t.Error("expected nil IP to be rejected")
	}
}

func TestIPAllowed_EmptyNets(t *testing.T) {
	if ipAllowed(net.ParseIP("127.0.0.1"), nil) {
		t.Error("expected IP to be rejected with no allowed nets")
	}
	if ipAllowed(net.ParseIP("127.0.0.1"), []*net.IPNet{}) {
		t.Error("expected IP to be rejected with empty allowed nets")
	}
}

func TestNewTCPServer_EmptyAddr(t *testing.T) {
	logger := log.New(io.Discard)
	_, err := NewTCPServer(TCPServerOptions{
		Addr: "",
	}, logger)
	if err == nil {
		t.Fatal("expected error for empty addr")
	}
}

func TestNewTCPServer_InvalidAllowedIPs(t *testing.T) {
	logger := log.New(io.Discard)
	_, err := NewTCPServer(TCPServerOptions{
		Addr:       "127.0.0.1:0",
		AllowedIPs: []string{"invalid-ip"},
	}, logger)
	if err == nil {
		t.Fatal("expected error for invalid allowed IPs")
	}
}

func TestNewTCPServer_NoAuthNoAllowlist(t *testing.T) {
	logger := log.New(io.Discard)
	srv, err := NewTCPServer(TCPServerOptions{
		Addr:        "127.0.0.1:0",
		RequireAuth: false,
		AllowedIPs:  nil,
	}, logger)
	if err != nil {
		t.Fatalf("NewTCPServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop() })

	addr := srv.listener.Addr().String()

	// Should be able to connect and ping without auth
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	// Still need to send handshake, but empty auth is OK
	if _, err := conn.Write([]byte(`{"auth":""}` + "\n")); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	if _, err := conn.Write([]byte(`{"method":"ping","id":1}` + "\n")); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	var resp RPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected rpc error: %s", resp.Error.Message)
	}
}

func TestTCPServer_ValidateAuthError(t *testing.T) {
	logger := log.New(io.Discard)

	srv, err := NewTCPServer(TCPServerOptions{
		Addr:        "127.0.0.1:0",
		RequireAuth: true,
		AllowedIPs:  []string{"127.0.0.1"},
		ValidateAuth: func(_ context.Context, sessionKey string) (bool, error) {
			return false, net.ErrClosed // Simulate an error
		},
	}, logger)
	if err != nil {
		t.Fatalf("NewTCPServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.Start(ctx) }()
	t.Cleanup(func() { _ = srv.Stop() })

	addr := srv.listener.Addr().String()

	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	_, _ = conn.Write([]byte(`{"auth":"any"}` + "\n"))
	_, _ = conn.Write([]byte(`{"method":"ping","id":1}` + "\n"))

	r := bufio.NewReader(conn)
	if _, err := r.ReadBytes('\n'); err == nil {
		t.Fatalf("expected connection to be rejected when auth validator returns error")
	}
}

func TestNewTCPServer_InvalidAddress(t *testing.T) {
	logger := log.New(io.Discard)
	_, err := NewTCPServer(TCPServerOptions{
		Addr: "invalid-address",
	}, logger)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestNewTCPServer_AddressInUse(t *testing.T) {
	// Start a listener on a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	addr := l.Addr().String()

	logger := log.New(io.Discard)
	_, err = NewTCPServer(TCPServerOptions{
		Addr: addr,
	}, logger)
	if err == nil {
		t.Error("expected error when address is in use")
	}
}

func TestExtractRemoteIP_HostName(t *testing.T) {
	// "example.com:80" -> host="example.com", port="80"
	// ParseIP("example.com") -> nil
	ip, err := extractRemoteIP(stringAddr("example.com:80"))
	if err == nil {
		t.Error("expected error for hostname")
	}
	if ip != nil {
		t.Errorf("expected nil IP, got %v", ip)
	}
}
