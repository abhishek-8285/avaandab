# Fleet SaaS Feature Page Blueprint

Implementation-ready specification synthesized from Fleetio, Onfleet, Stripe, Linear, and Indian SaaS best practices.

---

## PART A: SECTION-BY-SECTION BLUEPRINT

---

### SECTION 1: HERO

**Section name:** Hero / Hero Banner

**Heading options:**
1. "One Platform to Manage Your Entire Fleet"
2. "Stop Juggling Tools. Start Running Your Fleet."
3. "The Fleet Management System Built for Scale"

**Subhead options:**
1. "Track every vehicle, manage maintenance, and control costs — all from a single dashboard."
2. "From dispatch to delivery, get real-time visibility across your entire operation."
3. "Join 5,000+ fleet managers who replaced spreadsheets with one powerful platform."

**Layout:**
- Max-width: `max-w-7xl` (1280px)
- Padding: `py-20 px-6 sm:px-8 lg:px-12`
- Background: `bg-white` (or `bg-slate-900` for dark hero variant)
- Grid: 2-column on desktop (`grid-cols-1 lg:grid-cols-2`), 1-column on mobile

**Content structure:**
- Left column: H1 heading, subheading (max 2 lines), 2 CTA buttons (primary + secondary), trust badges row (G2, Capterra, 5-star rating)
- Right column: Product screenshot or animated demo (with subtle `shadow-2xl` and `rounded-2xl`)
- Below hero: Logo strip of 5-7 customer logos (`grayscale opacity-50 hover:opacity-100 transition`)

**CTA type and placement:**
- Primary CTA: "Book a Demo" (filled button)
- Secondary CTA: "Start Free Trial" (outline button)
- Position: Below subheading text, inline row

**Mobile adaptation:**
- Stack to single column, text above image
- Reduce heading to `text-3xl` from `text-5xl`
- Hide logo strip on mobile (show only on `lg:`)
- Reduce padding to `py-12 px-4`

---

### SECTION 2: SOCIAL PROOF BAR (Logo Strip)

**Section name:** Trust Bar / Logo Strip

**Heading options:**
1. "Trusted by Leading Fleets Across India"
2. "Chosen by 5,000+ Fleet Managers Worldwide"
3. "Powering Fleets That Deliver Every Day"

**Subhead options:** (none needed — this is visual)

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-12 px-6`
- Background: `bg-slate-50`
- Grid: `grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-8 items-center`

**Content structure:**
- 5-6 grayscale customer logos with hover transition to color
- Optional stat callout: "5,000+ fleets | 50,000+ vehicles | 99.9% uptime"

**CTA:** None

**Mobile adaptation:**
- Show 2 logos per row on mobile
- Hide stat callout on small screens

---

### SECTION 3: KEY CAPABILITIES OVERVIEW

**Section name:** Features Overview / What You Get

**Heading options:**
1. "Everything You Need to Run a Smarter Fleet"
2. "All the Tools. One Dashboard."
3. "Built for Every Aspect of Fleet Operations"

**Subhead options:**
1. "From vehicle tracking to preventive maintenance, manage it all in one place."
2. "Replace 5+ tools with a single platform designed for fleet teams."
3. "Reduce manual work, cut costs, and keep your fleet running at peak performance."

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-white`
- Grid: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6`

**Content structure:**
- 6 capability cards in a 3-column grid
- Each card: icon (48x48, colored bg), H3 title, 1-line description
- Card hover: `hover:shadow-lg hover:-translate-y-1 transition-all duration-200`

**Card content (example fleet capabilities):**
| Icon | Title | Description |
|------|-------|-------------|
| 📍 | GPS Tracking | Real-time vehicle location and route history |
| 🔧 | Maintenance | Preventive scheduling and work order management |
| ⛽ | Fuel Management | Track consumption, spot anomalies, control costs |
| 📋 | Inspections | Digital vehicle inspections with photo capture |
| 📊 | Analytics | Fleet dashboards and custom reports |
| 📱 | Mobile App | Driver assignments and field operations |

**CTA:** None (navigation cards link to detail pages)

**Mobile adaptation:**
- Stack to 1 column on mobile
- Show icon inline with text (horizontal card) on `sm:`

---

### SECTION 4: FEATURE DEEP DIVE (Primary Feature)

**Section name:** Feature Spotlight / Hero Feature

**Heading options:**
1. "Real-Time Fleet Tracking That Actually Works"
2. "Know Where Every Vehicle Is, Every Second"
3. "GPS Tracking Built for Fleet Operations"

**Subhead options:**
1. "Track your entire fleet on a single live map. Get alerts for unauthorized use, idling, and route deviations."
2. "Stop calling drivers for status updates. See real-time location, speed, and engine status for every vehicle."
3. "From dispatch to delivery, maintain complete visibility across your entire operation."

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-24 px-6 sm:px-8`
- Background: `bg-slate-50`
- Grid: 2-column (`grid-cols-1 lg:grid-cols-2 gap-12 items-center`)

