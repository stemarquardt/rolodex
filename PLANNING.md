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
- Runs as a service on the home server, reachable over Tailscale. No public internet
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

**Deferred:** multi-user/auth, ongoing vCard/CardDAV *sync*, tags beyond Circles, avatars/file
uploads, gift/loan tracking, journal/mood tracking, push/SMS notifications, hierarchical or
sub-circles, roles within Events, pet age/birthdate, freeform "thoughts" not tied to a person,
public/external share links (profile pages are only ever viewed over Tailscale, by the owner —
no share feature planned).

## UI review (first prototype pass)

First-pass mockups (Today, People, Person profile, Circles, Visits & events) reviewed against this
model — confirmed buildable with no architecture changes. Deltas folded into the sections above:
`PersonCircle.note`, the ImportantDate-vs-Reminder scope split. Explicitly cut from the mockups:
the "Add a thought" quick-capture flow, pet age tracking, and the profile "Share link" button.

## People CRUD build plan

Building out full create/edit for Person and everything attached to them, since People is the
foundation every other page (Today, Circles, Events, Reminders) will read from — want the
create/edit flows solid before layering more on top.

**Scope decision:** full edit-in-place for the Person's core fields (name/nickname/location/nudge
toggle), but for the repeating sub-collections (contact info, important dates, pets, facts,
circle memberships, relationships) — **add + delete only, no in-place row edit**. If a detail is
wrong, delete the row and re-add it. This keeps the number of new forms manageable; true per-row
editing can be added later if deleting-and-re-adding turns out to be annoying in practice.

**Bug fix bundled in:** `ListRelationships` currently only queries rows where `person_id = ?`, so
a relationship only shows up on the person who "owns" the row, not on the related person's profile
via the reverse label. Since relationships are stored as a single directional row (see
`RelationshipType.name_reverse` above), the query needs to `UNION` both directions — rows where
this person is `person_id` (shown using `rt.name`) and rows where this person is
`related_person_id` (shown using `rt.name_reverse`, with the *other* person as the one displayed).

**New Store methods** (`internal/model`):
- `Person`: `UpdatePerson`, `DeletePerson`
- `ContactInfo`: `CreateContactInfo`, `DeleteContactInfo`
- `ImportantDate`: `CreateImportantDate`, `DeleteImportantDate`
- `Relationship`: `CreateRelationship`, `DeleteRelationship`, `ListRelationshipTypes`; fix
  `ListRelationships` per above
- `Circle`: `ListCircles`, `GetOrCreateCircleByName`, `AddPersonToCircle`, `RemovePersonFromCircle`
  (full Circle list/detail pages are still out of scope — this is just enough plumbing to attach a
  person to a circle, typing a new or existing circle name into a `<datalist>`-backed input)
- `Pet`: `CreatePet`, `DeletePet`
- `Fact`: `CreateFact`, `DeleteFact`
- `Note`: `CreateNote`, `DeleteNote`

**htmx pattern (uniform across all sub-collections):** each profile panel has an inline "add" form
below its list. Every mutating endpoint (create or delete) returns the *refreshed list fragment*
for that sub-collection, targeting a `div id="{entity}-{personID}"` wrapper with `hx-swap="outerHTML"`
— so create and delete both "just re-render the list," no per-row optimistic-update logic needed.
Person core-field editing is the one exception: `GET /people/{id}/edit` swaps the header panel into
an edit form; `PUT /people/{id}` saves and swaps back to view mode.

**New routes** (`internal/web`):
```
PUT    /people/{id}                              update core fields
DELETE /people/{id}                              delete person (redirects to /people)
GET    /people/{id}/edit                         header panel -> edit form

POST   /people/{id}/contact-info
DELETE /people/{id}/contact-info/{ciID}
POST   /people/{id}/important-dates
DELETE /people/{id}/important-dates/{dateID}
POST   /people/{id}/pets
DELETE /people/{id}/pets/{petID}
POST   /people/{id}/facts
DELETE /people/{id}/facts/{factID}
POST   /people/{id}/notes
DELETE /people/{id}/notes/{noteID}
POST   /people/{id}/circles                      body: circle_name (+ optional note)
DELETE /people/{id}/circles/{circleID}
POST   /people/{id}/relationships                body: related_person_id, relationship_type_id
DELETE /people/{id}/relationships/{relID}
```

