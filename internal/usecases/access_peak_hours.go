package usecases

import (
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sebastiancorrales/gym-go/internal/domain/entities"
	"github.com/sebastiancorrales/gym-go/pkg/timeutil"
)

// HourStat is the traffic of one hour of the day (0–23, gym's wall clock).
type HourStat struct {
	Hour  int     `json:"hour"`
	Total int     `json:"total"`
	Avg   float64 `json:"avg"` // visits per day at this hour
}

// PeakHours is the average occupancy profile of the gym over a window.
type PeakHours struct {
	Days int    `json:"days"`
	From string `json:"from"` // local date where averaging starts
	To   string `json:"to"`
	// ObservedDays is the divisor behind every average: the days from the
	// first recorded entry in the window through today. A gym that started
	// logging two weeks ago must not have its averages diluted by the other
	// sixteen days of a 30-day window.
	ObservedDays int         `json:"observed_days"`
	TotalVisits  int         `json:"total_visits"`
	Hourly       []HourStat  `json:"hourly"`
	Grid         [][]float64 `json:"grid"`        // [weekday 0=Sunday][hour] → avg visits
	WeekdayAvg   []float64   `json:"weekday_avg"` // avg visits per day, by weekday
	TopHours     []HourStat  `json:"top_hours"`
	// BusiestWeekday is nil when there is no data.
	BusiestWeekday *int `json:"busiest_weekday"`
}

// GetPeakHours builds the gym's average hourly traffic over the last `days`
// days, read on the gym's wall clock (loc).
func (uc *AccessUseCase) GetPeakHours(gymID uuid.UUID, days int, loc *time.Location) (*PeakHours, error) {
	if loc == nil {
		loc = time.Local
	}
	today := dateOnly(time.Now().In(loc))
	return uc.GetPeakHoursBetween(gymID, today.AddDate(0, 0, -(days-1)), today, loc)
}

// GetPeakHoursBetween is GetPeakHours over an explicit span of local dates
// [firstDay, lastDay], both inclusive. Only their calendar date is used.
func (uc *AccessUseCase) GetPeakHoursBetween(gymID uuid.UUID, firstDay, lastDay time.Time, loc *time.Location) (*PeakHours, error) {
	if loc == nil {
		loc = time.Local
	}
	firstDay, lastDay = sameDateIn(firstDay, loc), sameDateIn(lastDay, loc)
	entries, err := uc.accessLogRepo.ListGrantedEntries(gymID,
		timeutil.StartOfDay(firstDay, loc), timeutil.EndOfDay(lastDay, loc))
	if err != nil {
		return nil, err
	}
	return buildPeakHours(entries, firstDay, lastDay, loc), nil
}

func buildPeakHours(entries []entities.AccessInstant, windowStart, lastDay time.Time, loc *time.Location) *PeakHours {
	var hourly [24]int
	var grid [7][24]int
	var weekdayTotals [7]int
	firstDay := lastDay
	total := 0

	// A member who scans twice in the same hour — the door did not read the
	// first time, or they stepped out for water — is one visit, not two.
	type visitKey struct {
		user uuid.UUID
		day  time.Time
		hour int
	}
	seen := make(map[visitKey]struct{}, len(entries))

	for _, e := range entries {
		t := e.AccessTime.In(loc)
		day := dateOnly(t)
		if day.Before(windowStart) || day.After(lastDay) {
			continue
		}
		k := visitKey{e.UserID, day, t.Hour()}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}

		if day.Before(firstDay) {
			firstDay = day
		}
		hourly[t.Hour()]++
		grid[t.Weekday()][t.Hour()]++
		weekdayTotals[t.Weekday()]++
		total++
	}

	res := &PeakHours{
		Days:        calendarDays(windowStart, lastDay),
		To:          lastDay.Format("2006-01-02"),
		TotalVisits: total,
		Hourly:      make([]HourStat, 24),
		Grid:        make([][]float64, 7),
		WeekdayAvg:  make([]float64, 7),
		TopHours:    []HourStat{},
	}
	if total == 0 {
		firstDay = windowStart
	}
	res.From = firstDay.Format("2006-01-02")

	// How many times each weekday occurs in [firstDay, lastDay]: the divisor
	// of that weekday's row, so "Monday 6 pm" averages over Mondays only.
	var weekdayCount [7]int
	for d := firstDay; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		weekdayCount[d.Weekday()]++
	}
	res.ObservedDays = calendarDays(firstDay, lastDay)

	for h := 0; h < 24; h++ {
		res.Hourly[h] = HourStat{Hour: h, Total: hourly[h], Avg: avg(hourly[h], res.ObservedDays)}
	}
	for wd := 0; wd < 7; wd++ {
		res.Grid[wd] = make([]float64, 24)
		for h := 0; h < 24; h++ {
			res.Grid[wd][h] = avg(grid[wd][h], weekdayCount[wd])
		}
		res.WeekdayAvg[wd] = avg(weekdayTotals[wd], weekdayCount[wd])
	}

	if total > 0 {
		ranked := make([]HourStat, 0, 24)
		for _, hs := range res.Hourly {
			if hs.Total > 0 {
				ranked = append(ranked, hs)
			}
		}
		sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Total > ranked[j].Total })
		if len(ranked) > 3 {
			ranked = ranked[:3]
		}
		res.TopHours = ranked

		best := 0
		for wd := 1; wd < 7; wd++ {
			if res.WeekdayAvg[wd] > res.WeekdayAvg[best] {
				best = wd
			}
		}
		res.BusiestWeekday = &best
	}
	return res
}

// sameDateIn is midnight in loc of t's own calendar date — it keeps the date
// the caller meant rather than converting the instant.
func sameDateIn(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// calendarDays counts the dates in [from, to], both inclusive. It walks the
// calendar instead of dividing hours, so a DST change cannot shave a day off.
func calendarDays(from, to time.Time) int {
	n := 0
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		n++
	}
	return n
}

// dateOnly is local midnight of t, in t's own location.
func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func avg(n, over int) float64 {
	if over == 0 {
		return 0
	}
	return math.Round(float64(n)/float64(over)*100) / 100
}