**Content structure:**
- Left column: Heading, subheading, 3 bullet points with checkmark icons, CTA button
- Right column: Large product screenshot (`rounded-2xl shadow-xl`)
- Below: 3 stat cards in a row ("40% reduction in fuel costs", "3x faster reporting", "99.9% uptime")

**Bullet points:**
- ✅ Live map with vehicle status, speed, and direction
- ✅ Geofencing alerts for unauthorized movement
- ✅ Route history and playback for compliance

**CTA:** "See It in Action →" (primary button)

**Mobile adaptation:**
- Stack to single column
- Reduce screenshot size
- Stats become a horizontal scroll on mobile

---

### SECTION 5: FEATURE GRID (3-4 Features)

**Section name:** Features Showcase / Capability Grid

**Heading options:**
1. "Built for Every Fleet Challenge"
2. "Tools That Work as Hard as You Do"
3. "Features Designed by Fleet Managers, for Fleet Managers"

**Subhead options:**
1. "Each feature solves a real operational pain point — no bloat, no filler."
2. "Stop switching between tools. Everything you need, integrated."
3. "From preventive maintenance to fuel analytics — one platform handles it all."

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-white`
- Grid: `grid grid-cols-1 md:grid-cols-2 gap-8`

**Content structure:**
- 4 feature blocks in a 2x2 grid
- Each block: screenshot (top), H3 title, 2-line description, "Learn more →" link
- Alternate layout: text-left/image-right, then flip for next row

**Feature block content:**
1. **Preventive Maintenance** — Auto-schedule service based on mileage, hours, or calendar intervals
2. **Fuel Management** — Track every litre, detect fraud, and optimize vehicle allocation
3. **Work Orders** — Plan, assign, and track service tasks with full cost visibility
4. **Driver Management** — Assign schedules, track performance, and manage compliance

**CTA:** None (each block links to its own feature page)

**Mobile adaptation:**
- Stack to single column
- Each block becomes: image on top, text below (vertical layout)

---

### SECTION 6: HOW IT WORKS (3-Step Process)

**Section name:** How It Works / Getting Started

**Heading options:**
1. "Up and Running in 3 Simple Steps"
2. "Get Started in Minutes, Not Months"
3. "From Signup to Full Fleet Visibility — Fast"

**Subhead options:**
1. "No complex onboarding. No lengthy implementation. Just results."
2. "Our guided setup gets your fleet online in under a week."
3. "Connect your vehicles, invite your team, and start tracking — today."

**Layout:**
- Max-width: `max-w-5xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-slate-50`
- Grid: `grid grid-cols-1 md:grid-cols-3 gap-8`

**Content structure:**
- 3 step cards in a horizontal row (connected by a subtle line/arrow on desktop)
- Each card: step number (large, colored), H3 title, 2-line description
- Below: single CTA button centered

**Steps:**
1. **Connect Your Vehicles** — Add vehicles via VIN decode or manual entry. Install GPS devices in minutes.
2. **Invite Your Team** — Add drivers, mechanics, and admins with role-based access controls.
3. **Start Saving** — Get real-time insights, automated alerts, and actionable reports from day one.

**CTA:** "Start Your Free Trial" (primary, centered)

**Mobile adaptation:**
- Stack to single column
- Remove connecting line
- Step numbers become inline with heading

---

### SECTION 7: STATS / METRICS BAR

**Section name:** Impact Metrics / Results Bar

**Heading options:**
1. "The Numbers Speak for Themselves"
2. "Proven Results Across Thousands of Fleets"
3. "Measurable Impact from Day One"

**Subhead options:**
1. "Real outcomes from real fleet operations — not marketing promises."
2. "See why fleet managers choose us for measurable cost savings."
3. "Average results reported by our customers within the first 6 months."

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-16 px-6 sm:px-8`
- Background: `bg-slate-900` (dark section for contrast)
- Grid: `grid grid-cols-2 md:grid-cols-4 gap-8`

