# Design Brief: Avandab.com

## Register
Product — the interface is an instrument. Authenticated web UI for operators managing transport logistics daily. Brand/landing page (`home.html`) is separate — handled under `voice` register.

## Users and Context
- **Primary user**: Logistics dispatchers and fleet managers. Arrive under pressure: drivers need assignments, trips are departing, vehicles are idle or broken. Need fast confirmation over pretty decoration.
- **Secondary user**: Fleet owners checking daily stats, revenue, availability.
- **Context**: Desktop-first workflows (1024px+), with critical mobile support for quick status checks and confirmations on warehouse floors.

## Product Purpose
Multi-Vehicle Transport Management System (MVTMS), branded "FlyFleet" on the marketing site. Manages bookings → trips → invoices → payments with full CRUD across drivers, vehicles, routes, customers, and users. The web UI is server-rendered Go templates with Datastar-powered partial refreshes. REST API exists separately for integrations.

## Work Patterns
- **Monitor**: Dashboard with stat cards (trip counts, revenue, availability) and activity feed
- **Operate**: Booking lifecycle (create → confirm → complete → cancel), trip assignment, payment recording
- **Compare**: Tables with sorting, filtering, pagination — vehicles, drivers, routes, customers, bookings, invoices, payments
- **Configure**: Company settings, profile updates, route management

## Artifact
Transport bookings and trips — concrete domain objects with route, pickup date, vehicle type, passengers, cargo weight, price, and status. Status flows: pending → confirmed → completed, or cancelled.

## Evidence of Trust
- Real-time status badges with color coding
- Tabular numbers on all financial/duration fields for clean column alignment
- Consistent table structure across list and detail views
- Flash messages for action confirmation
- Keyboard focus rings on all interactive elements

## Voice
Utility-forward. Confident but never decorative. Copy names the actual action: "Confirm", "Cancel", "Complete", "Create Trip", "New Booking". No exclamation points. Sentence case. Buttons use verb + object.

## Anti-References
- Generic SaaS admin panels (cream/purple, rounded-everything, card grids as decoration)
- Dark-themed dashboards (this is daylight office work, not midnight Ops)
- Generic Tailwind boilerplate (blue-600 buttons, gray-50 surfaces with no tinted neutrals)
- Material Design shadows and elevations without purpose

## Design Principles
1. Operators who open this screen daily should move without thinking
2. Tables are the primary surface — optimize for scanning speed and column stability
3. Hierarchy must be obvious at a glance: page title > section > subsection > body > caption
4. Every interactive element must have 44x44px hit area
5. Form elements must not trigger iOS zoom (1rem minimum font on mobile)
6. All numeric data must use tabular-nums for vertical alignment
7. Motion serves function: state changes, feedback, spatial orientation — never decoration

## Accessibility Expectations
- 3:1 contrast minimum for all text and UI elements
- Focus rings on every interactive element (2-3px width, offset)
- Table headers properly scoped
- `prefers-reduced-motion` respected on all animations
- `prefers-color-scheme` supported (dark mode toggle to be added)
- `viewport-fit=cover` with `safe-*` padding utilities for notched screens

## Visual Foundation
- **Fonts**: Plus Jakarta Sans (body/system UI), Outfit (landing headings only)
- **Type scale**: Major third (~1.25 ratio) — body 14px, section 20px, page title 24px, stat 28px
- **Line heights**: Body 1.5, headings 1.25-1.3, captions 1.0
- **Color**: Brand teal (T500-T800), status colors (blue/orange/green/red/purple/indigo/yellow), tinted gray surfaces
- **Spacing**: 4px grid base — 1/4/9 rhythm system
- **Breakpoints**: sm (640px), md (768px), lg (1024px)

## Component Rules
- **Buttons**: 44x44px minimum touch target, `min-h-[44px]`, focus ring, active scale feedback
- **Tables**: `divide-y` row separators, `text-xs font-semibold` headers, `text-sm` body, `tabular-nums` on numeric columns
- **Forms**: `text-base` font (no shrink on mobile), 44px min-height, focus ring with `ring-brand-500`
- **Status badges**: `text-xs rounded-full`, color per status (pending=gray, confirmed=blue, completed=green, cancelled=red)
- **Cards**: Used only for genuinely scannable data units (dashboard stats), not decorative containers
