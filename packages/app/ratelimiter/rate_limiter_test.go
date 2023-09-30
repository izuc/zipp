package ratelimiter_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/izuc/zipp.foundation/autopeering/peer"
	"github.com/izuc/zipp.foundation/autopeering/peer/service"
	"github.com/izuc/zipp.foundation/crypto/ed25519"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/logger"
	"github.com/izuc/zipp/packages/app/ratelimiter"
)

const (
	defaultTestInterval = time.Minute
	defaultTestLimit    = 3
)

func TestPeerRateLimiter_Count(t *testing.T) {
	t.Parallel()
	prl := newTestRateLimiter(t)
	testPeer := newTestPeer()
	testCount(t, prl, testPeer, defaultTestLimit)
}

func TestPeerRateLimiter_SetBaseLimit(t *testing.T) {
	t.Parallel()
	prl := newTestRateLimiter(t)
	customLimit := 5
	prl.SetBaseLimit(customLimit)
	testPeer := newTestPeer()
	testCount(t, prl, testPeer, customLimit)
}

func TestPeerRateLimiter_ExtendLimit(t *testing.T) {
	t.Parallel()
	prl := newTestRateLimiter(t)
	testPeer := newTestPeer()
	limitExtensionCount := 3
	for i := 0; i < limitExtensionCount; i++ {
		prl.ExtendLimit(testPeer.ID(), 1)
	}
	testCount(t, prl, testPeer, defaultTestLimit+limitExtensionCount)
}

func testCount(t testing.TB, prl *ratelimiter.PeerRateLimiter, testPeer *peer.Peer, testLimit int) {
	activityCount := atomic.NewInt32(0)
	expectedActivity := testLimit + 1
	eventCalled := atomic.NewInt32(0)
	prl.Events.Hit.Hook(func(event *ratelimiter.HitEvent) {
		p := event.Source
		rl := event.RateLimit
		eventCalled.Inc()
		require.Equal(t, int32(expectedActivity), activityCount.Load())
		require.Equal(t, testPeer.ID(), p)
		require.Equal(t, defaultTestInterval, rl.Interval)
		require.Equal(t, testLimit, rl.Limit)
	})
	for i := 0; i < expectedActivity; i++ {
		activityCount.Inc()
		prl.Count(testPeer.ID())
	}
	require.Eventually(t, func() bool { return eventCalled.Load() == 1 }, time.Second, time.Millisecond)
	for i := 0; i < expectedActivity; i++ {
		activityCount.Inc()
		prl.Count(testPeer.ID())
	}
	require.Never(t, func() bool { return eventCalled.Load() > 1 }, time.Second, time.Millisecond)
}

func newTestRateLimiter(t testing.TB) *ratelimiter.PeerRateLimiter {
	prl, err := ratelimiter.NewPeerRateLimiter(defaultTestInterval, defaultTestLimit, logger.NewNopLogger())
	require.NoError(t, err)
	return prl
}

func newTestPeer() *peer.Peer {
	services := service.New()
	services.Update(service.PeeringKey, "tcp4", 0)
	services.Update(service.P2PKey, "tcp4", 0)

	var publicKey ed25519.PublicKey
	copy(publicKey[:], "test peer")

	return peer.NewPeer(identity.New(publicKey), net.IPv4zero, services)
}