**Content structure:**
- 4 stat cards in a row
- Each card: large number (animated counter), label text, optional description
- White text on dark background

**Stats:**
| Stat | Label |
|------|-------|
| 40% | Reduction in fuel costs |
| 3x | Faster maintenance reporting |
| 25% | Fewer vehicle breakdowns |
| 99.9% | Platform uptime guaranteed |

**CTA:** None

**Mobile adaptation:**
- 2x2 grid on mobile
- Numbers reduce to `text-3xl` from `text-5xl`

---

### SECTION 8: TESTIMONIAL / CASE STUDY

**Section name:** Customer Story / Testimonial

**Heading options:**
1. "Don't Take Our Word for It"
2. "What Fleet Managers Are Saying"
3. "Results That Speak Louder Than Features"

**Subhead options:**
1. "Hear from fleet managers who transformed their operations with our platform."
2. "Real stories from teams who stopped guessing and started optimizing."
3. "From spreadsheet chaos to fleet clarity — their journey."

**Layout:**
- Max-width: `max-w-5xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-white`
- Grid: Single column centered

**Content structure:**
- Large quote block: opening quote icon, testimonial text (2-3 sentences, `text-xl italic`), customer name, title, company, company logo
- Below: 3 mini-testimonial cards in a row (name + company + 1-line quote)
- Optional: CTA to "Read the full case study →"

**Testimonial format:**
```
❝ [Quote text here — 2-3 sentences about a specific result]

— [Name], [Title], [Company]
   [Company Logo]
```

**CTA:** "Read Case Studies →" (text link)

**Mobile adaptation:**
- Full-width single column
- Mini testimonials stack vertically

---

### SECTION 9: INTEGRATIONS

**Section name:** Integrations / Works With Your Stack

**Heading options:**
1. "Integrates with the Tools You Already Use"
2. "Plug Into Your Existing Tech Stack"
3. "Connect Everything. Break Nothing."

**Subhead options:**
1. "From GPS providers to accounting software, we connect to 100+ tools out of the box."
2. "No rip-and-replace. We integrate with your current workflow."
3. "Open API, webhooks, and pre-built connectors for seamless data flow."

**Layout:**
- Max-width: `max-w-7xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-slate-50`
- Grid: Logo grid `grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 gap-6`

**Content structure:**
- Heading + subheading centered
- Logo grid of integration partners (GPS, fuel cards, ERP, accounting)
- Below: "View all integrations →" link
- Optional: code snippet showing API simplicity

**CTA:** "View Integration Directory →" (text link, centered)

**Mobile adaptation:**
- 3-column grid on mobile
- Reduce logo sizes

---

### SECTION 10: PRICING TEASER

**Section name:** Pricing Preview / Simple Pricing

**Heading options:**
1. "Pricing That Scales With Your Fleet"
2. "Transparent Pricing. No Hidden Fees."
3. "Start Free. Upgrade When You're Ready."

**Subhead options:**
1. "Per-vehicle pricing that grows with your operation. No long-term contracts."
2. "Try free for 14 days. Then pay only for what you use."
3. "Enterprise pricing available for fleets of 100+ vehicles."

