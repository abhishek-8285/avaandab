# Avandab Platform Architecture & Future Scope Roadmap

> **System Overview**: Product Roadmap for Avandab (MVTMS - Fleet & Transport Management System), mapping current capabilities against India-specific enterprise logistics expansion modules.

---

## 🏛️ Layer 0: Core Foundation (Currently Built)

The baseline operational control center providing core fleet management, trip scheduling, and client visibility.

- **Driver Management**: Driver profiles, vehicle assignments, license tracking, compliance status.
- **Vehicle Tracking**: Asset profiles, registration monitoring, maintenance schedules, fleet readiness.
- **Route & Dispatch Control**: Trip scheduling, route planning, automated dispatch workflows.
- **Secure Audit Logging**: Multi-tenant RBAC, complete operational action history, security audit trail.
- **Performance Analytics**: Trip completion rates, vehicle utilization, financial margins.
- **Shipper/Customer Portal**: Direct booking requests, automated invoicing, transparent billing, real-time tracking, historical reporting.

---

## 🚛 Layer 1: Foundational Operations – Driver-Centric & India Compliance

Critical operational and regulatory features required for real-world Indian interstate and intrastate transportation.

### 1. Driver Mobile Application (Native PWA / Mobile)
The essential bridge between dispatchers and road execution.
- **Trip Acceptance Workflow**: Drivers view detailed trip specs (cargo, schedule, consignee contact) and accept/reject loads.
- **Proof of Delivery (e-POD)**: Digital signature capture, delivered cargo photo uploads, optional Aadhaar masking for consignee verification, short/damage/refusal logging.
- **Expense & Advance Management**: On-the-fly logging for fuel receipts, RTO expenses, tyre repairs, tolls, and cash advances with receipt photo attachments.
- **Offline First**: Offline local storage for forms, e-Way bills, and route details in zero-network highway zones with auto-sync on connectivity restoration.
- **Vernacular Language Support**: Multi-lingual interface supporting Hindi, Tamil, Telugu, Kannada, Marathi, and Gujarati.

### 2. India-Specific Compliance Engine
Real-time regulatory compliance integrated into trip dispatch lifecycle.
- **E-Way Bill Integration (GSTN API)**:
  - Auto-generate Part-A and Part-B from dispatch order data.
  - Auto-update vehicle registration changes mid-transit.
  - Expiry monitoring with automated dispatcher notifications and auto-cancellation/extension handlers.
- **FASTag & Toll Reconciliation**:
  - Direct integration with FASTag issuer APIs (NETC / NPCI).
  - Automated toll expense mapping to specific trip IDs.
  - Flag overcharges, auto-calculate driver toll reimbursements, and factor toll costs into net trip profitability.
- **Document Vault & Expiry Blockers**:
  - Digital repository for RC, Fitness Certificates, Insurance, PUC, National/State Permits, and Driving Licenses.
  - **Hard Dispatch Blockers**: System automatically blocks vehicle/driver dispatch if any document expires on or before the trip date.
  - **Parivahan / Sarathi API Verification**: Automated DL and RC validation to catch forged credentials.

### 3. GST-Compliant Invoicing Upgrade
- **E-Invoicing (IRN)**: Automated Invoice Reference Number generation via NIC e-Invoice portal for B2B transactions.
- **Tax Breakdown Engine**: Automated CGST / SGST / IGST calculation based on origin and destination state codes.
- **HSN / SAC Code Mapping**: Prescribed freight service mapping (SAC 9965 / 9967).
- **Cross-Referencing**: Automatic embedding of E-Way Bill numbers on customer invoices.

---

## 💰 Layer 2: Financial Closure & Driver/Carrier Settlement

Closing the financial loop between shipper billing and driver/carrier payouts.

### 4. Driver & Carrier Settlement Engine
- **Flexible Rate Calculators**: Per-km rate, fixed lump sum, percentage of freight, or hybrid models.
- **Deduction Engine**: Automated deduction for cash advances, logged fuel receipts, TDS (Section 194C), damage penalties, and loading/unloading charges.
- **Settlement Statements**: Clear digital debit/credit breakdown per trip or weekly cycle with digital driver confirmation.

