Now I have the full picture. Here's the plan.

---

# Owner Onboarding & Landing Plan

## Current state (what blocks owners)

| Blocker | Location |
|---|---|
| Self-register = viewer only, never admin | `internal/handlers/auth.go:82` |
| Company onboarding admin-only, viewer locked out | `internal/handlers/settings.go:25-34` |
| No owner role, no first-user bootstrap | `internal/domain/user/entity.go:46-49` |
| User onboarding template is driver-focused (driving details, govt ID) | `internal/templates/user_onboarding.html` |
| Dashboard = data dump, no setup guide or quick actions | `internal/templates/dashboard.html` |
| No fleet setup wizard — owner must hunt across /vehicles, /drivers, /routes | router `cmd/server/main.go:435-444` |
| No progress tracking — owner doesn't know what's left | none |

## Target personas

| Persona | Role | Lands on | Needs |
|---|---|---|---|
| **Owner** (new) | admin (first user) | Setup wizard | Company → vehicle → driver → route → first booking, in one flow |
| **Owner** (returning) | admin | Owner dashboard | Stats, quick actions, alerts, pending approvals |
| **Dispatcher** | dispatcher | Dashboard | Bookings, trips, assignments |
| **Accountant** | accountant | Dashboard | Invoices, payments, reports |
| **Driver** | viewer (mobile) | Mobile app | Active trip, POD, kharcha |
| **Staff viewer** | viewer | Dashboard | Read-only |

## Phase 1 — Owner can sign up and set up (minimum viable)

### 1.1 First-user bootstrap
- Add `adminExists(ctx) bool` helper in `UserService` (count users with role_id=1).
- In `Register` (`internal/handlers/auth.go:62`):
  - If `!adminExists && company.CompanyName == ""` → create as **admin**, redirect to `/onboard/owner`.
  - Else → viewer (current behavior).
- Add `owner` as alias of admin in role checks (`internal/domain/user/entity.go`) — no new role ID, just clearer naming. Keep `admin` as the stored role; expose `owner` label in UI.

### 1.2 Owner setup wizard (`/onboard/owner`)
New route group inside protected, admin-only. Single page with stepped sections (not separate pages — fewer clicks).

```
/onboard/owner  (GET render, POST save-all)
```

**Steps shown as progress checklist:**

1. **Company** — reuse `company_onboard.html` fields (name, email, phone, address, GST, currency, logo). Prefill currency=INR, GST rate=18%.
2. **First vehicle** — registration number, type, capacity, fuel type. Inline form.
3. **First driver** — name, phone, license number. Inline form.
4. **First route** — source, destination, distance, estimated hours, base fare. Inline form.
5. **Invite team** (optional, skip) — email + role dropdown for dispatcher/accountant.
6. **Done** → redirect to `/dashboard`.

Each step saves independently via existing service methods (`Vehicles.Create`, `Drivers.Create`, `Routes.Create`). Mark step complete in `company_settings.onboarding_state` JSON column.

### 1.3 Onboarding state model
Add column to `company_settings`:

```sql
ALTER TABLE company_settings ADD COLUMN onboarding_state TEXT DEFAULT '{}';
```

```go
type OnboardingState struct {
    Company   bool `json:"company"`
    Vehicle   bool `json:"vehicle"`
    Driver    bool `json:"driver"`
    Route     bool `json:"route"`
    Team      bool `json:"team"`
    Completed bool `json:"completed"`
}
```

`DashboardService.GetDashboardData` returns `OnboardingState` so dashboard can show checklist.

### 1.4 Owner dashboard changes (`internal/templates/dashboard.html`)
When `OnboardingState.Completed == false`, show setup card at top:

```
┌─ Get started ─────────────────────────┐
│ ✓ Company details        [Edit]       │
│ ✓ Add a vehicle          [Edit]       │
│ ⬜ Add a driver           [Add]        │
│ ⬜ Create a route         [Add]        │
│ ⬜ Invite your team       [Invite]     │
│ [Resume setup →]                      │
└───────────────────────────────────────┘
```

When complete, show current dashboard (stats + tables). Add a **Quick actions** row for admin:

```
[+ Booking] [+ Trip] [+ Invoice] [+ Driver] [+ Vehicle] [+ Route]
```

### 1.5 Fix user onboarding template
Replace `user_onboarding.html` (currently driver-focused) with role-aware content:
- **Owner/admin** → skip (goes to company wizard).
- **Driver** → keep current (license, govt ID, bank details).
- **Dispatcher/accountant** → just phone + name.

