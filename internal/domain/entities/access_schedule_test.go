package entities

import (
	"testing"
	"time"
)

// Un plan sin horario es el de siempre: todo el que existía antes de esta
// función tiene que seguir entrando a cualquier hora.
func TestAccessSchedule_SinReglasNoRestringe(t *testing.T) {
	var nothing *AccessSchedule
	if !nothing.Allows(time.Now()) || !(&AccessSchedule{}).Allows(time.Now()) {
		t.Fatal("un plan sin horario no puede restringir la entrada")
	}
}

// El plan de estudiantes: entre semana de 12:00 a 15:00. El fin es exclusivo
// — a las 15:00 ya no se entra — igual que en el terminal.
func TestAccessSchedule_FranjaEstudiantes(t *testing.T) {
	schedule := &AccessSchedule{Rules: []ScheduleRule{{Days: []int{1, 2, 3, 4, 5}, From: "12:00", To: "15:00"}}}
	if err := schedule.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	// 2026-09-21 es lunes.
	at := func(day, hour, minute int) time.Time {
		return time.Date(2026, time.September, day, hour, minute, 0, 0, time.UTC)
	}
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"lunes 11:59", at(21, 11, 59), false},
		{"lunes 12:00", at(21, 12, 0), true},
		{"viernes 14:59", at(25, 14, 59), true},
		{"viernes 15:00", at(25, 15, 0), false},
		{"sábado 13:00", at(26, 13, 0), false},
	}
	for _, c := range cases {
		if got := schedule.Allows(c.t); got != c.want {
			t.Errorf("%s: Allows = %v, want %v", c.name, got, c.want)
		}
	}
}

// "Viernes a domingo, todo el día" se escribe hasta las 24:00.
func TestAccessSchedule_FinDeSemanaCompleto(t *testing.T) {
	schedule := &AccessSchedule{Rules: []ScheduleRule{{Days: []int{5, 6, 0}, From: "00:00", To: "24:00"}}}
	if err := schedule.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	sundayNight := time.Date(2026, time.September, 27, 23, 59, 0, 0, time.UTC)
	thursday := time.Date(2026, time.September, 24, 10, 0, 0, 0, time.UTC)
	if !schedule.Allows(sundayNight) {
		t.Error("el domingo a las 23:59 debería poder entrar")
	}
	if schedule.Allows(thursday) {
		t.Error("el jueves no debería poder entrar")
	}
}

// El terminal tiene ocho franjas por día; las que se solapan se funden para
// no gastar ranuras (y porque hay firmwares que rechazan solapes).
func TestAccessSchedule_FranjasSolapadasSeFunden(t *testing.T) {
	schedule := &AccessSchedule{Rules: []ScheduleRule{
		{Days: []int{1}, From: "06:00", To: "09:00"},
		{Days: []int{1}, From: "08:00", To: "10:00"},
		{Days: []int{1}, From: "18:00", To: "20:00"},
	}}
	got := schedule.WeekSegments()[time.Monday]
	want := []DaySegment{{6 * 60, 10 * 60}, {18 * 60, 20 * 60}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("segmentos = %v, want %v", got, want)
	}
}

func TestAccessSchedule_RechazaReglasInvalidas(t *testing.T) {
	cases := map[string]ScheduleRule{
		"sin días":         {Days: nil, From: "12:00", To: "15:00"},
		"día 7":            {Days: []int{7}, From: "12:00", To: "15:00"},
		"hora mal":         {Days: []int{1}, From: "12", To: "15:00"},
		"25:00":            {Days: []int{1}, From: "12:00", To: "25:00"},
		"cruza medianoche": {Days: []int{1}, From: "22:00", To: "02:00"},
	}
	for name, rule := range cases {
		if err := (&AccessSchedule{Rules: []ScheduleRule{rule}}).Validate(); err == nil {
			t.Errorf("%s: se aceptó una regla inválida", name)
		}
	}

	tooMany := &AccessSchedule{}
	for h := 0; h < MaxSegmentsPerDay+1; h++ {
		tooMany.Rules = append(tooMany.Rules, ScheduleRule{
			Days: []int{1}, From: clock(h * 2), To: clock(h*2 + 1),
		})
	}
	if err := tooMany.Validate(); err != ErrScheduleTooMany {
		t.Errorf("nueve franjas en un día: err = %v, want ErrScheduleTooMany", err)
	}
}

func clock(hour int) string {
	return time.Date(0, 1, 1, hour, 0, 0, 0, time.UTC).Format("15:04")
}
