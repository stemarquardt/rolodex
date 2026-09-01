# Personal CRM — Planning Doc

## Motivation

Inspired by [Monica](https://github.com/monicahq/monica), a personal CRM. We like the concept
but not the implementation (PHP/Laravel, built for multi-tenant self-hosted SaaS use with a lot
of generality — vaults, permissions, CardDAV sync, custom field templates — that a single-user
tool doesn't need). Decision: build our own, scoped to personal single-user use, borrowing
Monica's better data-model ideas where they fit.

## Stack

- **Go** backend (`net/http` or `chi`)
- **SQLite** storage (`modernc.org/sqlite`, pure Go, no cgo)
- **Server-rendered HTML** (`html/template` or `templ`) + **htmx** for interactivity — no separate
  TS/React frontend
- **`go:embed`** to bundle templates/assets into a single binary
- Runs as a service on the home Precision server, reachable over Tailscale. No public internet
  exposure planned, so **no auth for v1** (network boundary is the perimeter). Basic auth could be
  added later as a cheap second layer if ever needed.

Why Go over TypeScript: single static binary, no runtime/dependency management, very stable
long-term (matters for an app meant to run quietly for years on a home server with minimal
upkeep). Existing fluency in Go is a bonus, but the deployment story would favor Go even without
that. TypeScript/React would only clearly win if we wanted rich client-side interactivity
(drag-and-drop boards, offline PWA) — not needed for a forms-and-lists personal CRM.

## Data model

### Person
Core entity. First/last/nickname, location (city/region — plain text, no geocoding for v1),
`nudge_enabled` (bool, default true — controls whether this person shows up in "check in with"
nudges on the pull page).

### ContactInfo
`person_id`, `type` (phone/email/address/social handle/etc.), `value`. Generic type+value rows
rather than fixed columns, so new contact methods don't require schema changes. We deliberately
duplicate data that may also live in Google Contacts etc. — the point is owning this data
ourselves rather than depending on an external service.

### Circle
Loose, unordered social grouping — "Family," "Rock climbing friends," "Work friends," "Library
group," etc. `id`, `name`, `description`. Purely for browsing/context, not a structural/relational
concept.

### PersonCircle (join)
`person_id`, `circle_id`, `note` (nullable free text — context for *this specific membership*,
e.g. "Met at the bouldering wall," distinct from general Facts about the person). A person can
belong to multiple circles.

### RelationshipType (seeded, extensible)
`id`, `name`, `name_reverse` (e.g. "parent" ↔ "child"). Seed list to start: parent/child, sibling,
spouse/partner, grandparent/grandchild, close friend, mentor/mentee. More can be added later.

### Relationship
`person_id`, `related_person_id`, `relationship_type_id`. Pairwise, typed, directional ties — this
is where **family structure actually lives**, not in Circles. (Decision: Circles = loose grouping
with no structure between members; Relationships = precise pairwise ties. Conflating "family" into
a Circle would lose the parent/child/sibling distinctions. A "Family" Circle can still exist for
browsing/logistics convenience, but it's optional and separate from the relational truth.)

### ImportantDate
`person_id`, `type` (birthday/anniversary/custom, seeded + extensible), `label`, `month`, `day`,
`year` (nullable — handles "know the birthday, not the year"). Birthday is *not* a hardcoded Person
column — it's just an ImportantDate row like anything else. Reminders for these are computed on
the fly (query for any ImportantDate within the next N days) rather than materializing a row every
year.

**Scope note:** ImportantDate is strictly for things that recur **every year** by calendar date
(birthdays, anniversaries) — nothing else. One-time future dates tied to a person (e.g. "her big
presentation on June 12") are *not* ImportantDates — they're just a `Reminder` with a `due_date`
and no recurrence set (see below). Keeps the two concepts cleanly split: ImportantDate = always
annual, Reminder = everything else (one-time or recurring on any interval).

### Event
Generalizes "visit" (and later could cover trips/gatherings without new tables). `id`, `kind`
(visit/trip/gathering — extensible), `status` (idea → tentative → confirmed → done/cancelled),
`start_date`/`end_date` (nullable), `timeframe_note` (free text like "sometime in fall" for vague
plans with no real date yet), `notes`.

### EventPerson (join)
`event_id`, `person_id`. A visit can involve multiple people (e.g. sister + her husband for New
Year's). No role field needed for v1 (just participants).

### Reminder
The single mechanism for anything with a due date that isn't an annually-recurring ImportantDate —
covers both one-time, expiring nudges ("big presentation on June 12" — fires once, marked done,
drops off Today) and recurring check-ins ("call Grandma every Sunday," "text every 3 months"),
distinguished only by whether `recurrence_interval`/`recurrence_unit` are set. `person_id`
(nullable), `event_id` (nullable, e.g. "confirm dates before the visit"), `due_date`,
`recurrence_interval` (nullable int), `recurrence_unit` (days/weeks/months — e.g. interval=3,
unit=months = "every 3 months"), `note`, `status` (pending/done/snoozed). When a recurring reminder
is completed, compute the next `due_date` rather than building a full RRULE engine.

A reminder tied to a person is just one row queried two ways: filtered by `person_id` it renders as
a "Reminders for this person" section on their profile page; unfiltered (or filtered by due date)
it renders in the global Today/Reminders feed. No separate entity needed for "does this belong on
the person's page."

### Note
Timestamped, freeform log per person. `person_id`, `body`, `created_at`. Doubles as the source for
staleness detection (see below).

### Pet
`person_id` (owner), `name`, `species`, `notes`.

### Fact
Flexible catch-all for details that don't warrant their own column: allergies, kids' names,
coffee order, etc. `person_id`, `label`, `value`.

## The "Today" pull page (core feature)

Pull, not push — no email/SMS/notifications infra for v1, just a page to check. Surfaces:

1. **Upcoming ImportantDates** — birthdays/anniversaries within a 2-week window.
2. **Events needing attention** — tentative visits sitting without confirmed dates ("still
   happening?"), confirmed visits coming up soon.
3. **Due/overdue Reminders** — manual nudges, including recurring check-ins.
4. **Staleness nudges** — people with `nudge_enabled = true` where `MAX(Note.created_at)` is older
   than some threshold (e.g. 60 days) — "haven't logged anything with this person in a while."
   Opt-out is per-person (see `Person.nudge_enabled` above); Circle pages can bulk-toggle this flag
   for their members as a workflow convenience, but the underlying rule is always per-person, to
   avoid ambiguity when someone belongs to multiple circles with different defaults.

## Data ownership / backup

Export-to-JSON (all tables dumped) planned for v1 — cheap to build, gives a concrete backup story
and reinforces the "I own this data" motivation. No scheduled/automatic backups planned yet beyond
"the SQLite file + JSON export exist and can be copied somewhere."

## MVP scope

**In:** Person, ContactInfo, Circle/PersonCircle, RelationshipType/Relationship, ImportantDate,
Event/EventPerson, Reminder, Note, Pet, Fact, the Today pull page, JSON export.

**Deferred:** multi-user/auth, vCard/CardDAV import-sync, tags beyond Circles, avatars/file
uploads, gift/loan tracking, journal/mood tracking, push/SMS notifications, hierarchical or
sub-circles, roles within Events, pet age/birthdate, freeform "thoughts" not tied to a person,
public/external share links (profile pages are only ever viewed over Tailscale, by the owner —
no share feature planned).

## UI review (first prototype pass)

First-pass mockups (Today, People, Person profile, Circles, Visits & events) reviewed against this
model — confirmed buildable with no architecture changes. Deltas folded into the sections above:
`PersonCircle.note`, the ImportantDate-vs-Reminder scope split. Explicitly cut from the mockups:
the "Add a thought" quick-capture flow, pet age tracking, and the profile "Share link" button.

## Status

Data model agreed, reviewed against first UI mockups. Go project scaffolded: SQLite schema (13
tables) + db layer (`internal/db`), model/Store layer (`internal/model`), templ+htmx UI
(`internal/web`), Dockerized (multi-stage build, `docker-compose.yml`, volume-mounted SQLite file).
First working vertical slice — People (list, search, add, detail) — is fully wired end to end and
verified locally via `docker compose up --build`; Today/Circles/Events/Reminders/Notes are stub
pages with real nav entries. Not yet deployed to the Precision server (local verification only).

Next: build out Today (the real aggregation queries), Circles, Visits & events, and Reminders pages
against the same Store pattern; then deploy to the Precision server via `docker compose up -d
--build` over Tailscale.
