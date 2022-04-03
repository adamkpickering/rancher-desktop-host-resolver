/*
Copyright © 2022 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package commands

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	port := randomPort()
	cmd := runHostResovler(t, []string{"-a", "127.0.0.1", "-t", port, "-u", port})
	defer cmd.Process.Kill()

	t.Logf("Checking for TCP port is running on %v", port)
	tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if tcpListener != nil {
		defer tcpListener.Close()
	}
	require.Errorf(t, err, "host-resolver is not listening on TCP port %s", port)

	t.Logf("Checking for UDP port is running on %v", port)
	udpListener, err := net.Listen("udp", fmt.Sprintf(":%s", port))
	if udpListener != nil {
		defer udpListener.Close()
	}
	require.Errorf(t, err, "host-resolver is not listening on UDP port %s", port)

	output := netstat(t)
	require.Contains(t, string(output), fmt.Sprintf("%v/host-resolver", cmd.Process.Pid), "Expected the same Pid")
}

func TestQueryStaticHosts(t *testing.T) {
	port := randomPort()
	cmd := runHostResovler(t, []string{"-a", "127.0.0.1", "-t", port, "-u", port, "-c", "host.rd.test=111.111.111.111,host2.rd.test=222.222.222.222"})
	defer cmd.Process.Kill()

	t.Logf("Checking for TCP port on %s", port)
	addrs, err := dnsLookup(t, port, "tcp", "host.rd.test")
	require.NoError(t, err, "Lookup IP failed")

	expected := []net.IP{net.IPv4(111, 111, 111, 111)}
	require.ElementsMatch(t, ipToString(addrs), ipToString(expected))

	t.Logf("Checking for UDP port on %s", port)
	addrs, err = dnsLookup(t, port, "udp", "host2.rd.test")
	require.NoError(t, err, "Lookup IP failed")

	expected = []net.IP{net.IPv4(222, 222, 222, 222)}
	require.ElementsMatch(t, ipToString(addrs), ipToString(expected))
}

func TestQueryUpstreamServer(t *testing.T) {
	port := randomPort()
	cmd := runHostResovler(t, []string{"-a", "127.0.0.1", "-t", port, "-u", port, "-s", "[8.8.8.8]"})
	defer cmd.Process.Kill()

	t.Logf("Resolving via upstream server on [TCP] --> %s", port)
	addrs, err := dnsLookup(t, port, "tcp", "google.ca")
	require.NoError(t, err, "Lookup IP failed")
	require.True(t, len(addrs) > 0, true, "Expect at least an address")

	t.Logf("Resolving via upstream server on [UDP] --> %s", port)
	addrs, err = dnsLookup(t, port, "udp", "google.ca")
	require.NoError(t, err, "Lookup IP failed")
	require.True(t, len(addrs) > 0, true, "Expect at least an address")
}

func runHostResovler(t *testing.T, args []string) *exec.Cmd {
	// add run command to the tip
	args = append([]string{"run"}, args...)
	// add background process to the tail
	args = append(args, "&")

	cmd := exec.Command("/app/host-resolver", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	require.NoError(t, err, "host-resolver run failed")
	// little bit of pause is needed for the process to start
	// since cmd.Run() doesn't work in this situation :{
	time.Sleep(time.Second * 1)
	return cmd
}

func ipToString(ips []net.IP) (out []string) {
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

func dnsLookup(t *testing.T, resolverPort, resolverProtocol, domain string) ([]net.IP, error) {
	resolver := net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{}
			return dialer.DialContext(ctx, resolverProtocol, fmt.Sprintf(":%s", resolverPort))
		},
	}
	t.Logf("[DNS] lookup on :%s and %s -> %s", resolverPort, resolverProtocol, domain)
	// 10s timeout should be adequate
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return resolver.LookupIP(ctx, "ip4", domain)
}

func randomPort() string {
	return fmt.Sprint(rand.Intn(65535-54) + 54)
}

func netstat(t *testing.T) []byte {
	out, err := exec.Command("netstat", "-nlp").Output()
	require.NoError(t, err, "netstat -nlp")
	t.Logf("%s\n", out)
	return out
}
