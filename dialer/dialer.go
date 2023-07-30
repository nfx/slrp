package dialer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/nfx/slrp/app"
	"github.com/nfx/slrp/dialer/ini"
	"github.com/rs/zerolog/log"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// wireGuardDialer implements the app.Service interface and represents the WireGuard dialer.
type wireGuardDialer struct {
	standard net.Dialer
	tunnel   *netstack.Net
	conf     ini.Config
	verbose  bool
}

// NewDialer creates a new instance of the WireGuard dialer.
func NewDialer() *wireGuardDialer {
	return &wireGuardDialer{}
}

// Configure initializes the WireGuard dialer with the provided configuration.
func (d *wireGuardDialer) Configure(c app.Config) error {
	configFile := c.StrOr("wireguard_config_file", "")
	if configFile == "" {
		// If no WireGuard config file is specified, use the standard net.Dialer.
		log.Warn().Msg("using clear dialer")
		return nil
	}
	d.verbose = c.BoolOr("wireguard_verbose", false)
	conf, err := ini.ParseINI(configFile)
	if err != nil {
		return fmt.Errorf("parse %s: %w", configFile, err)
	}
	d.conf = conf

	log.Info().
		Str("config", configFile).
		Str("endpoint", d.conf["Peer"]["Endpoint"]).
		Msg("configured WireGuard dialer")

	// https://www.wireguard.com/xplatform/
	// Create the WireGuard tunnel and device based on the configuration.
	tun, tnet, err := d.createNetTUN()
	if err != nil {
		return fmt.Errorf("create net tun: %w", err)
	}
	bind := conn.NewDefaultBind()
	verboseF := func(format string, args ...any) {}
	if d.verbose {
		verboseF = func(format string, args ...any) {
			log.Debug().Str("service", "wireguard").Msgf(format, args...)
		}
	}
	dev := device.NewDevice(tun, bind, &device.Logger{
		// Define custom logger functions for WireGuard device logging.
		Errorf: func(format string, args ...any) {
			log.Error().Str("service", "wireguard").Msgf(format, args...)
		},
		Verbosef: verboseF,
	})
	ipc, err := d.getIpsSetShim()
	if err != nil {
		return fmt.Errorf("ipc shim: %w", err)
	}
	err = dev.IpcSetOperation(ipc)
	if err != nil {
		return fmt.Errorf("ipc set: %w", err)
	}
	err = dev.Up()
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}
	d.tunnel = tnet
	return nil
}

// addrsFromConfig parses a comma-separated list of IP addresses from the configuration section and key.
func (d *wireGuardDialer) addrsFromConfig(section, key string) (addrs []netip.Addr, err error) {
	// Fetch the comma-separated value from the configuration.
	value := d.conf[section][key]
	for _, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if strings.Contains(v, ":") {
			// Skip IPv6 addresses for now (not supported).
			continue
		}
		if strings.Contains(v, "/") {
			// Parse the IP address with subnet prefix if present.
			addr, err := netip.ParsePrefix(v)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, addr.Addr())
			continue
		}
		// Parse the IP address without subnet prefix.
		addr, err := netip.ParseAddr(v)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

// createNetTUN creates a network TUN interface with the specified IP addresses and DNS servers.
func (d *wireGuardDialer) createNetTUN() (tun.Device, *netstack.Net, error) {
	addrs, err := d.addrsFromConfig("Interface", "Address")
	if err != nil {
		return nil, nil, err
	}
	dns, err := d.addrsFromConfig("Interface", "DNS")
	if err != nil {
		return nil, nil, err
	}
	// Create the network TUN interface using the netstack package with the obtained IP addresses and DNS servers.
	return netstack.CreateNetTUN(addrs, dns, 1420)
}

// writeHexKeyAs writes a base64-encoded key to the buffer with the specified label.
func (d *wireGuardDialer) writeHexKeyAs(b *bytes.Buffer, section, key, as string) error {
	// Decode the base64-encoded key from the configuration.
	raw, err := base64.StdEncoding.DecodeString(d.conf[section][key])
	if err != nil {
		return err
	}
	// Write the key to the buffer as a hexadecimal value with the given label.
	_, err = b.WriteString(fmt.Sprintf("%s=%x\n", as, raw))
	return err
}

func (d *wireGuardDialer) getIpsSetShim() (*bytes.Buffer, error) {
	b := &bytes.Buffer{}
	// Write private, public, and preshared keys to the buffer as hexadecimal values.
	err := d.writeHexKeyAs(b, "Interface", "PrivateKey", "private_key")
	if err != nil {
		return nil, err
	}
	err = d.writeHexKeyAs(b, "Peer", "PublicKey", "public_key")
	if err != nil {
		return nil, err
	}
	err = d.writeHexKeyAs(b, "Peer", "PresharedKey", "preshared_key")
	if err != nil {
		return nil, err
	}
	// Add the allowed IP and endpoint information to the buffer.
	_, err = b.WriteString("allowed_ip=0.0.0.0/0\n")
	//_, err = b.WriteString(fmt.Sprintf("allowed_ip=%s\n", d.conf["Peer"]["AllowedIPs"]))
	if err != nil {
		return nil, err
	}
	_, err = b.WriteString(fmt.Sprintf("endpoint=%s\n", d.conf["Peer"]["Endpoint"]))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DialContext establishes a network connection using the WireGuard tunnel if available,
// otherwise, it uses the standard net.Dialer.
func (d *wireGuardDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.tunnel != nil {
		// If the WireGuard tunnel is available, use it to establish the connection.
		return d.tunnel.DialContext(ctx, network, address)
	}
	// If there is no WireGuard tunnel, fall back to the standard net.Dialer.
	return d.standard.DialContext(ctx, network, address)
}

// Dial is a convenience function that calls DialContext with a background context.
func (d *wireGuardDialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}
