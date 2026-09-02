package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Person struct {
	ID           int64
	FirstName    string
	LastName     string
	Nickname     string
	Location     string
	NudgeEnabled bool
	Circles      []string // populated by ListPeople only
	LastContact  string   // populated by ListPeople only; empty if no notes yet
}

func (p Person) FullName() string {
	name := strings.TrimSpace(p.FirstName + " " + p.LastName)
	if name == "" {
		return p.Nickname
	}
	return name
}

// PersonDetail is the full profile view: a Person plus everything attached
// to them across the other tables.
type PersonDetail struct {
	Person
	ContactInfo       []ContactInfo
	ImportantDates    []ImportantDate
	Relationships     []RelationshipView
	CircleMemberships []CircleMembership
	Pets              []Pet
	Facts             []Fact
	Notes             []Note
	Reminders         []Reminder
}

// ListPeople returns all people, each annotated with their circle tags and
// most recent note timestamp. search filters by name (case-insensitive
// substring); pass "" for no filter.
func (s *Store) ListPeople(ctx context.Context, search string) ([]Person, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.first_name, p.last_name, p.nickname, p.location, p.nudge_enabled,
		       COALESCE((SELECT MAX(n.created_at) FROM notes n WHERE n.person_id = p.id), '')
		FROM people p
		WHERE (? = '' OR p.first_name LIKE '%' || ? || '%' OR p.last_name LIKE '%' || ? || '%'
		       OR p.nickname LIKE '%' || ? || '%')
		ORDER BY p.last_name, p.first_name
	`, search, search, search, search)
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		var nudge int
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &p.Location, &nudge, &p.LastContact); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		p.NudgeEnabled = nudge != 0
		people = append(people, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(people) == 0 {
		return people, nil
	}

	byID := make(map[int64]*Person, len(people))
	for i := range people {
		byID[people[i].ID] = &people[i]
	}

	circleRows, err := s.db.QueryContext(ctx, `
		SELECT pc.person_id, c.name
		FROM person_circles pc
		JOIN circles c ON c.id = pc.circle_id
		ORDER BY c.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list person circles: %w", err)
	}
	defer circleRows.Close()
	for circleRows.Next() {
		var personID int64
		var circleName string
		if err := circleRows.Scan(&personID, &circleName); err != nil {
			return nil, fmt.Errorf("scan person circle: %w", err)
		}
		if p, ok := byID[personID]; ok {
			p.Circles = append(p.Circles, circleName)
		}
	}
	if err := circleRows.Err(); err != nil {
		return nil, err
	}

	return people, nil
}

