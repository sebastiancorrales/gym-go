package hikvision

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// The terminal restricts *when* a person may enter through a chain of three
// records, all addressed by number:
//
//	person.RightPlan.planTemplateNo → UserRightPlanTemplate.weekPlanNo → UserRightWeekPlanCfg
//
// A week plan holds, per weekday, up to eight time segments. A plan template
// points at one week plan (and optionally a holiday group, unused here). A
// person lists the templates that apply to them; template 1 is the device's
// built-in all-day template, which is what everyone had before schedules
// existed.
//
// gym-go uses the same number for a template and the week plan it points at
// — the plan's AccessSlot — so there is no mapping table to keep in step.

// TimeSegment is a span within one day, in minutes since midnight: [Start, End).
type TimeSegment struct {
	Start int
	End   int
}

// WeekSchedule is a week plan: the segments the door is open, per weekday,
// indexed like time.Weekday (0 = Sunday).
type WeekSchedule [7][]TimeSegment

// segmentsPerDay is how many segment slots a week plan has for each day.
// Every slot is written on every save — the unused ones disabled — so a
// segment removed from the schedule cannot linger on the device.
const segmentsPerDay = 8

// maxTemplateNameBytes is the length the device accepts for templateName.
const maxTemplateNameBytes = 32

var isapiWeekdays = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// deviceClock renders minutes since midnight as the device's "HH:MM:SS".
// The end of the day is written as 23:59:59 rather than 24:00:00, which is a
// valid clock reading on every firmware.
func deviceClock(minutes int) string {
	if minutes >= 24*60 {
		return "23:59:59"
	}
	return fmt.Sprintf("%02d:%02d:00", minutes/60, minutes%60)
}

// weekPlanPayload builds the UserRightWeekPlanCfg body: all seven days, all
// eight slots each.
func weekPlanPayload(schedule WeekSchedule) map[string]any {
	entries := make([]map[string]any, 0, 7*segmentsPerDay)
	for day, name := range isapiWeekdays {
		for slot := 0; slot < segmentsPerDay; slot++ {
			entry := map[string]any{
				"week":   name,
				"id":     slot + 1,
				"enable": false,
				"TimeSegment": map[string]any{
					"beginTime": "00:00:00",
					"endTime":   "00:00:00",
				},
			}
			if slot < len(schedule[day]) {
				seg := schedule[day][slot]
				entry["enable"] = true
				entry["TimeSegment"] = map[string]any{
					"beginTime": deviceClock(seg.Start),
					"endTime":   deviceClock(seg.End),
				}
			}
			entries = append(entries, entry)
		}
	}
	return map[string]any{
		"UserRightWeekPlanCfg": map[string]any{
			"enable":      true,
			"WeekPlanCfg": entries,
		},
	}
}

func templatePayload(name string, weekPlanNo int) map[string]any {
	return map[string]any{
		"UserRightPlanTemplate": map[string]any{
			"enable":         true,
			"templateName":   truncateBytes(name, maxTemplateNameBytes),
			"weekPlanNo":     weekPlanNo,
			"holidayGroupNo": "",
		},
	}
}

// SetAccessSchedule writes week plan N and plan template N pointing at it.
// Afterwards, a person whose RightPlan names template N may only enter
// during the schedule's segments.
//
// The week plan goes first: a template pointing at a week plan that is not
// yet written would, for a moment, restrict its members to whatever that
// slot held before.
func (c *Client) SetAccessSchedule(slot int, name string, schedule WeekSchedule) error {
	if slot < 2 {
		return fmt.Errorf("access schedule slot %d is reserved: template 1 is the device's all-day template", slot)
	}
	for day, segments := range schedule {
		if len(segments) > segmentsPerDay {
			return fmt.Errorf("%s has %d segments; the device holds at most %d", isapiWeekdays[day], len(segments), segmentsPerDay)
		}
	}

	weekPlan, err := json.Marshal(weekPlanPayload(schedule))
	if err != nil {
		return err
	}
	if err := c.jsonStatusCall("PUT",
		fmt.Sprintf("/ISAPI/AccessControl/UserRightWeekPlanCfg/%d?format=json", slot),
		weekPlan, "set week plan"); err != nil {
		return err
	}

	template, err := json.Marshal(templatePayload(name, slot))
	if err != nil {
		return err
	}
	return c.jsonStatusCall("PUT",
		fmt.Sprintf("/ISAPI/AccessControl/UserRightPlanTemplate/%d?format=json", slot),
		template, "set plan template")
}

func truncateBytes(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	// Cut on a rune boundary: half an accented letter is invalid UTF-8 and
	// the device rejects the whole body for it.
	cut := 0
	for i := range s {
		if i > limit {
			break
		}
		cut = i
	}
	if cut == 0 || !utf8.ValidString(s[:cut]) {
		return ""
	}
	return s[:cut]
}
