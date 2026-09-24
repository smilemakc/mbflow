// Package netguard provides an outbound-connection policy that protects
// workflow executors against SSRF (server-side request forgery) and
// unbounded response bodies.
//
// Every connection made through a Policy's dialer is checked right before the
// socket connects, against the actual resolved IP address. This covers
// redirects (every hop goes through the dialer) and defeats DNS rebinding
// (the check happens on the IP being connected to, not on an earlier lookup).
package netguard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
)

// MaxResponseBytes is the default maximum number of bytes read from an
// outbound HTTP response body (50 MiB).
const MaxResponseBytes int64 = 50 << 20

// ErrBlocked is matched (via errors.Is) by every error returned when a
// connection is rejected by a Policy.
var ErrBlocked = errors.New("netguard: connection blocked")

// ErrResponseTooLarge is returned by ReadAllLimited when the body exceeds the limit.
var ErrResponseTooLarge = errors.New("netguard: response body too large")

// BlockedError describes a rejected connection.
type BlockedError struct {
	IP     netip.Addr
	Reason string // e.g. "private", "loopback", "link-local"
}

func (e *BlockedError) Error() string {
	return fmt.Sprintf("netguard: connection to %s address %s is not allowed", e.Reason, e.IP)
}

// Is makes errors.Is(err, ErrBlocked) true.
func (e *BlockedError) Is(target error) bool { return target == ErrBlocked }

// Policy is an outbound-connection policy.
//
// The zero value blocks non-public addresses. Use AllowPrivate to disable
// checks entirely, or AllowedHosts to exempt specific destinations.
type Policy struct {
	// AllowPrivate disables all address checks.
	AllowPrivate bool
	// AllowedHosts are always allowed. Entries may be hostnames
	// ("internal.example.com", case-insensitive; "*.example.com" matches
	// subdomains), IP addresses ("10.0.0.5") or CIDRs ("10.1.0.0/16").
	AllowedHosts []string
}

type blockedRange struct {
	prefix netip.Prefix
	reason string
}

var blockedRanges = func() []blockedRange {
	specs := []struct{ cidr, reason string }{
		{"0.0.0.0/8", "unspecified"},
		{"10.0.0.0/8", "private"},
		{"100.64.0.0/10", "carrier-grade NAT"},
		{"127.0.0.0/8", "loopback"},
		{"169.254.0.0/16", "link-local"},
		{"172.16.0.0/12", "private"},
		{"192.0.0.0/24", "reserved"},
		{"192.0.2.0/24", "reserved"},
		{"192.168.0.0/16", "private"},
		{"198.18.0.0/15", "reserved"},
		{"198.51.100.0/24", "reserved"},
		{"203.0.113.0/24", "reserved"},
		{"224.0.0.0/4", "multicast"},
		{"240.0.0.0/4", "reserved"}, // includes 255.255.255.255 broadcast
		{"::/128", "unspecified"},
		{"::1/128", "loopback"},
		{"fc00::/7", "private"},
		{"fe80::/10", "link-local"},
		{"fec0::/10", "private"}, // deprecated site-local
		{"ff00::/8", "multicast"},
		{"100::/64", "reserved"}, // discard-only
		{"2001:db8::/32", "reserved"},
	}
	out := make([]blockedRange, len(specs))
	for i, s := range specs {
		out[i] = blockedRange{prefix: netip.MustParsePrefix(s.cidr), reason: s.reason}
	}
	return out
}()

var nat64Prefix = netip.MustParsePrefix("64:ff9b::/96")

// Classify reports whether ip is a non-public address and, if so, why.
// IPv4-mapped (::ffff:a.b.c.d), IPv4-compatible (::a.b.c.d) and NAT64
// (64:ff9b::a.b.c.d) forms are classified by their embedded IPv4 address.
func Classify(ip netip.Addr) (reason string, blocked bool) {
	if !ip.IsValid() {
		return "invalid", true
	}
	ip = ip.WithZone("")
	ip = embeddedIPv4(ip)
	for _, r := range blockedRanges {
		if r.prefix.Contains(ip) {
			return r.reason, true
		}
	}
	return "", false
}

// embeddedIPv4 returns the IPv4 address embedded in IPv6 transition forms,
// or ip unchanged.
func embeddedIPv4(ip netip.Addr) netip.Addr {
	if ip.Is4In6() {
		return ip.Unmap()
	}
	if !ip.Is6() {
		return ip
	}
	b := ip.As16()
	// NAT64 well-known prefix 64:ff9b::/96
	if nat64Prefix.Contains(ip) {
		return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	// Deprecated IPv4-compatible ::a.b.c.d (excluding :: and ::1)
	isCompat := true
	for i := 0; i < 12; i++ {
		if b[i] != 0 {
			isCompat = false
			break
		}
	}
	if isCompat && (b[12]|b[13]) != 0 {
		return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	return ip
}

// hostAllowed reports whether a hostname (not IP) is in the allowlist.
func (p Policy) hostAllowed(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if host == "" {
		return false
	}
	for _, entry := range p.AllowedHosts {
		e := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(entry)), ".")
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, "*.") {
			if strings.HasSuffix(host, e[1:]) {
				return true
			}
			continue
		}
		if e == host {
			return true
		}
	}
	return false
}

