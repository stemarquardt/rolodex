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
