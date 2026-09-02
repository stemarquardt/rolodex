package templates

import (
	"fmt"
	"strings"

	"rolodex/internal/model"
)

func initials(p model.Person) string {
	out := ""
	if r := []rune(p.FirstName); len(r) > 0 {
		out += string(r[0])
	}
	if r := []rune(p.LastName); len(r) > 0 {
		out += string(r[0])
	}
	if out == "" {
		if r := []rune(p.Nickname); len(r) > 0 {
			out = string(r[0])
		}
	}
	return strings.ToUpper(out)
}

var monthNames = []string{
	"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

func formatImportantDate(d model.ImportantDate) string {
	name := "?"
	if d.Month >= 1 && d.Month <= 12 {
		name = monthNames[d.Month]
	}
	if d.Year.Valid {
		return fmt.Sprintf("%s %d, %d", name, d.Day, d.Year.Int64)
	}
	return fmt.Sprintf("%s %d", name, d.Day)
}

func personURL(id int64) string {
	return fmt.Sprintf("/people/%d", id)
}

func personHeaderID(id int64) string  { return fmt.Sprintf("person-header-%d", id) }
func personHeaderURL(id int64) string { return fmt.Sprintf("/people/%d/header", id) }
func personEditURL(id int64) string   { return fmt.Sprintf("/people/%d/edit", id) }
func personNudgeURL(id int64) string  { return fmt.Sprintf("/people/%d/nudge", id) }

func contactInfoListID(personID int64) string { return fmt.Sprintf("contact-info-list-%d", personID) }
func contactInfoCreateURL(personID int64) string {
	return fmt.Sprintf("/people/%d/contact-info", personID)
}
func contactInfoDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/contact-info/%d", personID, id)
}

func importantDateListID(personID int64) string {
	return fmt.Sprintf("important-dates-list-%d", personID)
}
func importantDateCreateURL(personID int64) string {
	return fmt.Sprintf("/people/%d/important-dates", personID)
}
func importantDateDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/important-dates/%d", personID, id)
}

func relationshipListID(personID int64) string { return fmt.Sprintf("relationships-list-%d", personID) }
func relationshipCreateURL(personID int64) string {
	return fmt.Sprintf("/people/%d/relationships", personID)
}
func relationshipDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/relationships/%d", personID, id)
}

func circleListID(personID int64) string { return fmt.Sprintf("circles-list-%d", personID) }
func circleCreateURL(personID int64) string {
	return fmt.Sprintf("/people/%d/circles", personID)
}
func circleDeleteURL(personID, circleID int64) string {
	return fmt.Sprintf("/people/%d/circles/%d", personID, circleID)
}

func circleURL(id int64) string       { return fmt.Sprintf("/circles/%d", id) }
func circleHeaderID(id int64) string  { return fmt.Sprintf("circle-header-%d", id) }
func circleHeaderURL(id int64) string { return fmt.Sprintf("/circles/%d/header", id) }
func circleEditURL(id int64) string   { return fmt.Sprintf("/circles/%d/edit", id) }

func circleMemberListID(circleID int64) string { return fmt.Sprintf("circle-members-%d", circleID) }
func circleMemberCreateURL(circleID int64) string {
	return fmt.Sprintf("/circles/%d/members", circleID)
}
func circleMemberDeleteURL(circleID, personID int64) string {
	return fmt.Sprintf("/circles/%d/members/%d", circleID, personID)
}

func petListID(personID int64) string    { return fmt.Sprintf("pets-list-%d", personID) }
func petCreateURL(personID int64) string { return fmt.Sprintf("/people/%d/pets", personID) }
func petDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/pets/%d", personID, id)
}

func factListID(personID int64) string    { return fmt.Sprintf("facts-list-%d", personID) }
func factCreateURL(personID int64) string { return fmt.Sprintf("/people/%d/facts", personID) }
func factDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/facts/%d", personID, id)
}

func noteListID(personID int64) string    { return fmt.Sprintf("notes-list-%d", personID) }
func noteCreateURL(personID int64) string { return fmt.Sprintf("/people/%d/notes", personID) }
func noteDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/notes/%d", personID, id)
}

func idStr(id int64) string { return fmt.Sprintf("%d", id) }

func relativeDays(daysUntil int) string {
	switch {
	case daysUntil == 0:
		return "today"
	case daysUntil == 1:
		return "tomorrow"
	default:
		return fmt.Sprintf("in %d days", daysUntil)
	}
}

func eventPeopleNames(people []model.Person) string {
	names := make([]string, len(people))
	for i, p := range people {
		names[i] = p.FullName()
	}
	return strings.Join(names, ", ")
}

