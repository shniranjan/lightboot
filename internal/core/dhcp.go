package core

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// DHCPProxyServer implements a ProxyDHCP server (RFC 2131) that listens for
// DHCPDISCOVER broadcasts and responds with a DHCPOFFER containing ONLY
// PXE boot options. It does NOT assign IP addresses, subnet masks, or
// gateways — the real DHCP server on the network handles that.
//
// Clients merge the IP configuration from the real DHCPOFFER with the PXE
// options from this proxy. This enables fully automatic network booting
// without manual DHCP configuration.
type DHCPProxyServer struct {
	listenAddr string
	httpAddr   string // HTTP server address (for iPXE script URL)
	tftpAddr   string // TFTP server address (for boot file downloads)
	log        *Logger

	conn   net.PacketConn
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDHCPProxyServer creates a new DHCP Proxy server.
//
//   - listenAddr: UDP address to bind, e.g. "0.0.0.0:67"
//   - httpAddr:   our HTTP server address, e.g. "192.168.1.10:8080"
//   - tftpAddr:   our TFTP server address, e.g. "192.168.1.10"
func NewDHCPProxyServer(listenAddr, httpAddr, tftpAddr string, log *Logger) *DHCPProxyServer {
	return &DHCPProxyServer{
		listenAddr: listenAddr,
		httpAddr:   httpAddr,
		tftpAddr:   tftpAddr,
		log:        log,
	}
}

// Start begins listening for DHCPDISCOVER broadcasts on the configured UDP
// port. It returns immediately; packet handling runs in background goroutines.
//
// On Linux, binding to port 67 typically requires CAP_NET_BIND_SERVICE or
// root. Run with:
//
//	sudo setcap cap_net_bind_service=+ep ./bin/lightboot
func (d *DHCPProxyServer) Start() error {
	d.ctx, d.cancel = context.WithCancel(context.Background())

	pc, err := net.ListenPacket("udp4", d.listenAddr)
	if err != nil {
		return fmt.Errorf("dhcp proxy: failed to bind %s: %w (try: sudo setcap cap_net_bind_service=+ep ./lightboot)", d.listenAddr, err)
	}
	d.conn = pc

	d.wg.Add(1)
	go d.loop()

	d.log.Info("dhcp", "ProxyDHCP server listening on %s", d.listenAddr)
	d.log.Info("dhcp", "TFTP server address: %s, HTTP server: %s", d.tftpAddr, d.httpAddr)
	return nil
}

// Stop shuts down the DHCP proxy server gracefully.
func (d *DHCPProxyServer) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	if d.conn != nil {
		d.conn.Close()
	}
	d.wg.Wait()
	d.log.Info("dhcp", "ProxyDHCP server stopped")
}

// loop reads packets from the UDP socket and dispatches them to handlers.
func (d *DHCPProxyServer) loop() {
	defer d.wg.Done()

	buf := make([]byte, 1500) // MTU-sized buffer
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
		}

		// Read without deadline — conn.Close() on Stop will unblock
		n, peer, err := d.conn.ReadFrom(buf)
		if err != nil {
			select {
			case <-d.ctx.Done():
				return
			default:
				d.log.Error("dhcp", "Read error: %v", err)
				continue
			}
		}

		// Parse the DHCP packet
		msg, err := dhcpv4.FromBytes(buf[:n])
		if err != nil {
			d.log.Error("dhcp", "Failed to parse DHCP packet from %s: %v", peer, err)
			continue
		}

		// Only process DHCPDISCOVER (option 53 == 1) for proxy mode.
		// We ignore DHCPREQUEST, DHCPOFFER from other servers, etc.
		if msg.MessageType() != dhcpv4.MessageTypeDiscover {
			continue
		}

		// Handle the discovery concurrently
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.handleDiscover(msg, peer)
		}()
	}
}