In `Login` (`internal/handlers/auth.go:201-208`), only redirect to `/user/onboard` if phone missing AND role is driver/viewer. Admin goes to company wizard if company not set.

## Phase 2 — Solo-owner mode (reduce friction)

When `adminExists == false` or user count == 1:
- Auto-grant all permissions to the single user (already admin).
- Hide **Users** menu, **Audit logs**, **Integrations** from sidebar until team exists.
- Hide **Settings → Permissions** complexity.
- Show a "You're flying solo" banner with "Invite your first teammate" CTA.

This keeps the UI simple for a one-person owner and reveals advanced features as they grow.

## Phase 3 — Sensible defaults & bulk import

### Defaults on company creation
| Setting | Default |
|---|---|
| Currency | INR |
| GST enabled | true |
| GST rate | 18 |
| Booking prefix | BK |
| Trip prefix | TR |
| Invoice prefix | INV |
| Financial year | current |

### Bulk import (owner has existing fleet)
- CSV upload at `/vehicles/import` and `/drivers/import`.
- Columns: vehicle = reg_number, type, capacity, fuel; driver = name, phone, license.
- Dry-run preview, then commit.
- Reuse `FileService.UploadFile` for the CSV, parse in a new `ImportService`.

## Phase 4 — Owner-specific views

### Owner dashboard widgets (beyond current stats)
- **Revenue this month** vs last month (delta %).
- **Outstanding invoices** total + "Send reminders" button.
- **Driver settlements pending** count.
- **Fleet utilization** (% vehicles on trip).
- **Compliance alerts** (licenses expiring, insurance due).
- **Recent approvals** queue (kharcha, POD).

### Owner-only menu
```
Dashboard
Bookings → Trips → Invoices → Payments
Drivers → Vehicles → Routes → Customers
Kharcha (approvals)
Reports
Team (users)         ← only if >1 user
Settings
```

## Phase 5 — Mobile owner view
Owner checks status from phone. Current mobile app is driver-focused. Add an owner mode:
- Read-only dashboard (stats, active trips, alerts).
- Approve kharcha / POD from phone.
- No editing — keep simple.

Reuse existing API (`/api/v1/*`) with admin token.

## Data model changes summary

```sql
-- company_settings
ALTER TABLE company_settings ADD COLUMN onboarding_state TEXT DEFAULT '{}';

-- users: no schema change; owner = admin role
-- Optional: add owner_profile table later for business details
```

## Code changes map

| File | Change |
|---|---|
| `internal/handlers/auth.go:62-108` | First-user bootstrap → admin + redirect to wizard |
| `internal/handlers/auth.go:201-208` | Role-aware post-login redirect |
| `internal/handlers/settings.go` | New `OwnerOnboard` handler (wizard GET/POST) |
| `internal/service/company_service.go` | `GetOnboardingState`, `UpdateOnboardingState` |
| `internal/service/dashboard_service.go` | Include `OnboardingState` in `DashboardData` |
| `internal/templates/owner_onboard.html` | New wizard template |
| `internal/templates/dashboard.html` | Setup checklist card + quick actions |
| `internal/templates/user_onboarding.html` | Role-aware content |
| `internal/templates/layout.html` | Hide advanced menu items in solo mode |
| `db/migrations/00041_onboarding_state.sql` | Add column |
| `cmd/server/main.go:423` | Mount `/onboard/owner` route |

## What NOT to build now
- Multi-tenant isolation (single-tenant works).
- New `owner` role ID (use admin).
- Payments setup in wizard (Razorpay optional, not blocking).
- Email verification (out of scope; rate-limit registration instead).
- Advanced RBAC editor (auto-grant admin to first user; add roles when team grows).

## Rollout order

1. **First-user bootstrap + company wizard** (Phase 1) — unblocks owners today.
2. **Dashboard checklist** (Phase 1.4) — owners know what's left.
3. **Solo mode** (Phase 2) — simpler UI for one-person ops.
4. **Defaults** (Phase 3) — less typing.
5. **Owner widgets** (Phase 4) — value after setup.
6. **Bulk import** (Phase 3) — for owners migrating from spreadsheets.
7. **Mobile owner view** (Phase 5) — later.

## Success metric
A new transport owner can: sign up → set company → add 1 vehicle → add 1 driver → create 1 route → create 1 booking → see it on dashboard, **in under 5 minutes, from a phone**, without reading docs.

---

Want me to start implementing Phase 1 (first-user bootstrap + owner wizard skeleton)?