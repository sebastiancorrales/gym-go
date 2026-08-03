package entities

import (
	"testing"
	"time"
)

// activeSubscription returns a subscription that started 60 days ago and ends in
// 30. The long history is the point: the old Unfreeze counted days since
// StartDate, so an old subscription got a much bigger (and wronger) extension.
func activeSubscription() *Subscription {
	now := time.Now().UTC().Round(0)
	return &Subscription{
		StartDate: now.AddDate(0, 0, -60),
		EndDate:   now.AddDate(0, 0, 30),
		Status:    SubscriptionStatusActive,
	}
}

// daysApart returns the whole days between two instants, tolerant of the odd
// second of drift while the test runs.
func daysApart(from, to time.Time) int {
	return int((to.Sub(from).Hours() + 1) / 24)
}

func TestFreezeExtendsEndDateByExactlyTheRequestedDays(t *testing.T) {
	sub := activeSubscription()
	originalEnd := sub.EndDate

	sub.Freeze(7, "viaje")

	if sub.Status != SubscriptionStatusFrozen {
		t.Errorf("status = %s, want FROZEN", sub.Status)
	}
	if got := daysApart(originalEnd, sub.EndDate); got != 7 {
		t.Errorf("la fecha de fin se extendió %d días, want 7", got)
	}
	if sub.TotalFreezeDays != 7 {
		t.Errorf("TotalFreezeDays = %d, want 7", sub.TotalFreezeDays)
	}
	if sub.FrozenUntil == nil {
		t.Fatal("FrozenUntil quedó nil")
	}
	if got := daysApart(time.Now().UTC(), *sub.FrozenUntil); got != 7 {
		t.Errorf("FrozenUntil está a %d días, want 7", got)
	}
}

// This is the regression test for the bug: freeze then unfreeze must be worth
// exactly the days actually frozen, never the age of the subscription. The old
// code gave this subscription about 67 extra days instead of 7.
func TestFreezeThenUnfreezeAtTheEndGivesExactlyTheFrozenDays(t *testing.T) {
	sub := activeSubscription()
	originalEnd := sub.EndDate

	sub.Freeze(7, "viaje")

	// Simulate the freeze having run its course.
	elapsed := time.Now().UTC().Add(-1 * time.Minute)
	sub.FrozenUntil = &elapsed

	sub.Unfreeze()

	if sub.Status != SubscriptionStatusActive {
		t.Errorf("status = %s, want ACTIVE", sub.Status)
	}
	if got := daysApart(originalEnd, sub.EndDate); got != 7 {
		t.Errorf("congelar 7 días y descongelar movió la fecha de fin %d días, want 7", got)
	}
	if sub.TotalFreezeDays != 7 {
		t.Errorf("TotalFreezeDays = %d, want 7", sub.TotalFreezeDays)
	}
	if sub.FrozenUntil != nil {
		t.Error("FrozenUntil debería quedar nil tras descongelar")
	}
}

func TestUnfreezeEarlyReclaimsTheUnusedDays(t *testing.T) {
	sub := activeSubscription()
	originalEnd := sub.EndDate

	sub.Freeze(10, "viaje")

	// El socio vuelve tras 4 días: quedan 6 sin usar.
	sixLeft := time.Now().UTC().AddDate(0, 0, 6)
	sub.FrozenUntil = &sixLeft

	sub.Unfreeze()

	if got := daysApart(originalEnd, sub.EndDate); got != 4 {
		t.Errorf("volver a los 4 días de un congelamiento de 10 movió la fecha %d días, want 4", got)
	}
	if sub.TotalFreezeDays != 4 {
		t.Errorf("TotalFreezeDays = %d, want 4", sub.TotalFreezeDays)
	}
}

func TestFreezeIgnoresNonPositiveDays(t *testing.T) {
	for _, days := range []int{0, -5} {
		sub := activeSubscription()
		originalEnd := sub.EndDate

		sub.Freeze(days, "sin sentido")

		if sub.Status != SubscriptionStatusActive {
			t.Errorf("Freeze(%d) cambió el estado a %s", days, sub.Status)
		}
		if !sub.EndDate.Equal(originalEnd) {
			t.Errorf("Freeze(%d) movió la fecha de fin", days)
		}
	}
}

func TestRepeatedFreezesAccumulate(t *testing.T) {
	sub := activeSubscription()
	originalEnd := sub.EndDate

	for _, days := range []int{5, 3} {
		sub.Freeze(days, "varios")
		past := time.Now().UTC().Add(-1 * time.Minute)
		sub.FrozenUntil = &past
		sub.Unfreeze()
	}

	if got := daysApart(originalEnd, sub.EndDate); got != 8 {
		t.Errorf("dos congelamientos de 5 y 3 días movieron la fecha %d días, want 8", got)
	}
	if sub.TotalFreezeDays != 8 {
		t.Errorf("TotalFreezeDays = %d, want 8", sub.TotalFreezeDays)
	}
}

func TestFreezeExpired(t *testing.T) {
	sub := activeSubscription()
	if sub.FreezeExpired() {
		t.Error("una suscripción activa no tiene congelamiento vencido")
	}

	sub.Freeze(7, "viaje")
	if sub.FreezeExpired() {
		t.Error("un congelamiento de 7 días no está vencido hoy")
	}

	past := time.Now().UTC().Add(-1 * time.Hour)
	sub.FrozenUntil = &past
	if !sub.FreezeExpired() {
		t.Error("un congelamiento cuya fecha ya pasó debería estar vencido")
	}
}