**Layout:**
- Max-width: `max-w-5xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-white`
- Grid: `grid grid-cols-1 md:grid-cols-3 gap-8`

**Content structure:**
- 3 pricing cards: Free Trial, Pro, Enterprise
- Each card: plan name, price, feature list (6-8 items with checkmarks), CTA button
- Middle card highlighted (border-2, primary color)

**Pricing cards:**
| | Free Trial | Pro | Enterprise |
|---|---|---|---|
| Price | ₹0 for 14 days | ₹499/vehicle/mo | Custom |
| Vehicles | Up to 10 | Unlimited | Unlimited |
| Users | 3 | Unlimited | Unlimited |
| Support | Email | Priority | Dedicated |
| API | ❌ | ✅ | ✅ |

**CTA:** "Start Free Trial" (primary) / "Contact Sales" (secondary)

**Mobile adaptation:**
- Stack to single column
- Free Trial card first (mobile-first purchase intent)

---

### SECTION 11: FAQ

**Section name:** Frequently Asked Questions

**Heading options:**
1. "Got Questions? We've Got Answers."
2. "Frequently Asked Questions"
3. "Everything You Need to Know Before You Start"

**Subhead options:** (none needed)

**Layout:**
- Max-width: `max-w-3xl`
- Padding: `py-20 px-6 sm:px-8`
- Background: `bg-slate-50`
- Grid: Single column, accordion items

**Content structure:**
- 8-10 accordion FAQ items
- Each: question as clickable header, answer as expandable body
- Default: first item expanded

**FAQ questions:**
1. How long does it take to get set up?
2. Do you support all vehicle types?
3. Can I import data from my existing system?
4. Is there a mobile app for drivers?
5. What integrations do you offer?
6. How do you handle data security?
7. Can I try before I buy?
8. What kind of support do you provide?

**CTA:** None

**Mobile adaptation:**
- Same accordion pattern works on mobile
- Reduce padding

---

### SECTION 12: FINAL CTA

**Section name:** Bottom CTA / Closing Banner

**Heading options:**
1. "Ready to Transform Your Fleet Operations?"
2. "Start Managing Your Fleet Smarter — Today"
3. "Join 5,000+ Fleet Managers Who Made the Switch"

**Subhead options:**
1. "Start your free trial. No credit card required. Set up in minutes."
2. "Get a personalized demo and see how we fit your fleet."
3. "Free 14-day trial. Full access. No commitment."

**Layout:**
- Max-width: `max-w-5xl`
- Padding: `py-24 px-6 sm:px-8`
- Background: `bg-slate-900` (dark) or gradient
- Grid: Single column centered

**Content structure:**
- Centered heading, subheading, 2 CTA buttons
- Below: trust badges (security certifications, uptime guarantee)
- Optional: "Or call us at +91-XXXX-XXXXXX"

**CTA:**
- Primary: "Start Free Trial"
- Secondary: "Book a Demo"

**Mobile adaptation:**
- Stack buttons vertically
- Reduce heading size

---

## PART B: TAILWIND CSS DESIGN SYSTEM

---

### COLOR PALETTE

```javascript
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6',  // Main primary
          600: '#2563eb',  // Primary hover
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a',
        },
        secondary: {
          50: '#f0fdf4',
          100: '#dcfce7',
          200: '#bbf7d0',
          300: '#86efac',
          400: '#4ade80',
          500: '#22c55e',  // Success/growth green
          600: '#16a34a',
          700: '#15803d',
          800: '#166534',
          900: '#14532d',
        },
        accent: {
          50: '#fff7ed',
          100: '#ffedd5',
          200: '#fed7aa',
          300: '#fdba74',
          400: '#fb923c',
          500: '#f97316',  // Action/warning orange
          600: '#ea580c',
          700: '#c2410c',
          800: '#9a3412',
          900: '#7c2d12',
        },
        neutral: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617',
        },
      },
    },
  },
}
```

### COLOR USAGE RULES