**Templates:** split `person_detail.templ`'s panels into standalone renderable fragments (so the
same fragment fn is used for both the full-page initial render and the htmx create/delete
responses), plus an editable version of the header panel and small add-forms per panel.

**Verification:** Store-level Go tests for every new method (mirroring the existing
`model_test.go` pattern), plus a manual end-to-end pass: create a person, add one of each
sub-entity (contact info, important date, pet, fact, note, circle membership, relationship to a
second person), confirm the reverse relationship shows correctly on the second person's profile,
delete a couple of rows, edit the core fields, then delete the person entirely.

## Status

Data model agreed, reviewed against first UI mockups. Go project scaffolded: SQLite schema (13
tables) + db layer (`internal/db`), model/Store layer (`internal/model`), templ+htmx UI
(`internal/web`), Dockerized (multi-stage build, `docker-compose.yml`, volume-mounted SQLite file).
First working vertical slice — People (list, search, add, detail) — is fully wired end to end and
verified locally via `docker compose up --build`; Today/Circles/Events/Reminders/Notes are stub
pages with real nav entries. Not yet deployed to the home server (local verification only).

People is now a complete create/edit/delete experience: core fields are editable in place, and
every sub-collection (contact info, important dates, relationships, circles, pets, facts, notes)
supports add + delete through the uniform htmx list-fragment pattern described above. The
relationship bidirectionality bug is fixed and covered by a test (`TestRelationshipBidirectionalDisplay`).
Full manual verification pass (create two people, attach one of everything, confirm the reverse
relationship label, edit core fields, delete rows, delete a person, confirm cascade cleanup,
confirm persistence across a container restart) completed successfully via
`docker compose up --build`. All Go tests pass (`internal/db`, `internal/model`).

Today is now built against the real aggregation queries described above: upcoming ImportantDates
(next 14 days, next annual occurrence resolved in Go since the recurrence is calendar-based, not
something SQLite date math handles cleanly), Events needing attention (tentative regardless of
date, or confirmed within 14 days), due/overdue Reminders, and staleness nudges (nudge-enabled
people with no Note in 60+ days, computed via a `julianday` SQL query). Covered by Store-level
tests (`internal/model/today_test.go`) and verified manually via `docker compose up --build` with
seeded events/reminders/notes exercising all four sections, including a person dropping out of the
staleness list after a new Note is added. No CRUD UI for Events/Reminders yet — Today is read-only
against those tables until those pages are built.

