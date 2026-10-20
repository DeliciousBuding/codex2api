package auth

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/codex2api/database"
)

func newQuotaAdmissionTestStore(t *testing.T, limit int64, guard int, used float64) (*Store, *Account) {
	t.Helper()
	s := NewStore(nil, nil, &database.SystemSettings{
		MaxConcurrency:              int(limit),
		AutoPause5hGuardBandPercent: 5,
		AutoPause5hGuardConcurrency: guard,
	})
	t.Cleanup(s.Stop)
	a := newQuotaAutoPauseTestAccount()
	a.AutoPause5hThreshold = .95
	a.AutoPause7dThreshold = .95
	a.recomputeEffectiveAutoPause(s)
	a.SetUsageSnapshot5h(used, time.Now().Add(time.Hour))
	s.AddAccount(a)
	return s, a
}

func TestQuotaAdmissionRechecksThresholdAtSlotReservation(t *testing.T) {
	for _, window := range []string{"5h", "7d"} {
		t.Run(window, func(t *testing.T) {
			_, a := newQuotaAdmissionTestStore(t, 100, 1, 80)
			_, _, cachedLimit, _, available := a.fastSchedulerSnapshot(100, time.Now())
			if !available || cachedLimit != 100 {
				t.Fatalf("initial candidate: available=%v limit=%d", available, cachedLimit)
			}
			if window == "5h" {
				a.SetUsageSnapshot5h(95, time.Now().Add(time.Hour))
			} else {
				a.SetUsageSnapshot(95, time.Now())
			}
			if reserveOccupiedAccountSlot(a, cachedLimit) {
				releaseOccupiedAccountSlot(a)
				t.Fatal("cached candidate bypassed the newly observed quota threshold")
			}
			if a.GetActiveRequests() != 0 || a.GetOccupiedRequests() != 0 {
				t.Fatal("threshold rejection leaked capacity")
			}
		})
	}
}

func TestQuotaAdmissionRechecksGuardLimitFor100ConcurrentSelectors(t *testing.T) {
	_, a := newQuotaAdmissionTestStore(t, 100, 1, 80)
	_, _, cachedLimit, _, _ := a.fastSchedulerSnapshot(100, time.Now())
	// All candidates were read outside the band before a frame lowered the
	// live cap. Reservations must use that cap, not the cached value of 100.
	a.SetUsageSnapshot5h(91, time.Now().Add(time.Hour))
	start := make(chan struct{})
	var admitted atomic.Int64
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			<-start
			if reserveOccupiedAccountSlot(a, cachedLimit) {
				admitted.Add(1)
			}
		})
	}
	close(start)
	wg.Wait()
	got := admitted.Load()
	for range got {
		releaseOccupiedAccountSlot(a)
	}
	if got != 1 {
		t.Fatalf("guard admitted %d simultaneous stale candidates, want 1", got)
	}
}

func TestQuotaAdmissionDrainWaitsForPostTerminal5hObservation(t *testing.T) {
	s, a := newQuotaAdmissionTestStore(t, 100, 1, 94)
	if !reserveOccupiedAccountSlot(a, 1) {
		t.Fatal("first guarded request was not admitted")
	}
	s.Release(a)
	if !a.QuotaGuardPending() || !a.GetAccountLiveUsageSnapshot().QuotaGuardPending {
		t.Fatal("guard drain did not expose its pending observation state")
	}
	if reserveOccupiedAccountSlot(a, 1) {
		releaseOccupiedAccountSlot(a)
		t.Fatal("released guard slot immediately reused the pre-terminal quota snapshot")
	}
	// Connection-level quota metadata may arrive after the response terminal.
	// It must fence the next admission before any persistence completes.
	at := time.Now()
	s.ApplyUsageFrameObservation(a, at, func() {
		a.SetUsageSnapshot5hAt(95, at.Add(time.Hour), at)
	})
	waitUsageFrameDrain(t, a)
	if a.QuotaGuardPending() {
		t.Fatal("post-terminal observation did not clear the guard drain")
	}
	if reserveOccupiedAccountSlot(a, 1) {
		releaseOccupiedAccountSlot(a)
		t.Fatal("delayed terminal quota frame did not fence the next request")
	}
}

func TestQuotaAdmissionRechecksThresholdBeforeSessionSlotReclaim(t *testing.T) {
	s, a := newQuotaAdmissionTestStore(t, 100, 1, 80)
	s.SetSessionSlotBuffer(time.Minute)
	s.SetSessionSlotBufferEnabled(true)
	if !reserveOccupiedAccountSlot(a, 100) {
		t.Fatal("initial session slot was not admitted")
	}
	s.ReleaseForSession(a, "owner")
	if a.GetActiveRequests() != 0 || a.GetOccupiedRequests() != 1 {
		t.Fatalf("buffered slot counters active=%d occupied=%d, want 0/1", a.GetActiveRequests(), a.GetOccupiedRequests())
	}

	a.SetUsageSnapshot5h(95, time.Now().Add(time.Hour))
	if s.tryReclaimSessionSlot(a, "owner", 100, DispatchPolicyStandard, false, true) {
		releaseOccupiedAccountSlot(a)
		t.Fatal("session slot reclaimed capacity after a newer threshold observation")
	}
	if a.GetActiveRequests() != 0 || a.GetOccupiedRequests() != 1 {
		t.Fatalf("rejected reclaim mutated counters active=%d occupied=%d, want 0/1", a.GetActiveRequests(), a.GetOccupiedRequests())
	}
}

func TestQuotaAdmissionArmsOneBoundedRecoveryProbe(t *testing.T) {
	s, a := newQuotaAdmissionTestStore(t, 100, 1, 94)
	probes := make(chan struct{}, 2)
	s.SetUsageProbeFunc(func(context.Context, *Account) error {
		probes <- struct{}{}
		return nil
	})
	if !reserveOccupiedAccountSlot(a, 1) {
		t.Fatal("guarded request was not admitted")
	}
	s.Release(a)
	select {
	case <-probes:
	case <-time.After(3 * time.Second):
		t.Fatal("guard fence did not arm the bounded usage probe")
	}
	select {
	case <-probes:
		t.Fatal("one guard fence issued more than one recovery probe")
	case <-time.After(100 * time.Millisecond):
	}
}