// handleDiscover processes a DHCPDISCOVER and sends an appropriate
// DHCPOFFER with PXE boot options.
func (d *DHCPProxyServer) handleDiscover(msg *dhcpv4.DHCPv4, peer net.Addr) {
	// Detect client architecture from option 93
	arch := d.detectArchitecture(msg)
	bootFile := d.bootFileForArch(arch)

	// Detect if this is an iPXE client (user-class contains "iPXE")
	isIPXE := d.isIPXEClient(msg)

	// Extract client identifier for logging
	clientID := d.clientIdentifier(msg)

	d.log.Info("dhcp", "DHCPDISCOVER from %s (client=%s, arch=%d, iPXE=%v) → offering %s",
		peer, clientID, arch, isIPXE, bootFile)

	// Build the DHCPOFFER
	offer, err := d.buildOffer(msg, bootFile, isIPXE)
	if err != nil {
		d.log.Error("dhcp", "Failed to build DHCPOFFER for %s: %v", peer, err)
		return
	}

	// UDPAddr for the client (broadcast since client doesn't have an IP yet)
	dst := &net.UDPAddr{
		IP:   net.IPv4bcast,
		Port: 68, // DHCP client port
	}
	// If the message requests unicast and the broadcast flag is 0, use the
	// peer address directed to the client. For simplicity, always use broadcast.
	// Some clients only accept broadcast offers during discovery.

	if _, err := d.conn.WriteTo(offer.ToBytes(), dst); err != nil {
		d.log.Error("dhcp", "Failed to send DHCPOFFER to %s: %v", peer, err)
		return
	}

	d.log.Info("dhcp", "DHCPOFFER sent to %s (bootfile=%s)", peer, bootFile)
}

// buildOffer constructs a ProxyDHCP DHCPOFFER packet.
//
// In ProxyDHCP mode we:
//   - Do NOT set yiaddr (no IP offered)
//   - Do NOT set subnet mask, router, DNS, or lease time
//   - Set option 53 = DHCPOFFER
//   - Set option 54 = our server identifier (our IP)
//   - Set option 60 = "PXEClient"
//   - Set option 66 = TFTP server address
//   - Set option 67 = boot filename
//   - For iPXE clients: set option 43 with the chain script URL
func (d *DHCPProxyServer) buildOffer(discover *dhcpv4.DHCPv4, bootFile string, isIPXE bool) (*dhcpv4.DHCPv4, error) {
	// Start with a DHCPOFFER based on the discover
	offer, err := dhcpv4.NewReplyFromRequest(discover)
	if err != nil {
		return nil, fmt.Errorf("build DHCPOFFER: %w", err)
	}

	// Clear fields we don't want to offer in Proxy mode
	offer.YourIPAddr = net.IPv4zero // No IP offered
	// offer.Router = nil (not assignable in this library version)
	// offer.SubnetMask = nil (not assignable in this library version)
	// offer.DNS = nil (not assignable in this library version)
	// offer.IPAddressLeaseTime = 0 (not assignable in this library version)

	// Determine our server IP (the IP we're listening on)
	serverIP := d.localIP()
	if serverIP == nil {
		serverIP = net.IPv4(0, 0, 0, 0)
	}

	// Option 54: DHCP Server Identifier — our IP
	offer.UpdateOption(dhcpv4.OptServerIdentifier(serverIP))

	// Option 60: Vendor Class Identifier
	offer.UpdateOption(dhcpv4.OptClassIdentifier("PXEClient"))

	// Translate tftpAddr to an IP if it's an IP:port string
	tftpIP := d.tftpAddrToIP()

	// Option 66: TFTP Server Name — our IP as string
	offer.UpdateOption(dhcpv4.OptTFTPServerName(tftpIP.String()))

	// Option 67: Bootfile Name
	offer.UpdateOption(dhcpv4.OptBootFileName(bootFile))

	// For iPXE clients, embed the script URL in Option 43 (vendor encapsulated)
	// Sub-option 71 = iPXE script URL
	if isIPXE {
		scriptURL := fmt.Sprintf("http://%s/api/boot/chain", d.httpAddr)
		vendorOpts := dhcpv4.Options{
			71: []byte(scriptURL),
		}
		offer.UpdateOption(dhcpv4.OptGeneric(dhcpv4.GenericOptionCode(43), vendorOpts.ToBytes()))
	}

	return offer, nil
}