Circles now has full list/detail pages: a master list (`/circles`) with member counts and an
inline "+ New circle" form, and a detail page (`/circles/{id}`) with edit-in-place core fields
(name/description, mirroring Person's header edit pattern) and add/delete-only membership
management (mirroring every other Person sub-collection). Reuses the `AddPersonToCircle`/
`RemovePersonFromCircle` plumbing built during the People CRUD pass, so a membership added from
either the circle's page or the person's page shows up correctly on both. Covered by
`internal/model/circle_test.go`; verified manually via `docker compose up --build` (create circle,
add/remove members, edit fields, delete with cascade confirmed on both the circles list and the
affected person's own profile).

Known minor UX nit, not fixed: after adding/removing a circle member via htmx, the "add member"
`<select>` isn't live-refiltered (it only excludes existing members on a full page load) — matches
the same tolerance already present in `RelationshipsSection` on the person profile page, and is
harmless here since `AddPersonToCircle` is an upsert (re-selecting an already-added person just
updates their note, no duplicate).

Visits & Events and Reminders are now built out, closing the last gap Today had (those sections
were previously only exercised via raw SQL in tests). Events (`/events`) has a list page grouping
by status (Idea / Tentative / Confirmed & upcoming, with done/cancelled collapsed into "Past") and
a detail page with edit-in-place core fields plus add/delete-only participant management (reusing
the `AddPersonToCircle`-style plumbing pattern, now `AddPersonToEvent`/`RemovePersonFromEvent`).
`kind` and `status` are free-text-with-datalist, matching how `ContactInfo.type` already works, so
new kinds/statuses need no schema change. Reminders (`/reminders`) has a create form (optional
person/event link, optional recurrence) plus Pending/Done sections; a matching Reminders section
was added to the person profile page. The one new interaction beyond add/delete is **complete**:
`CompleteReminder` marks a one-off reminder `done`, but advances a recurring reminder's `due_date`
by its interval/unit and leaves it `pending` — implemented and covered by
`internal/model/reminder_test.go` (`TestCompleteReminderRecurring`, one case per unit: days/weeks/
months). Also covered: `internal/model/event_test.go` (create/update/delete, participant add/remove
including idempotent re-add, `ListEvents` group-ordering). Verified manually via
`docker compose up --build`: created an idea-status event (showed in the right group), added it a
title/date/participants, confirmed it surfaced in Today's "Visits needing attention"; created both a
one-off and a recurring person-linked reminder, confirmed both display correctly on `/reminders` and
the person's own profile, completed both and confirmed the one-off/recurring behavior split exactly
as designed; deleted an event with participants and confirmed cascade cleanup.

The `/notes` nav stub has been removed (along with `templates.ComingSoon` and `internal/web/stub.go`,
now unused) — Notes are logged inline per-person via `NotesSection` on the profile page, so a
separate Notes page was redundant scope, not a deferred feature.

A one-time Google Contacts import tool now exists: `cmd/import` reads a Google Takeout vCard (.vcf)
export and creates People plus attached ContactInfo/ImportantDates/Circles/Notes/Facts, via a new
`internal/importer` package that composes existing Store methods (no schema changes). Deliberately
*not* a persistent sync feature or web UI — a plain CLI (`go run ./cmd/import -dry-run
contacts.vcf`, then without `-dry-run` for real), safe to re-run against a fresh export later since
people are matched by exact first+last name (`Store.FindPersonByName`, new) and skipped rather than
duplicated on a repeat run. Google's Labels become Circles for free via the existing
`GetOrCreateCircleByName`/`AddPersonToCircle`. Covered by `internal/importer/importer_test.go`
(field mapping including both no-year birthday forms — vCard's `--MM-DD` and Google's `1604`
placeholder-year convention — plus the idempotent-rerun and dry-run-writes-nothing cases); verified
manually end-to-end (dry-run preview, real import, re-run no-op, browsed the imported profiles in
the UI). Must be run with the docker compose server stopped (or before starting it) — SQLite is
single-writer and `internal/db.Open` caps the pool at 1 connection.

The importer was revised after the user ran it against a real Google Takeout export, which turned
out to be structured differently than assumed: Takeout produces a `Contacts/` directory with one
subfolder per Google Contacts Label (plus device-sync sources), not a single flat `.vcf`. Fixes:
- `internal/importer.ResolveSource` accepts either that directory or a specific `.vcf` file — for a
  directory, it uses `All Contacts/All Contacts.vcf`, which turned out to be the superset: every
  labeled/saved card's `CATEGORIES` field lists *all* its label memberships, so the per-label files
  are redundant. `cmd/import`'s usage was updated to point at the directory.
- `Import` now defaults to only importing cards with a `CATEGORIES` field at all (i.e. contacts the
  user actually saved/labeled) — real Takeout exports include hundreds of auto-collected contacts
  (one-off email senders, company accounts) alongside the ones worth having a profile for, and
  those don't belong in a curated personal CRM by default. `-all` opts into everyone. New
  `Summary.CardsSkippedUnlabeled` field reports the split.
- The system-label filter previously assumed Google marks its own pseudo-labels with a `*` prefix
  (`* myContacts`) — the real export has no such marker (plain `myContacts`, `starred`). Fixed to an
  exact-match denylist.
- Confirmed via reading `go-vcard`'s decoder source that line folding, mixed CRLF/LF, Google's
  `item1.`-grouped-property convention, and repeated `TYPE=` params are already handled correctly
  by the library — those were not bugs, just needed verification against real data.
- `.gitignore` now excludes `/Contacts/`, `/Takeout/`, and `*.vcf` — a real Takeout export sitting
  in the repo working directory during testing must never be tracked.

The importer is deployed and working against the user's real data; the Precision server is running
the app over the LAN (rebranded from "Personal CRM" to "Rolodex" in the sidebar/tab title/
breadcrumbs, with the `rolodex_mini.png` icon, and Docker's published port defaults to `8090` now,
overridable via `ROLODEX_PORT`, to avoid clashing with other local dev servers on `8080`). Also
fixed along the way: events with a differently-cased status (e.g. `Tentative` typed via the edit
form) were silently disappearing from `/events` and from Today's "Visits needing attention" —
both used exact-case string matching against a free-text column. Fixed to case-insensitive matching
throughout, plus a catch-all "Other" status group on the Events page so no status value can ever
make an event vanish from the list again, regardless of what ends up stored.

**Managed option lists + Settings page.** The four fields that used to be
`<input list="..."><datalist>` (Event kind/status, ContactInfo/ImportantDate type) rendered as
Chrome's native autofill-style popup, which isn't stylable and read as "not a real dropdown."
Replaced with actual `<select>` elements backed by a new `option_values` table
(`internal/model/optionvalue.go`, seeded in `internal/db/seed.go`) and a `/settings` page to
add/remove each category's values — the same admin-managed-lookup-table shape `RelationshipType`
already used, generalized to a single shared table since these four have no extra columns. Circle
name stays free text (it's how a brand-new Circle gets created inline via
`GetOrCreateCircleByName`, which a fixed `<select>` can't do). `option_values` is deliberately not a
foreign key from `events`/`contact_info`/`important_dates` — deleting a managed option only removes
it from the dropdown going forward, it doesn't touch rows that already used it. `EventHeaderEdit`'s
edit-form selects handle the case where a row's current value isn't in the managed list (deleted
since, or a pre-existing odd value) by adding it as a one-off preserved `<option>` rather than
silently changing it on next save.

The default `option_values` (Visit/Trip/Gathering, Idea/Tentative/.../Cancelled, Mobile/Email/...,
Birthday/Anniversary/Custom) are now capitalized rather than lowercase. `internal/db/seed.go` gained
`renameLegacyOptionValues`, run unconditionally on every `db.Open` (unlike the seed inserts, which
only fire against an empty table) — it renames an exact match against the old lowercase defaults to
the new capitalized form, so an already-deployed database (like the home server's) picks up the fix
on next restart with no manual migration step. Custom values a user added themselves are untouched
(only exact matches against the specific old default strings get renamed). Note this only fixes the
*managed list* — People/Event/etc. rows that already stored a lowercase value (e.g. an event saved
with `status = "tentative"` before this change) keep displaying that lowercase value verbatim on
their tag until that record is edited and saved again, at which point `EventHeaderEdit`'s
case-insensitive select match picks the capitalized canonical option and writes that back — a
gradual, non-destructive migration path rather than a bulk rewrite of existing data.

Two more day-to-day polish fixes: the Important Date `label` field is now optional, defaulting to
the type when left blank (typing "Birthday" right after picking "Birthday" from the type dropdown
was pure friction — label still matters for telling apart multiple same-type dates, or naming what
a "Custom" date actually is); and the month field on that same form is now a name dropdown
(January..December) instead of a bare 1-12 number input, matching how the date already displays
once saved.

**`nudge_enabled` flipped from opt-out to opt-in.** With a large bulk-imported contact list, an
opt-out default made Today's "People going quiet" show almost the entire People list, which
defeats its purpose as a nudge — the signal only means something for people you've deliberately
chosen to track. `internal/db/schema.sql`'s default changed to 0, `Store.CreatePerson`
(`internal/model/person.go`) now explicitly writes `NudgeEnabled` into the INSERT instead of
silently relying on the SQL default regardless of what the caller passed (a latent bug — the field
was previously ignored on create), and the importer no longer opts imported contacts in.

Existing data needed a real one-time migration, not just a new default — added
`schema_migrations` (`internal/db/schema.sql`) plus `applyOneTimeMigrations`
(`internal/db/seed.go`), a general mechanism for migrations that (unlike
`renameLegacyOptionValues`) aren't safely re-runnable by content-matching alone. The nudge flip is
registered there: clears `nudge_enabled` for every existing person, runs exactly once ever via the
tracking table, so manually re-opting someone in later survives future restarts instead of being
silently reset by a re-run. Verified manually by simulating the home server's actual pre-migration
state (existing people with `nudge_enabled=1`, no migration row) and confirming a restart both
fixes them and — critically — leaves a subsequently re-enabled person alone on the *next* restart.

Cleared most of the rough-edge backlog noted above:
- **Relationship types are now manageable from Settings** — `internal/model/relationship.go` gained
  `CreateRelationshipType`/`DeleteRelationshipType`, `/settings` gained a 5th panel. One real gotcha
  caught during implementation: unlike `option_values`, `relationships.relationship_type_id` is a
  genuine foreign key with `ON DELETE CASCADE` in the schema, so deleting a type also deletes every
  Relationship using it — the delete button's confirm text says so explicitly, and this is covered
  by a dedicated test (`TestDeleteRelationshipTypeCascadesToRelationships`) precisely because the
  first-drafted code comment claimed the opposite (no cascade, matching `option_values`) before the
  schema was re-checked.
- **The top-bar search box now works** — wrapped in a plain `GET` form to `/people?q=`, reusing the
  People page's existing search rather than building anything new. No JS, no ⌘K shortcut (out of
  scope, would need real JS this project has avoided everywhere else).
- **Notes (person profile) and Event notes are now `<textarea>`s** instead of single-line inputs —
  no model/handler changes needed, `r.FormValue` reads a textarea's value the same way.
- **The nudge checkbox is now live on the profile view** — no more Edit-click-Save round trip for
  one flag. `Store.SetNudgeEnabled` is a dedicated single-column update (deliberately not routed
  through `UpdatePerson`, which would overwrite name/nickname/location from a partial payload); the
  checkbox posts via `hx-put` with `hx-vals="js:{nudge_enabled: event.target.checked}"` to
  correctly send `false` on uncheck (bare checkboxes don't submit at all when unchecked). No
  confirmation dialog — a boolean preference toggle doesn't warrant one, `hx-confirm` in this app is
  reserved for deletes.

Circle name's datalist UI is the one item from that list intentionally left alone — already decided
as a kept tradeoff (free-text is how a brand-new Circle gets created inline), not a bug.

Added a "Mark as caught up" button to the person profile header (`internal/web/personnote.go`'s
`PersonCheckIn`) — logs a fixed "Checked in" `Note` with no user input required, a low-friction way
to reset the staleness clock (still purely `MAX(Note.created_at)`-driven, see `ListStalePeople`)
for contact you had but don't have anything specific to write down. Deliberately *not* a separate
"last contacted" field or a broadened definition of "contact" that includes any field edit — the
latter was considered and rejected: editing a record (e.g. correcting a phone number a mutual
friend mentioned) isn't the same signal as actually seeing someone, and would produce false
staleness resets during routine data cleanup. The button's `hx-target` points at the Notes list
(not the header it lives in) so the new entry is immediately visible as confirmation something
happened. Verified manually: a nudge-enabled person showing in Today's "People going quiet"
disappeared from that list immediately after clicking the button, with the "Checked in" entry
visible in their Notes section.

Next: nothing left from the original MVP scope or the rough-edge backlog — the app is deployed and
in use. Future work is whatever surfaces from actually using it day to day, same as every fix this
session.
