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