// eventLabel picks the best available display name for an event: its title
// if set, otherwise the participants' names, otherwise just its kind.
func eventLabel(e model.Event) string {
	if e.Title != "" {
		return e.Title
	}
	if names := eventPeopleNames(e.People); names != "" {
		return names
	}
	return e.Kind
}

// eventWhen renders an event's timing: its start date if set, otherwise its
// freeform timeframe note, otherwise a placeholder for still-undated events.
func eventWhen(e model.Event) string {
	if e.StartDate.Valid && e.StartDate.String != "" {
		return e.StartDate.String
	}
	if e.TimeframeNote != "" {
		return e.TimeframeNote
	}
	return "no date yet"
}

const eventGroupsID = "event-groups"

// filterEventsByStatus returns the subset of events with the given status
// (case-insensitive — status is a free-text field, so "Tentative" and
// "tentative" should group the same way), preserving order (events is
// already sorted by ListEvents).
func filterEventsByStatus(events []model.Event, status string) []model.Event {
	var out []model.Event
	for _, e := range events {
		if strings.EqualFold(e.Status, status) {
			out = append(out, e)
		}
	}
	return out
}

// filterEventsPast returns events that are done or cancelled, collapsed into
// one "Past" group on the Events page.
func filterEventsPast(events []model.Event) []model.Event {
	var out []model.Event
	for _, e := range events {
		if strings.EqualFold(e.Status, "done") || strings.EqualFold(e.Status, "cancelled") {
			out = append(out, e)
		}
	}
	return out
}

// filterEventsOtherStatus returns events whose status doesn't match any of
// the known buckets above (idea/tentative/confirmed/done/cancelled,
// case-insensitive) — a catch-all so a typo'd or unexpected status value
// still shows up somewhere on the Events page instead of silently
// disappearing from the list while still being reachable by direct URL.
func filterEventsOtherStatus(events []model.Event) []model.Event {
	known := []string{"idea", "tentative", "confirmed", "done", "cancelled"}
	var out []model.Event
	for _, e := range events {
		matched := false
		for _, k := range known {
			if strings.EqualFold(e.Status, k) {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, e)
		}
	}
	return out
}

func eventURL(id int64) string       { return fmt.Sprintf("/events/%d", id) }
func eventHeaderID(id int64) string  { return fmt.Sprintf("event-header-%d", id) }
func eventHeaderURL(id int64) string { return fmt.Sprintf("/events/%d/header", id) }
func eventEditURL(id int64) string   { return fmt.Sprintf("/events/%d/edit", id) }

func eventParticipantListID(eventID int64) string {
	return fmt.Sprintf("event-participants-%d", eventID)
}
func eventParticipantCreateURL(eventID int64) string {
	return fmt.Sprintf("/events/%d/participants", eventID)
}
func eventParticipantDeleteURL(eventID, personID int64) string {
	return fmt.Sprintf("/events/%d/participants/%d", eventID, personID)
}

const reminderSectionsID = "reminder-sections"

// filterRemindersByStatus splits ListReminders/ListRemindersForPerson's
// combined pending+done result into one bucket or the other.
func filterRemindersByStatus(reminders []model.Reminder, done bool) []model.Reminder {
	var out []model.Reminder
	for _, r := range reminders {
		if (r.Status == "done") == done {
			out = append(out, r)
		}
	}
	return out
}

func reminderDeleteURL(id int64) string   { return fmt.Sprintf("/reminders/%d", id) }
func reminderCompleteURL(id int64) string { return fmt.Sprintf("/reminders/%d/complete", id) }

func personReminderListID(personID int64) string { return fmt.Sprintf("reminders-list-%d", personID) }
func personReminderCreateURL(personID int64) string {
	return fmt.Sprintf("/people/%d/reminders", personID)
}
func personReminderDeleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/reminders/%d", personID, id)
}
func personReminderCompleteURL(personID, id int64) string {
	return fmt.Sprintf("/people/%d/reminders/%d/complete", personID, id)
}

const settingsOptionsBodyID = "settings-options-body"

func settingsOptionCreateURL() string         { return "/settings/options" }
func settingsOptionDeleteURL(id int64) string { return fmt.Sprintf("/settings/options/%d", id) }

const settingsRelationshipTypesBodyID = "settings-relationship-types-body"

func settingsRelationshipTypeCreateURL() string { return "/settings/relationship-types" }
func settingsRelationshipTypeDeleteURL(id int64) string {
	return fmt.Sprintf("/settings/relationship-types/%d", id)
}

// containsOptionValue reports whether value matches one of options
// (case-insensitive — these back free-text-originated columns like
// Event.Status, so a pre-existing row's value might differ only in case
// from the managed list).
func containsOptionValue(options []model.OptionValue, value string) bool {
	for _, o := range options {
		if strings.EqualFold(o.Value, value) {
			return true
		}
	}
	return false
}