// detectArchitecture extracts the client system architecture from DHCP
// option 93. Returns 0 (BIOS x86) if the option is not present.
//
// Common values:
//
//	  0 — BIOS x86
//	  6 — EFI x86-64 (IA32)
//	  7 — EFI BC (x86-64)
//	  9 — EFI x64
//	 11 — EFI ARM64
func (d *DHCPProxyServer) detectArchitecture(msg *dhcpv4.DHCPv4) int {

	opt := msg.Options.Get(dhcpv4.GenericOptionCode(93))
	if opt == nil {
		return 0 // Default: BIOS x86
	}

	if len(opt) >= 2 {
		return int(opt[0])<<8 | int(opt[1])
	}
	return 0
}

// bootFileForArch returns the appropriate boot file for the given client
// architecture. This is the initial NBP (Network Bootstrap Program) that
// the client downloads via TFTP.
func (d *DHCPProxyServer) bootFileForArch(arch int) string {
	switch arch {
	case 0: // BIOS x86
		return "undionly.kpxe"
	case 6: // EFI IA32
		return "ipxe.efi"
	case 7: // EFI BC (x86-64)
		return "ipxe.efi"
	case 9: // EFI x64
		return "ipxe.efi"
	case 11: // EFI ARM64
		return "snponly.efi"
	default:
		d.log.Warn("dhcp", "Unknown client architecture %d, defaulting to undionly.kpxe", arch)
		return "undionly.kpxe"
	}
}

// isIPXEClient checks whether the DHCPDISCOVER came from an iPXE client.
// iPXE includes "iPXE" in the user-class option (77).
func (d *DHCPProxyServer) isIPXEClient(msg *dhcpv4.DHCPv4) bool {

	opt := msg.Options.Get(dhcpv4.GenericOptionCode(77))
	if opt == nil {
		return false
	}
	return strings.Contains(string(opt), "iPXE")
}

// clientIdentifier returns a human-readable client identifier for logging.
func (d *DHCPProxyServer) clientIdentifier(msg *dhcpv4.DHCPv4) string {
	// Try DHCP Client Identifier (option 61)
	if opt := msg.Options.Get(dhcpv4.GenericOptionCode(61)); opt != nil {
		return fmt.Sprintf("%x", opt)
	}
	// Fall back to hardware address
	return msg.ClientHWAddr.String()
}

// localIP tries to determine the local IP address that this server is
// listening on. Used for the DHCP Server Identifier option.
func (d *DHCPProxyServer) localIP() net.IP {
	host, _, err := net.SplitHostPort(d.listenAddr)
	if err != nil {
		return nil
	}

	// If explicitly bound to a specific IP, use it
	if host != "" && host != "0.0.0.0" {
		if ip := net.ParseIP(host); ip != nil {
			return ip.To4()
		}
	}

	// Otherwise find the first non-loopback IPv4 address
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4
			}
		}
	}
	return nil
}

// tftpAddrToIP parses the TFTP address and returns an IPv4 address.
// The TFTP address may be "host:port" or just "host".
func (d *DHCPProxyServer) tftpAddrToIP() net.IP {
	host, _, err := net.SplitHostPort(d.tftpAddr)
	if err != nil {
		// Not a host:port format, try parsing as plain IP
		host = d.tftpAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.To4()
	}
	return d.localIP()
}

// GetLocalIP is a helper that returns a string IP address suitable for use
// in DHCP options. If the provided address is a specific IP (not 0.0.0.0),
// it returns that. Otherwise it finds the first non-loopback IPv4 address.
func GetLocalIP(listenAddr string) string {
	host, _, err := net.SplitHostPort(listenAddr)
	if err != nil {
		host = listenAddr
	}

	if host != "" && host != "0.0.0.0" {
		return host
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}
