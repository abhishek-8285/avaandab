# Feature Tech-Spec Template

Every feature spec in this folder MUST contain these sections, in order.
Goal: a beginner developer (or an AI agent) can implement the feature end-to-end
without further clarification.

```
# <FEATURE> — Implementation Spec v1
Status: ready
Depends-on: <spec ids / migration numbers from 00-migration-ownership-index.md>
Migration owner: db/migrations/00XXX_*.sql   (reserve exactly one in the index)

0. Verified ground truth      file:line facts + grep proofs of current state
1. Overview / goal            1-para + explicit non-goals
2. API contract               every route: method, path, auth/permission,
                              request JSON, response JSON, error codes
3. DB contract                goose migration, column-level DDL, FKs, indexes,
                              DOWN, seed rows (RBAC, company_config)
4. UI                         pages, templates, partials, JS assets, RBAC resources
5. Business logic             algorithms, state machines, formulas, pseudo-code
6. Config / env               table: var | default | purpose | which package reads
7. Tests                      unit + HTTP/integration cases, fixtures,
                              coverage gate, pass-before-merge checklist
8. Future / GPS-provider      third-party adapters, AIS-140/VLT, OSM fallback, scaling
9. Edge cases                 enumerated, with handling
10. Phased rollout            build order
11. Open items / VERIFY       decisions that MUST be resolved before coding
12. File list                 create / modify, exact paths
```

## Conventions
- All JSON examples MUST be valid and show real field names.
- All SQL MUST be copy-pasteable goose migrations with Up + Down.
- Reference real files via `path:line`.
- Integrations (GSTN, EWB, FASTag, Accounting, telematics) MUST be built as a
  real adapter interface with a config-flagged MOCK behind it (no external
  creds required to run/test). Real NIC/provider calls only fire when enabled.
- Multi-tenancy: never trust client `tenant_id`; derive from `auth.ContextUser`.
- GPS/telematics: define a `TelematicsProvider` interface; own hardware (MQTT,
  IMEI-auth) is primary, third-party (LocoNav/WheelsEye/MapMyIndia/TelaBit/OBD)
  are pluggable adapters behind the same interface.