| Token | Hex | Usage |
|-------|-----|-------|
| `primary-500` | `#3b82f6` | CTAs, links, active states, focus rings |
| `primary-600` | `#2563eb` | CTA hover, primary button hover |
| `secondary-500` | `#22c55e` | Success states, positive metrics, checkmarks |
| `accent-500` | `#f97316` | Warnings, badges, highlights, alerts |
| `neutral-50` | `#f8fafc` | Page backgrounds, section alternation |
| `neutral-100` | `#f1f5f9` | Card backgrounds, subtle fills |
| `neutral-200` | `#e2e8f0` | Borders, dividers |
| `neutral-500` | `#64748b` | Secondary text, captions |
| `neutral-800` | `#1e293b` | Body text |
| `neutral-900` | `#0f172a` | Headings, dark section backgrounds |
| `white` | `#ffffff` | Card backgrounds, hero backgrounds |

### TYPOGRAPHY SCALE

```css
/* Exact typography values */
.text-hero    { font-size: 3rem;    line-height: 1.1;  font-weight: 700; letter-spacing: -0.02em; } /* 48px */
.text-h1      { font-size: 2.25rem; line-height: 1.2;  font-weight: 700; letter-spacing: -0.02em; } /* 36px */
.text-h2      { font-size: 1.875rem;line-height: 1.25; font-weight: 700; letter-spacing: -0.015em;}// 30px
.text-h3      { font-size: 1.25rem; line-height: 1.4;  font-weight: 600; letter-spacing: -0.01em;  }// 20px
.text-body    { font-size: 1rem;    line-height: 1.6;  font-weight: 400; }                          // 16px
.text-small   { font-size: 0.875rem;line-height: 1.5;  font-weight: 400; }                          // 14px
.text-caption { font-size: 0.75rem; line-height: 1.5;  font-weight: 500; letter-spacing: 0.05em; text-transform: uppercase; } // 12px
```

**Tailwind classes:**
```
Heading H1:   text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight text-neutral-900
Heading H2:   text-3xl sm:text-4xl font-bold tracking-tight text-neutral-900
Heading H3:   text-xl sm:text-2xl font-semibold text-neutral-900
Body:         text-base sm:text-lg text-neutral-600 leading-relaxed
Small:        text-sm text-neutral-500
Caption:      text-xs font-semibold uppercase tracking-widest text-primary-500
```

### SPACING SCALE

```css
/* Section spacing */
.section-padding     { padding-top: 5rem; padding-bottom: 5rem; }    /* 80px = py-20 */
.section-padding-sm  { padding-top: 3rem; padding-bottom: 3rem; }    /* 48px = py-12 */
.section-padding-lg  { padding-top: 6rem; padding-bottom: 6rem; }    /* 96px = py-24 */

/* Container */
.container-max       { max-width: 1280px; margin-left: auto; margin-right: auto; }  /* max-w-7xl */
.container-narrow    { max-width: 768px; margin-left: auto; margin-right: auto; }   /* max-w-3xl */
.container-medium    { max-width: 1024px; margin-left: auto; margin-right: auto; }  /* max-w-5xl */

/* Card spacing */
.card-padding        { padding: 1.5rem; }       /* 24px = p-6 */
.card-padding-lg     { padding: 2rem; }         /* 32px = p-8 */
.card-gap            { gap: 1.5rem; }           /* 24px = gap-6 */
.card-gap-lg         { gap: 2rem; }             /* 32px = gap-8 */

/* Element spacing */
.element-gap-sm      { gap: 0.5rem; }           /* 8px = gap-2 */
.element-gap         { gap: 1rem; }             /* 16px = gap-4 */
.element-gap-lg      { gap: 1.5rem; }           /* 24px = gap-6 */
```

**Tailwind classes:**
```
Section:      py-20 px-6 sm:px-8 lg:px-12
Card:         p-6 sm:p-8
Grid gap:     gap-6 sm:gap-8
Element gap:  space-y-4 or gap-4
Container:    max-w-7xl mx-auto
```

### CARD STYLES

