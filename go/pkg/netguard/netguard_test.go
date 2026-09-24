package netguard

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
		reason  string
	}{
		// public
		{"8.8.8.8", false, ""},
		{"1.1.1.1", false, ""},
		{"93.184.216.34", false, ""},
		{"2606:4700:4700::1111", false, ""},
		{"172.32.0.1", false, ""},
		{"100.128.0.1", false, ""},
		{"::ffff:8.8.8.8", false, ""},
		{"64:ff9b::808:808", false, ""},
		// loopback
		{"127.0.0.1", true, "loopback"},
		{"127.255.255.254", true, "loopback"},
		{"::1", true, "loopback"},
		// private
		{"10.0.0.1", true, "private"},
		{"172.16.0.1", true, "private"},
		{"172.31.255.255", true, "private"},
		{"192.168.1.1", true, "private"},
		{"fc00::1", true, "private"},
		{"fd12:3456::1", true, "private"},
		// link-local / metadata
		{"169.254.169.254", true, "link-local"},
		{"169.254.0.1", true, "link-local"},
		{"fe80::1", true, "link-local"},
		{"fe80::1%eth0", true, "link-local"},
		// unspecified
		{"0.0.0.0", true, "unspecified"},
		{"0.1.2.3", true, "unspecified"},
		{"::", true, "unspecified"},
		// multicast
		{"224.0.0.1", true, "multicast"},
		{"239.255.255.250", true, "multicast"},
		{"ff02::1", true, "multicast"},
		// CGNAT
		{"100.64.0.1", true, "carrier-grade NAT"},
		{"100.127.255.255", true, "carrier-grade NAT"},
		// reserved / broadcast
		{"255.255.255.255", true, "reserved"},
		{"240.0.0.1", true, "reserved"},
		// IPv4-mapped / compatible / NAT64 forms of blocked addresses
		{"::ffff:127.0.0.1", true, "loopback"},
		{"::ffff:10.0.0.1", true, "private"},
		{"::ffff:169.254.169.254", true, "link-local"},
		{"::ffff:100.64.0.1", true, "carrier-grade NAT"},
		{"::127.0.0.1", true, "loopback"},
		{"::10.0.0.1", true, "private"},
		{"64:ff9b::7f00:1", true, "loopback"},
		{"64:ff9b::a9fe:a9fe", true, "link-local"},
	}
	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := netip.MustParseAddr(tt.ip)
			reason, blocked := Classify(ip)
			if blocked != tt.blocked || reason != tt.reason {
				t.Fatalf("Classify(%s) = (%q, %v), want (%q, %v)", tt.ip, reason, blocked, tt.reason, tt.blocked)
			}
		})
	}
}

func TestClassifyInvalid(t *testing.T) {
	if _, blocked := Classify(netip.Addr{}); !blocked {
		t.Fatal("invalid address must be blocked")
	}
}

func TestCheckIP(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		ip      string
		wantErr bool
	}{
		{"public blocked policy", Policy{}, "8.8.8.8", false},
		{"private blocked", Policy{}, "10.0.0.1", true},
		{"metadata blocked", Policy{}, "169.254.169.254", true},
		{"allow private", Policy{AllowPrivate: true}, "10.0.0.1", false},
		{"allowlisted ip", Policy{AllowedHosts: []string{"10.0.0.1"}}, "10.0.0.1", false},
		{"allowlisted ip other ip", Policy{AllowedHosts: []string{"10.0.0.1"}}, "10.0.0.2", true},
		{"allowlisted cidr", Policy{AllowedHosts: []string{"10.1.0.0/16"}}, "10.1.2.3", false},
		{"allowlisted cidr mapped", Policy{AllowedHosts: []string{"10.1.0.0/16"}}, "::ffff:10.1.2.3", false},
		{"allowlisted cidr outside", Policy{AllowedHosts: []string{"10.1.0.0/16"}}, "10.2.0.1", true},
		{"allowlisted v6 cidr", Policy{AllowedHosts: []string{"fd00::/8"}}, "fd00::5", false},
		{"hostname entry doesn't allow ip", Policy{AllowedHosts: []string{"internal.local"}}, "10.0.0.1", true},
		{"bad entries ignored", Policy{AllowedHosts: []string{"", "  ", "not/a/cidr"}}, "10.0.0.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.CheckIP(netip.MustParseAddr(tt.ip))
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckIP(%s) err = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrBlocked) {
				t.Fatalf("error %v should match ErrBlocked", err)
			}
		})
	}
}

