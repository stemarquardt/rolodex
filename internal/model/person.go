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

// CreatePerson inserts a new person and returns their id.
func (s *Store) CreatePerson(ctx context.Context, p Person) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO people (first_name, last_name, nickname, location) VALUES (?, ?, ?, ?)
	`, p.FirstName, p.LastName, p.Nickname, p.Location)
	if err != nil {
		return 0, fmt.Errorf("create person: %w", err)
	}
	return res.LastInsertId()
}

// GetPersonDetail loads a person's full profile. Returns (nil, nil) if no
// person with that id exists.
func (s *Store) GetPersonDetail(ctx context.Context, id int64) (*PersonDetail, error) {
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

	return &PersonDetail{
		Person:            p,
		ContactInfo:       contactInfo,
		ImportantDates:    dates,
		Relationships:     rels,
		CircleMemberships: circles,
		Pets:              pets,
		Facts:             facts,
		Notes:             notes,
	}, nil
}