```css
/* Default card */
.card {
  background-color: white;
  border-radius: 0.75rem;           /* rounded-xl = 12px */
  border: 1px solid #e2e8f0;        /* border border-neutral-200 */
  padding: 1.5rem;                   /* p-6 */
  transition: all 0.2s ease;
}
.card:hover {
  box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -4px rgba(0,0,0,0.1);  /* shadow-lg */
  transform: translateY(-2px);       /* -translate-y-0.5 */
}

/* Feature card (with screenshot) */
.feature-card {
  background-color: white;
  border-radius: 1rem;               /* rounded-2xl = 16px */
  border: 1px solid #e2e8f0;
  overflow: hidden;
  transition: all 0.3s ease;
}
.feature-card:hover {
  box-shadow: 0 25px 50px -12px rgba(0,0,0,0.15);  /* shadow-2xl */
}

/* Pricing card (highlighted) */
.pricing-card-featured {
  background-color: white;
  border-radius: 1rem;
  border: 2px solid #3b82f6;         /* border-2 border-primary-500 */
  padding: 2rem;
  position: relative;
  transform: scale(1.05);            /* scale-[1.05] */
}
```

**Tailwind classes:**
```
Default:      bg-white rounded-xl border border-neutral-200 p-6 hover:shadow-lg hover:-translate-y-0.5 transition-all duration-200
Feature:      bg-white rounded-2xl border border-neutral-200 overflow-hidden hover:shadow-2xl transition-all duration-300
Pricing:      bg-white rounded-2xl border-2 border-primary-500 p-8 relative scale-105
Dark:         bg-neutral-900 rounded-2xl p-8 text-white
```

### BUTTON STYLES

```css
/* Primary button */
.btn-primary {
  display: inline-flex;
  align-items: center;
  padding: 0.75rem 1.5rem;           /* py-3 px-6 */
  font-size: 1rem;                   /* text-base */
  font-weight: 600;                  /* font-semibold */
  color: white;
  background-color: #3b82f6;         /* bg-primary-500 */
  border-radius: 0.5rem;             /* rounded-lg */
  transition: all 0.2s ease;
  cursor: pointer;
}
.btn-primary:hover {
  background-color: #2563eb;         /* bg-primary-600 */
  box-shadow: 0 4px 12px rgba(59,130,246,0.4);  /* shadow with primary */
}

/* Secondary button (outline) */
.btn-secondary {
  display: inline-flex;
  align-items: center;
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
  font-weight: 600;
  color: #3b82f6;                    /* text-primary-500 */
  background-color: transparent;
  border: 1.5px solid #3b82f6;       /* border border-primary-500 */
  border-radius: 0.5rem;
  transition: all 0.2s ease;
}
.btn-secondary:hover {
  background-color: #eff6ff;         /* bg-primary-50 */
}

/* Text link / ghost button */
.btn-text {
  display: inline-flex;
  align-items: center;
  padding: 0;
  font-size: 1rem;
  font-weight: 600;
  color: #3b82f6;
  background: none;
  border: none;
  cursor: pointer;
  transition: color 0.2s ease;
}
.btn-text:hover {
  color: #2563eb;
  text-decoration: underline;
}
```

**Tailwind classes:**
```
Primary:   inline-flex items-center px-6 py-3 text-base font-semibold text-white bg-primary-500 rounded-lg hover:bg-primary-600 hover:shadow-lg hover:shadow-primary-500/25 transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2

Secondary: inline-flex items-center px-6 py-3 text-base font-semibold text-primary-500 bg-transparent border-1.5 border-primary-500 rounded-lg hover:bg-primary-50 transition-all duration-200

Text:      inline-flex items-center text-base font-semibold text-primary-500 hover:text-primary-600 hover:underline transition-colors duration-200

Small:     inline-flex items-center px-4 py-2 text-sm font-semibold text-white bg-primary-500 rounded-lg hover:bg-primary-600 transition-all duration-200
```

### GRID SYSTEM