func TestBlockedErrorMessage(t *testing.T) {
	err := Policy{}.CheckIP(netip.MustParseAddr("10.0.0.1"))
	want := "netguard: connection to private address 10.0.0.1 is not allowed"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
	var be *BlockedError
	if !errors.As(err, &be) || be.Reason != "private" {
		t.Fatalf("expected *BlockedError with reason private, got %#v", err)
	}
}

func TestHostAllowed(t *testing.T) {
	p := Policy{AllowedHosts: []string{"Internal.Example.com", "*.svc.local", "10.0.0.0/8"}}
	tests := []struct {
		host string
		want bool
	}{
		{"internal.example.com", true},
		{"INTERNAL.example.com.", true},
		{"other.example.com", false},
		{"a.svc.local", true},
		{"a.b.svc.local", true},
		{"svc.local", false},
		{"evilsvc.local", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := p.hostAllowed(tt.host); got != tt.want {
			t.Errorf("hostAllowed(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func newLocalServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return srv, u.Port()
}

func TestTransportDial(t *testing.T) {
	srv, port := newLocalServer(t)

	tests := []struct {
		name    string
		policy  Policy
		url     string
		wantErr bool
	}{
		{"blocked by default policy", Policy{}, srv.URL, true},
		{"blocked via localhost name", Policy{}, "http://localhost:" + port, true},
		{"allowed with AllowPrivate", Policy{AllowPrivate: true}, srv.URL, false},
		{"allowed via IP allowlist", Policy{AllowedHosts: []string{"127.0.0.1"}}, srv.URL, false},
		{"allowed via CIDR allowlist", Policy{AllowedHosts: []string{"127.0.0.0/8"}}, srv.URL, false},
		{"allowed via hostname allowlist", Policy{AllowedHosts: []string{"localhost"}}, "http://localhost:" + port, false},
		{"hostname allowlist for other name", Policy{AllowedHosts: []string{"example.com"}}, srv.URL, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.policy.Client(5 * time.Second)
			resp, err := client.Get(tt.url)
			if tt.wantErr {
				if err == nil {
					resp.Body.Close()
					t.Fatal("expected error, got none")
				}
				if !errors.Is(err, ErrBlocked) {
					t.Fatalf("expected ErrBlocked, got %v", err)
				}
				if !strings.Contains(err.Error(), "is not allowed") {
					t.Fatalf("unexpected error text: %v", err)
				}
				return
			}
			if err != nil {
				// localhost may resolve only to ::1 on some hosts; httptest listens on 127.0.0.1.
				var opErr *net.OpError
				if errors.As(err, &opErr) && !errors.Is(err, ErrBlocked) {
					t.Skipf("local dial failed (environment): %v", err)
				}
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if string(body) != "ok" {
				t.Fatalf("unexpected body %q", body)
			}
		})
	}
}

func TestRedirectToPrivateBlocked(t *testing.T) {
	target, _ := newLocalServer(t)
	// Both servers listen on 127.0.0.1. The redirector is reached via the
	// allowlisted hostname "localhost" and redirects to the 127.0.0.1 literal,
	// which is not allowlisted, so the second hop must be blocked.
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()
	u, _ := url.Parse(redirector.URL)

	p := Policy{AllowedHosts: []string{"localhost"}}
	resp, err := p.Client(5 * time.Second).Get("http://localhost:" + u.Port())
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected redirect to 127.0.0.1 to be blocked")
	}
	if !errors.Is(err, ErrBlocked) {
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			t.Skipf("local dial failed (environment): %v", err)
		}
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
}

func TestTransportClonesBaseAndDisablesProxy(t *testing.T) {
	base := &http.Transport{MaxIdleConns: 7, Proxy: http.ProxyFromEnvironment}
	tr := Policy{}.Transport(base)
	if tr == base {
		t.Fatal("Transport must clone base")
	}
	if tr.MaxIdleConns != 7 {
		t.Fatal("base settings not preserved")
	}
	if tr.Proxy != nil {
		t.Fatal("proxy must be disabled when blocking private networks")
	}
	if tr.DialContext == nil {
		t.Fatal("DialContext not set")
	}
	if base.DialContext != nil {
		t.Fatal("base transport was mutated")
	}
	if (Policy{AllowPrivate: true}).Transport(base).Proxy == nil {
		t.Fatal("proxy should be preserved for permissive policy")
	}
}

func TestCheckURL(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		policy  Policy
		url     string
		wantErr bool
	}{
		{"public ip", Policy{}, "https://8.8.8.8/x", false},
		{"private ip", Policy{}, "http://10.0.0.1/", true},
		{"metadata", Policy{}, "http://169.254.169.254/latest/meta-data", true},
		{"v6 loopback", Policy{}, "http://[::1]:8080/", true},
		{"localhost resolves to loopback", Policy{}, "http://localhost/", true},
		{"bad scheme", Policy{}, "file:///etc/passwd", true},
		{"bad scheme permissive", Policy{AllowPrivate: true}, "gopher://x/", true},
		{"no host", Policy{}, "http:///path", true},
		{"permissive", Policy{AllowPrivate: true}, "http://10.0.0.1/", false},
		{"allowlisted host", Policy{AllowedHosts: []string{"localhost"}}, "http://localhost/", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.CheckURL(ctx, tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CheckURL(%s) err = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestReadAllLimited(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		limit   int64
		wantErr bool
	}{
		{"under", "abc", 5, false},
		{"exact", "abcde", 5, false},
		{"over", "abcdef", 5, true},
		{"empty", "", 5, false},
		{"default limit", "abc", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadAllLimited(strings.NewReader(tt.data), tt.limit)
			if tt.wantErr {
				if !errors.Is(err, ErrResponseTooLarge) {
					t.Fatalf("expected ErrResponseTooLarge, got %v", err)
				}
				return
			}
			if err != nil || string(got) != tt.data {
				t.Fatalf("got (%q, %v)", got, err)
			}
		})
	}
}

func TestDefaultPolicy(t *testing.T) {
	orig := Default()
	t.Cleanup(func() { SetDefault(orig) })

	if !orig.AllowPrivate {
		t.Fatal("initial default must be permissive")
	}

	srv, _ := newLocalServer(t)

	resp, err := DefaultClient(5 * time.Second).Get(srv.URL)
	if err != nil {
		t.Fatalf("permissive default should allow local: %v", err)
	}
	resp.Body.Close()

	SetDefault(Policy{AllowPrivate: false})
	if Default().AllowPrivate {
		t.Fatal("SetDefault not applied")
	}
	if _, err := DefaultClient(5 * time.Second).Get(srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected blocked after SetDefault, got %v", err)
	}
	// Guard on a plain client uses the default transport.
	if _, err := Guard(&http.Client{Timeout: time.Second}).Get(srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("Guard should apply default policy, got %v", err)
	}
	// Guard keeps an explicit transport.
	custom := &http.Client{Transport: http.DefaultTransport}
	if Guard(custom) != custom {
		t.Fatal("Guard must not override explicit transport")
	}

	SetDefault(Policy{AllowedHosts: []string{"127.0.0.1"}})
	resp, err = DefaultClient(5 * time.Second).Get(srv.URL)
	if err != nil {
		t.Fatalf("allowlisted default should allow: %v", err)
	}
	resp.Body.Close()

	// Default returns a copy.
	p := Default()
	p.AllowedHosts[0] = "mutated"
	if Default().AllowedHosts[0] != "127.0.0.1" {
		t.Fatal("Default must return a copy")
	}
}

func TestDefaultConcurrent(t *testing.T) {
	orig := Default()
	t.Cleanup(func() { SetDefault(orig) })
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			SetDefault(Policy{AllowPrivate: i%2 == 0})
		}(i)
		go func() {
			defer wg.Done()
			_ = Default()
			_ = DefaultTransport()
		}()
	}
	wg.Wait()
}
