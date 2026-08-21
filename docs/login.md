  ### 🏢 Organization Admin (org_admin) Implemented & Seeded
  
  We have introduced the Organization Administrator (org_admin) role (Role ID: 6) via database migration 00064_org_admin_role.sql.
  ──────
  ### 🛡️ Role Separation Matrix: Super Admin vs. Org Admin
  
   Capability / Area                                     │        Super Admin (admin)        │       Org Admin (org_admin)       │            Dispatcher            │            Accountant            │              Viewer              │              Driver
  ───────────────────────────────────────────────────────┼───────────────────────────────────┼───────────────────────────────────┼──────────────────────────────────┼──────────────────────────────────┼──────────────────────────────────┼──────────────────────────────────
   User & Team Management (/users)                       │          ✅ Cross-Tenant          │          ✅ Tenant Only           │                ❌                │                ❌                │                ❌                │                ❌
   Fleet & Assets (Vehicles, Drivers, Routes)            │                ✅                 │                ✅                 │                ✅                │                ❌                │               Read               │               Read
   Operations (Bookings, Trips, Dispatches)              │                ✅                 │                ✅                 │                ✅                │                ❌                │               Read               │             Assigned
   Live Tracking & Telemetry (/tracking, /telemetry/*)   │                ✅                 │                ✅                 │                ✅                │                ❌                │               Read               │                ❌
   Geofences & Ops Alerts (/geofences, /alerts)          │                ✅                 │                ✅                 │                ✅                │                ❌                │               Read               │                ❌
   Financials (Invoices, Payments, Kharcha, Settlements) │                ✅                 │                ✅                 │                ❌                │                ✅                │               Read               │                ❌
   GST E-Way Bills & FASTag                              │                ✅                 │                ✅                 │                ❌                │                ✅                │               Read               │                ❌
   Company Settings & Audit Logs                         │                ✅                 │                ✅                 │                ❌                │                ❌                │               Read               │                ❌
   Founder Dashboard & Cross-Tenant Signals (/founder/*) │                ✅                 │           ❌ Restricted           │                ❌                │                ❌                │                ❌                │                ❌
   Platform A/B Experiments Engine (/experiments)        │                ✅                 │           ❌ Restricted           │                ❌                │                ❌                │                ❌                │                ❌
  ──────
  ### 🔑 Complete Test Users Matrix
  
  All accounts are live in transport.db with password Password123!:
  
   Role Name                               │                Role ID                 │ Email                                  │ Password                               │ Primary Purpose
  ─────────────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┼────────────────────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────
   Super Admin (Platform Admin)            │                   1                    │ admin@avandab.com                      │ Password123!                           │ Global platform administrator (Founder signals, Experiments, Cross-tenant settings).
   Org Admin (Organization Admin)          │                   6                    │ orgadmin@avandab.com                   │ Password123!                           │ Organization administrator (Full tenant-scoped team, fleet, asset, financial, and dispatch control).
   Dispatcher                              │                   2                    │ dispatcher@avandab.com                 │ Password123!                           │ Operational dispatch, GPS telemetry, live tracking, trips, bookings.
   Accountant                              │                   3                    │ accountant@avandab.com                 │ Password123!                           │ Invoices, payments, driver settlements, kharcha ledger, GST & FASTag.
   Viewer                                  │                   4                    │ viewer@avandab.com                     │ Password123!                           │ Read-only visibility across operational dashboards.
   Driver                                  │                   5                    │ driver@avandab.com                     │ Password123!                           │ Driver mobile interface, assigned trips, ePOD uploads.
  ──────