```css
/* Page grid */
.page-grid {
  display: grid;
  gap: 1.5rem;
}

/* 2-column feature layout */
.feature-split {
  display: grid;
  grid-template-columns: 1fr;
  gap: 3rem;
  align-items: center;
}
@media (min-width: 1024px) {
  .feature-split {
    grid-template-columns: 1fr 1fr;
  }
}

/* 3-column capability grid */
.capability-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.5rem;
}
@media (min-width: 640px) {
  .capability-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
@media (min-width: 1024px) {
  .capability-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* 4-column stats grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 2rem;
}
@media (min-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

/* 3-column pricing grid */
.pricing-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 2rem;
  align-items: start;
}
@media (min-width: 768px) {
  .pricing-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
```

**Tailwind classes:**
```
2-col split:     grid grid-cols-1 lg:grid-cols-2 gap-12 items-center
3-col cards:     grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6
4-col stats:     grid grid-cols-2 md:grid-cols-4 gap-8
Logo strip:      grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-8 items-center
Pricing:         grid grid-cols-1 md:grid-cols-3 gap-8 items-start
FAQ:             max-w-3xl mx-auto space-y-4
```

### RESPONSIVE BREAKPOINTS

```
sm:  640px   → 2-col grids, text scaling
md:  768px   → 3-4 col grids, stats row
lg:  1024px  → 2-col feature splits, full layout
xl:  1280px  → max-width containers
```

### FOCUS & ACCESSIBILITY

```css
/* Focus ring for all interactive elements */
*:focus-visible {
  outline: 2px solid #3b82f6;
  outline-offset: 2px;
  border-radius: 4px;
}

/* Skip to content link */
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  background: #3b82f6;
  color: white;
  padding: 8px 16px;
  z-index: 100;
  transition: top 0.3s;
}
.skip-link:focus {
  top: 0;
}
```

---

## PART C: COPY FRAMEWORK

---

### HEADLINE FORMULA

**Pattern:** `[Action verb] + [Core benefit] + [Scope/context]`

**Formula variants:**
1. **Verb-First:** "Track Every Vehicle in Real Time"
2. **Problem-First:** "Stop Losing Money on Fleet Inefficiency"
3. **Outcome-First:** "Reduce Fuel Costs by 40%"
4. **Question-First:** "What If You Knew Where Every Vehicle Was?"
5. **Social-Proof-First:** "Join 5,000+ Fleets Saving 40% on Fuel"

**Examples:**
- "Manage Your Entire Fleet from One Dashboard"
- "Cut Fuel Costs Without Cutting Corners"
- "Know Before It Breaks: Predictive Fleet Maintenance"

### SUBHEAD FORMULA

**Pattern:** `[Specific benefit] + [How/what] + [Proof/scope]`

**Formula variants:**
1. **Benefit + Mechanism:** "Track every vehicle with real-time GPS and get alerts for unauthorized use."
2. **Problem + Solution:** "Stop calling drivers for updates. See location, speed, and status live."
3. **Social + Scale:** "Join 5,000+ fleet managers who replaced spreadsheets with one platform."
4. **Action + Outcome:** "Connect your vehicles today and start saving on fuel tomorrow."
5. **Quantified:** "Save 40% on fuel costs with AI-powered route optimization."

**Character limits:**
- Hero subhead: 80-120 characters
- Section subhead: 60-100 characters
- Card description: 40-60 characters

### CAPABILITY DESCRIPTION FORMULA

**Pattern:** `[What it does] + [Why it matters] + [Outcome]`

**Formula:**
1. Feature: "Preventive Maintenance"
2. What: "Auto-schedule service based on mileage, hours, or calendar intervals."
3. Why: "Never miss a service deadline again."
4. Outcome: "Reduce breakdowns by 25%."

**Examples:**
- "GPS Tracking — Know where every vehicle is, every second. Get alerts for route deviations and idling."
- "Fuel Management — Track consumption per vehicle, detect anomalies, and optimize allocation."
- "Digital Inspections — Complete vehicle inspections in 5 minutes with photo capture and auto-logging."

### BENEFIT STATEMENT FORMULA

**Pattern:** `[Quantified result] + [in what timeframe] + [for whom]`