// ipAllowed reports whether ip matches an IP or CIDR allowlist entry.
func (p Policy) ipAllowed(ip netip.Addr) bool {
	ip = ip.WithZone("")
	unmapped := ip.Unmap()
	for _, entry := range p.AllowedHosts {
		e := strings.TrimSpace(entry)
		if strings.Contains(e, "/") {
			if pfx, err := netip.ParsePrefix(e); err == nil && (pfx.Contains(ip) || pfx.Contains(unmapped)) {
				return true
			}
			continue
		}
		if a, err := netip.ParseAddr(e); err == nil && (a.Unmap() == unmapped) {
			return true
		}
	}
	return false
}

// CheckIP returns a *BlockedError if the policy forbids connecting to ip.
func (p Policy) CheckIP(ip netip.Addr) error {
	if p.AllowPrivate {
		return nil
	}
	reason, blocked := Classify(ip)
	if !blocked || p.ipAllowed(ip) {
		return nil
	}
	return &BlockedError{IP: ip.WithZone(""), Reason: reason}
}

// control is a net.Dialer.Control hook: it runs after DNS resolution with the
// concrete IP about to be connected.
func (p Policy) control(_ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("netguard: invalid dial address %q: %w", address, err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("netguard: dial address %q is not an IP: %w", address, err)
	}
	return p.CheckIP(ip)
}

// DialContext dials like net.Dialer.DialContext, rejecting connections to
// disallowed addresses. Every resolved IP is checked immediately before
// connecting.
func (p Policy) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	if !p.AllowPrivate {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("netguard: invalid dial address %q: %w", address, err)
		}
		if !p.hostAllowed(host) {
			d.Control = p.control
		}
	}
	return d.DialContext(ctx, network, address)
}

// Transport returns a clone of base (http.DefaultTransport if nil) whose
// connections go through the policy's dialer. When the policy blocks private
// addresses, any configured proxy is disabled, since a proxy would make the
// final connection on our behalf and bypass the check.
func (p Policy) Transport(base *http.Transport) *http.Transport {
	var t *http.Transport
	if base != nil {
		t = base.Clone()
	} else if dt, ok := http.DefaultTransport.(*http.Transport); ok {
		t = dt.Clone()
	} else {
		t = &http.Transport{}
	}
	t.DialContext = p.DialContext
	t.DialTLSContext = nil
	t.Dial = nil    //nolint:staticcheck // clear deprecated hook so DialContext is used
	t.DialTLS = nil //nolint:staticcheck
	if !p.AllowPrivate {
		t.Proxy = nil
	}
	return t
}

// Client returns an *http.Client with the given timeout using p.Transport(nil).
func (p Policy) Client(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: p.Transport(nil)}
}

// CheckURL performs a best-effort pre-flight check of rawURL: the scheme must
// be http or https and the host (resolved via DNS if needed) must be allowed.
// It is useful for rejecting bad callback URLs early; it does NOT replace the
// dial-time check (DNS may change between CheckURL and the actual request).
func (p Policy) CheckURL(ctx context.Context, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("netguard: invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: unsupported URL scheme %q", ErrBlocked, u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("netguard: URL has no host")
	}
	if p.AllowPrivate || p.hostAllowed(host) {
		return nil
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return p.CheckIP(ip)
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("netguard: resolve %q: %w", host, err)
	}
	for _, ip := range ips {
		if err := p.CheckIP(ip); err != nil {
			return err
		}
	}
	return nil
}

// ReadAllLimited reads r up to limit bytes. If the stream is longer than
// limit it returns an error wrapping ErrResponseTooLarge. A limit <= 0 uses
// MaxResponseBytes.
func ReadAllLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = MaxResponseBytes
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: exceeds %d bytes limit", ErrResponseTooLarge, limit)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Package-level default policy
// ---------------------------------------------------------------------------

var (
	defaultMu        sync.RWMutex
	defaultPolicy    = Policy{AllowPrivate: true}
	defaultTransport *http.Transport // lazily built for defaultPolicy
)

// SetDefault replaces the process-wide default policy used by DefaultTransport
// and DefaultClient. The initial default is permissive (AllowPrivate: true) so
// library users and tests are unaffected; servers should opt in to blocking by
// calling SetDefault at startup. Safe for concurrent use.
func SetDefault(p Policy) {
	p.AllowedHosts = append([]string(nil), p.AllowedHosts...)
	defaultMu.Lock()
	old := defaultTransport
	defaultPolicy = p
	defaultTransport = nil
	defaultMu.Unlock()
	if old != nil {
		old.CloseIdleConnections()
	}
}

// Default returns the current process-wide default policy.
func Default() Policy {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	p := defaultPolicy
	p.AllowedHosts = append([]string(nil), p.AllowedHosts...)
	return p
}

// DefaultTransport returns a shared *http.Transport enforcing the current
// default policy. It is rebuilt when SetDefault is called, so callers should
// fetch it at request time rather than caching it.
func DefaultTransport() *http.Transport {
	defaultMu.RLock()
	t := defaultTransport
	defaultMu.RUnlock()
	if t != nil {
		return t
	}
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultTransport == nil {
		defaultTransport = defaultPolicy.Transport(nil)
	}
	return defaultTransport
}

// DefaultClient returns an *http.Client with the given timeout that uses
// DefaultTransport(). Cheap to call per request (connection pool is shared).
func DefaultClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: DefaultTransport()}
}

// Guard returns c with its Transport replaced by DefaultTransport() when c has
// no explicit Transport. Clients with a custom Transport are returned as-is.
// A nil c yields DefaultClient(30s).
func Guard(c *http.Client) *http.Client {
	if c == nil {
		return DefaultClient(30 * time.Second)
	}
	if c.Transport != nil {
		return c
	}
	cp := *c
	cp.Transport = DefaultTransport()
	return &cp
}