### 5. Automated Payout Integration
- **Direct Payouts**: Integration with banking APIs (ICICI, HDFC, RazorpayX) for direct UPI / IMPS / NEFT payouts.
- **Fuel Card Integration**: Direct integration with HPCL / IOCL / BPCL fleet fuel cards for automated advance allocations.

### 6. Detention & Accessorial Automation
- **Geofence Waiting Time**: Automatic entry/exit timestamping at pickup/drop nodes to compute loading/unloading delays.
- **Detention Billing**: Automated calculation of halting charges appended directly to customer invoices.
- **Accessorial Billing**: Automatic line-item additions for helper fees, tolls, and loading labor logged by the driver app.

---

## 🔄 Layer 3: Load Sourcing & Network Efficiency

Maximizing fleet capacity utilization and reducing empty runs.

### 7. Return Load / Reverse Trip Engine
- **Backhaul Matching**: Instant reverse route search upon forward trip creation.
- **Driver Alerts**: Automated notifications to drivers/fleet owners for high-margin return loads along their destination route.
- **Deadhead Reduction**: Minimizing unladen mileage (empty runs).

### 8. Indian Load Board & Aggregator Integration
- **Platform Connectors**: Integration with Indian freight marketplaces (Vahak, TruckSuvidha, BlackBuck).
- **Single-Truck Owner (STO) Portal**: Self-onboarding portal for individual truck owners to upload documents and bid on open loads.

---

## 🧠 Layer 4: Advanced Visibility & Intelligence

Machine-learned analytics, predictive modeling, and ESG metrics.

### 9. Predictive ETAs & Exception Routing
- **AI-Driven ETAs**: Machine learning arrival estimates accounting for historical highway traffic, weather/monsoon disruptions, and known checkpost delays.
- **Real-Time Exception Alerts**: Automated alerts for route deviation, prolonged unscheduled stops, delayed pickup, or delivery risks.

### 10. Driver & Carrier Performance Scorecards
- **Performance Indexing**: Rating based on on-time delivery %, e-POD compliance speed, claim/damage rates, document freshness, and shipper feedback.
- **Automated Incentives**: Preferred load allocation and performance bonus triggers for top-ranked drivers.

### 11. Carbon Footprint (ESG) Dashboard
- **Emissions Calculator**: Per-trip CO₂ estimation based on distance, fuel type, and load factor.
- **Corporate ESG Reporting**: Automated export of sustainability metrics for enterprise shipper compliance.

---

## 🔌 Layer 5: Integration Ecosystem

Seamless data sync with enterprise software stack.

- **Accounting Software**: Two-way sync with Tally Prime, Zoho Books, and QuickBooks India for voucher creation, GST reconciliation, and ledger posts.
- **Telematics & IoT**: Integration with Indian GPS providers (LocoNav, Sensel, Chakra, WheelsEye) for live location, fuel sensors, and reefer temperature monitoring. **Design-complete — implementation-ready tech specs in `docs/tech-specs/`:**
  - [`01-telematics-ingestion.md`](docs/tech-specs/01-telematics-ingestion.md) — own-GPS device registry, MQTT/REST ingest, canonical PositionEvent/AlertEvent/SOSEvent contracts, LocoNav/WheelsEye adapters (migrations 00039–00040)
  - [`02-geofence-engine.md`](docs/tech-specs/02-geofence-engine.md) — zones, 4-state dwell machine, trip auto-transitions, detention invoicing (00041)
  - [`03-fuel-audit-scorecard.md`](docs/tech-specs/03-fuel-audit-scorecard.md) — fuel anomaly engine, kharcha claim audit, driver scorecard + settlement bonus (00042)
  - [`04-live-map-share-maintenance.md`](docs/tech-specs/04-live-map-share-maintenance.md) — FlyFleet map stack (keyless Google tiles + OSM fallback + Nominatim), SSE live map, share links + ETA, preventive maintenance (00043)
  - [`05-alerting-compliance.md`](docs/tech-specs/05-alerting-compliance.md) — alert pipeline, compliance dispatch gates, e-way bill lifecycle, SOS, AIS-140 contract (00044–00048)
- **Enterprise ERP / WMS**: REST APIs for SAP, Oracle SCM, and Unicommerce to auto-ingest orders and push back e-PODs and invoice data.

---
*Documented on: August 12, 2026 | Avandab Engineering & Product Architecture*