**Formula:**
1. Quantified: "40%"
2. Result: "reduction in fuel costs"
3. Timeframe: "within 6 months"
4. For whom: "for mid-size fleet operations"

**Examples:**
- "40% reduction in fuel costs within 6 months for mid-size fleets"
- "3x faster maintenance reporting across all vehicle types"
- "25% fewer vehicle breakdowns with predictive maintenance alerts"
- "Save 10 hours per week on manual fleet admin tasks"

### FAQ QUESTION FORMULA

**Pattern:** `[Topic] + [Specific concern] + [Implied ease]**

**Formula:**
1. Setup: "How long does it take to get set up?" → Implies: it's fast
2. Compatibility: "Do you support all vehicle types?" → Implies: yes
3. Migration: "Can I import data from my existing system?" → Implies: yes
4. Support: "What kind of support do you provide?" → Implies: comprehensive
5. Pricing: "Are there any hidden fees?" → Implies: no
6. Security: "How do you protect my data?" → Implies: thoroughly
7. Trial: "Can I try before I buy?" → Implies: yes, free
8. Mobile: "Is there a mobile app for drivers?" → Implies: yes

### CTA FORMULA

**Pattern:** `[Action verb] + [What they get] + [Risk reversal]`

**Primary CTAs:**
| Context | CTA Text |
|---------|----------|
| Free trial | "Start Free Trial" |
| Demo | "Book a Demo" |
| Pricing | "View Pricing" |
| Case study | "Read Case Study" |
| Feature | "See It in Action" |
| Contact | "Talk to Sales" |

**Secondary CTAs:**
| Context | CTA Text |
|---------|----------|
| Learn more | "Learn More →" |
| Docs | "Read the Docs" |
| Compare | "Compare Plans" |
| Features | "Explore Features" |

**Risk reversal phrases (for subhead below CTA):**
- "No credit card required"
- "Free 14-day trial"
- "Set up in under 10 minutes"
- "Cancel anytime"

### SOCIAL PROOF FORMULA

**Quote pattern:**
```
"[Specific result with number] — [What changed]."

— [Name], [Title], [Company]
```

**Example:**
"It would take me 8-10 hours to route 500 orders. In [Product], I was able to do it in less than an hour."

— Robin, Delivery Manager, Inspired Go

### SECTION HEADING PATTERN

Every section follows:
```
[Caption/eyebrow text] → text-xs font-semibold uppercase tracking-widest text-primary-500
[Main heading] → text-3xl sm:text-4xl font-bold tracking-tight text-neutral-900
[Subheading] → text-lg text-neutral-600 max-w-2xl mx-auto
```

---

## PART D: IMPLEMENTATION CHECKLIST

---

### Pre-Build

- [ ] Set up Tailwind config with color palette from Part B
- [ ] Create component files: `Hero.tsx`, `FeatureCard.tsx`, `StatCard.tsx`, `PricingCard.tsx`, `TestimonialCard.tsx`, `FAQAccordion.tsx`
- [ ] Add typography classes to global CSS
- [ ] Set up responsive container utility

### Build Order

1. Hero section (with trust badges)
2. Logo strip / social proof bar
3. Feature grid (3-column capability cards)
4. Feature deep dive (2-column split)
5. Stats bar (dark section)
6. Testimonial section
7. How it works (3-step)
8. Integrations logo grid
9. Pricing teaser
10. FAQ accordion
11. Final CTA banner
12. Footer

### QA Checklist

- [ ] All CTAs have `focus-visible` ring
- [ ] Mobile responsive at 375px, 768px, 1024px, 1280px
- [ ] Images have `alt` text
- [ ] Heading hierarchy is sequential (h1 → h2 → h3)
- [ ] Color contrast meets WCAG AA (4.5:1 for text)
- [ ] Page loads in < 3 seconds
- [ ] No layout shift on image load (set width/height)
- [ ] FAQ accordion works with keyboard navigation
- [ ] All links have `hover` and `focus` states

---

*Blueprint v1.0 — Synthesized from Fleetio, Onfleet, Stripe, Linear, Motive, Routific, and Indian SaaS patterns.*
