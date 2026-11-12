package statsd

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUDPWriterWritesToDestination(t *testing.T) {
	srv, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer srv.Close()

	w, err := newUDPWriter(srv.LocalAddr().String(), 0, 0)
	require.NoError(t, err)
	defer w.Close()

	_, err = w.Write([]byte("ping"))
	require.NoError(t, err)

	buf := make([]byte, 16)
	srv.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _, err := srv.ReadFrom(buf)
	require.NoError(t, err)
	assert.Equal(t, "ping", string(buf[:n]))
}

func TestUDPWriterRefreshesDestination(t *testing.T) {
	srv, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer srv.Close()

	w, err := newUDPWriter(srv.LocalAddr().String(), 0, 10*time.Millisecond)
	require.NoError(t, err)
	defer w.Close()

	// Simulate a stale destination (e.g. the Agent moved to a new IP): the
	// refresh loop must re-resolve the original address and recover.
	stale := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
	w.dst.set(stale)

	assert.Eventually(t, func() bool {
		return w.dst.get().String() == srv.LocalAddr().String()
	}, 2*time.Second, 5*time.Millisecond)
}

func TestUDPWriterCloseIsIdempotent(t *testing.T) {
	w, err := newUDPWriter("127.0.0.1:8125", 0, time.Minute)
	require.NoError(t, err)
	assert.NoError(t, w.Close())
	assert.NotPanics(t, func() { w.Close() })
}

func TestUDPAddrRefreshRateOption(t *testing.T) {
	o, err := resolveOptions([]Option{WithUDPAddrRefreshRate(time.Minute)})
	require.NoError(t, err)
	assert.Equal(t, time.Minute, o.udpAddrRefreshRate)

	o, err = resolveOptions(nil)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), o.udpAddrRefreshRate)
}