// CreatePerson inserts a new person and returns their id. NudgeEnabled
// defaults to false (opt-in) unless the caller explicitly sets it.
func (s *Store) CreatePerson(ctx context.Context, p Person) (int64, error) {
	nudge := 0
	if p.NudgeEnabled {
		nudge = 1
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO people (first_name, last_name, nickname, location, nudge_enabled) VALUES (?, ?, ?, ?, ?)
	`, p.FirstName, p.LastName, p.Nickname, p.Location, nudge)
	if err != nil {
		return 0, fmt.Errorf("create person: %w", err)
	}
	return res.LastInsertId()
}

// UpdatePerson updates a person's core fields (not their sub-collections).
func (s *Store) UpdatePerson(ctx context.Context, p Person) error {
	nudge := 0
	if p.NudgeEnabled {
		nudge = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE people SET first_name = ?, last_name = ?, nickname = ?, location = ?, nudge_enabled = ?
		WHERE id = ?
	`, p.FirstName, p.LastName, p.Nickname, p.Location, nudge, p.ID)
	if err != nil {
		return fmt.Errorf("update person: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update person rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetNudgeEnabled toggles just the check-in-nudge flag, deliberately not
// going through UpdatePerson (which overwrites every core field from a full
// Person struct) — this lets the profile page's checkbox write immediately
// on click without needing the rest of the Edit form's fields.
func (s *Store) SetNudgeEnabled(ctx context.Context, personID int64, enabled bool) error {
	nudge := 0
	if enabled {
		nudge = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE people SET nudge_enabled = ? WHERE id = ?`, nudge, personID)
	if err != nil {
		return fmt.Errorf("set nudge enabled: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set nudge enabled rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeletePerson removes a person and (via ON DELETE CASCADE) everything
// attached to them: contact info, important dates, relationships, circle
// memberships, pets, facts, notes, and reminders/events that reference them.
func (s *Store) DeletePerson(ctx context.Context, id int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM people WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete person: %w", err)
	}
	return nil
}

// GetPerson loads a person's core fields only (no sub-collections). Returns
// (nil, nil) if no person with that id exists. Use this for lightweight
// operations (edit-form prefill, header re-render) that don't need the full
// profile.
func (s *Store) GetPerson(ctx context.Context, id int64) (*Person, error) {
	var p Person
	var nudge int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, nickname, location, nudge_enabled
		FROM people WHERE id = ?
	`, id).Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &p.Location, &nudge)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get person: %w", err)
	}
	p.NudgeEnabled = nudge != 0
	return &p, nil
}

// FindPersonByName looks up a person by an exact case-insensitive match on
// both first and last name. Returns (nil, nil) if no match exists. Used by
// the contacts importer (internal/importer) to skip re-creating a person on
// a repeat import run, rather than the fuzzier OR-across-fields matching
// ListPeople's search does.
func (s *Store) FindPersonByName(ctx context.Context, firstName, lastName string) (*Person, error) {
	var p Person
	var nudge int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, first_name, last_name, nickname, location, nudge_enabled
		FROM people WHERE first_name = ? COLLATE NOCASE AND last_name = ? COLLATE NOCASE
		LIMIT 1
	`, firstName, lastName).Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &p.Location, &nudge)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find person by name: %w", err)
	}
	p.NudgeEnabled = nudge != 0
	return &p, nil
}

// StalePerson is a nudge-enabled person flagged for the Today page's
// staleness section because no Note has been logged for them recently.
type StalePerson struct {
	PersonID   int64
	PersonName string
	DaysStale  int // -1 if no note has ever been logged for this person
}

// ListStalePeople returns nudge-enabled people who haven't had a note logged
// in more than thresholdDays days (or ever), most-stale (or never-contacted)
// first.
func (s *Store) ListStalePeople(ctx context.Context, thresholdDays int) ([]StalePerson, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, first_name, last_name, nickname, last_contact,
		       CASE WHEN last_contact = '' THEN -1
		            ELSE CAST(julianday('now') - julianday(last_contact) AS INTEGER)
		       END AS days_stale
		FROM (
			SELECT p.id, p.first_name, p.last_name, p.nickname,
			       COALESCE(MAX(n.created_at), '') AS last_contact
			FROM people p
			LEFT JOIN notes n ON n.person_id = p.id
			WHERE p.nudge_enabled = 1
			GROUP BY p.id
		)
		WHERE last_contact = '' OR julianday('now') - julianday(last_contact) > ?
		ORDER BY (last_contact = '') DESC, last_contact ASC
	`, thresholdDays)
	if err != nil {
		return nil, fmt.Errorf("list stale people: %w", err)
	}
	defer rows.Close()

	var stale []StalePerson
	for rows.Next() {
		var p Person
		var lastContact string
		var sp StalePerson
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Nickname, &lastContact, &sp.DaysStale); err != nil {
			return nil, fmt.Errorf("scan stale person: %w", err)
		}
		sp.PersonID = p.ID
		sp.PersonName = p.FullName()
		stale = append(stale, sp)
	}
	return stale, rows.Err()
}

// GetPersonDetail loads a person's full profile. Returns (nil, nil) if no
// person with that id exists.
func (s *Store) GetPersonDetail(ctx context.Context, id int64) (*PersonDetail, error) {
	p, err := s.GetPerson(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}

	contactInfo, err := s.ListContactInfo(ctx, id)
	if err != nil {
		return nil, err
	}
	dates, err := s.ListImportantDates(ctx, id)
	if err != nil {
		return nil, err
	}
	rels, err := s.ListRelationships(ctx, id)
	if err != nil {
		return nil, err
	}
	circles, err := s.ListCircleMemberships(ctx, id)
	if err != nil {
		return nil, err
	}
	pets, err := s.ListPets(ctx, id)
	if err != nil {
		return nil, err
	}
	facts, err := s.ListFacts(ctx, id)
	if err != nil {
		return nil, err
	}
	notes, err := s.ListNotes(ctx, id)
	if err != nil {
		return nil, err
	}
	reminders, err := s.ListRemindersForPerson(ctx, id)
	if err != nil {
		return nil, err
	}

	return &PersonDetail{
		Person:            *p,
		ContactInfo:       contactInfo,
		ImportantDates:    dates,
		Relationships:     rels,
		CircleMemberships: circles,
		Pets:              pets,
		Facts:             facts,
		Reminders:         reminders,
		Notes:             notes,
	}, nil
}
