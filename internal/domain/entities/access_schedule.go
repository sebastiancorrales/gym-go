package entities

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// AccessSchedule restricts when a plan's members may come in: "weekdays from
// 12:00 to 15:00", "Friday to Sunday, all day". A plan without one — nil or
// no rules — is unrestricted, which is every plan that existed before this.
//
// It is enforced in two places that must agree: the Hikvision terminal, which
// receives it as a week plan and applies it on its own (see
// hikvision_sync_usecase.go), and the kiosk's RecordEntry. The terminal never
// asks the server, so toggling members on and off from an hourly job would let
// them in until the next pass and fail open with the backend down; pushing
// the schedule into the device is what makes it exact to the minute.
type AccessSchedule struct {
	Rules []ScheduleRule `json:"rules"`
}

// ScheduleRule opens the door on the given weekdays between From and To.
//
// Days uses time.Weekday numbering (0 = Sunday), the same as JavaScript's
// getDay(), so the frontend needs no translation. Times are "HH:MM" in the
// gym's wall clock; To may be "24:00" to mean "until midnight". A span that
// crosses midnight is not supported — write it as two rules.
type ScheduleRule struct {
	Days []int  `json:"days"`
	From string `json:"from"`
	To   string `json:"to"`
}

// MaxSegmentsPerDay is how many separate spans the terminal can hold for one
// weekday of a week plan.
const MaxSegmentsPerDay = 8

// DaySegment is a span within one day, in minutes since midnight: [Start, End).
type DaySegment struct {
	Start int
	End   int
}

var (
	ErrScheduleInvalidDay  = errors.New("día inválido en el horario (0=domingo … 6=sábado)")
	ErrScheduleInvalidTime = errors.New("hora inválida en el horario, se espera HH:MM")
	ErrScheduleEmptySpan   = errors.New("en el horario, la hora de inicio debe ser anterior a la de fin")
	ErrScheduleNoDays      = errors.New("cada franja del horario necesita al menos un día")
	ErrScheduleTooMany     = fmt.Errorf("un mismo día admite como máximo %d franjas", MaxSegmentsPerDay)
)

// IsRestricted reports whether the schedule limits anything at all.
func (s *AccessSchedule) IsRestricted() bool {
	return s != nil && len(s.Rules) > 0
}

// Validate checks the rules and that each weekday fits in the terminal once
// overlapping spans are merged.
func (s *AccessSchedule) Validate() error {
	if !s.IsRestricted() {
		return nil
	}
	for _, rule := range s.Rules {
		if len(rule.Days) == 0 {
			return ErrScheduleNoDays
		}
		for _, d := range rule.Days {
			if d < 0 || d > 6 {
				return ErrScheduleInvalidDay
			}
		}
		from, err := parseClock(rule.From)
		if err != nil {
			return err
		}
		to, err := parseClock(rule.To)
		if err != nil {
			return err
		}
		if from >= to {
			return ErrScheduleEmptySpan
		}
	}
	for _, segments := range s.WeekSegments() {
		if len(segments) > MaxSegmentsPerDay {
			return ErrScheduleTooMany
		}
	}
	return nil
}

// WeekSegments returns, per weekday (indexed by time.Weekday), the sorted and
// merged spans the door is open. Overlapping or touching rules collapse into
// one span, because the terminal only has eight slots per day and some
// firmwares refuse overlapping segments.
//
// Invalid rules are skipped; call Validate first to reject them.
func (s *AccessSchedule) WeekSegments() [7][]DaySegment {
	var week [7][]DaySegment
	if !s.IsRestricted() {
		return week
	}
	for _, rule := range s.Rules {
		from, errFrom := parseClock(rule.From)
		to, errTo := parseClock(rule.To)
		if errFrom != nil || errTo != nil || from >= to {
			continue
		}
		for _, d := range rule.Days {
			if d >= 0 && d <= 6 {
				week[d] = append(week[d], DaySegment{Start: from, End: to})
			}
		}
	}
	for d := range week {
		week[d] = mergeSegments(week[d])
	}
	return week
}

// Allows reports whether the schedule lets somebody in at t, read in t's own
// location — pass a time already in the gym's timezone. An unrestricted
// schedule allows everything.
func (s *AccessSchedule) Allows(t time.Time) bool {
	if !s.IsRestricted() {
		return true
	}
	minute := t.Hour()*60 + t.Minute()
	for _, seg := range s.WeekSegments()[t.Weekday()] {
		if minute >= seg.Start && minute < seg.End {
			return true
		}
	}
	return false
}

func mergeSegments(segments []DaySegment) []DaySegment {
	if len(segments) < 2 {
		return segments
	}
	sort.Slice(segments, func(i, j int) bool { return segments[i].Start < segments[j].Start })
	merged := []DaySegment{segments[0]}
	for _, seg := range segments[1:] {
		last := &merged[len(merged)-1]
		if seg.Start <= last.End {
			if seg.End > last.End {
				last.End = seg.End
			}
			continue
		}
		merged = append(merged, seg)
	}
	return merged
}

// parseClock reads "HH:MM" as minutes since midnight. "24:00" is accepted as
// the end of the day.
func parseClock(raw string) (int, error) {
	var h, m int
	if n, err := fmt.Sscanf(raw, "%d:%d", &h, &m); err != nil || n != 2 || len(raw) != 5 {
		return 0, ErrScheduleInvalidTime
	}
	if h == 24 && m == 0 {
		return 24 * 60, nil
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, ErrScheduleInvalidTime
	}
	return h*60 + m, nil
}
