# Samsara GPS Fleet Tracking — Complete Wireframe Document

**Source:** https://www.samsara.com/products/telematics/gps-fleet-tracking

---

## 1. Design System Tokens

### Fonts
| Token | Family | Weight | Used For |
|-------|--------|--------|----------|
| `Samsara Sans` | `Samsara Sans, FK Grotesk, system-ui, sans-serif` | 700 (Bold) | H1, H2, h2 |
| `Samsara Sans` | Same | 500 (Medium) | Body paragraphs, nav links |
| `FK Grotesk` | `FK Grotesk` | 500 (Medium) | Subheadings, form labels |
| `FK Grotesk` | Same | 700 (Bold) | Metric callouts, stat numbers |
| Font preload: `SamsaraSans-Regular.woff2`, `FKGrotesk-Medium.woff2`, `FKGrotesk-Bold.woff2` | | | |

### Color Palette (Extracted from CSS)
| Color Name | CSS Value | Hex | Usage |
|------------|-----------|-----|-------|
| `charcoal` | `rgb(28, 25, 23)` | `#1c1917` | Primary text, headings |
| `ash` | `rgb(64, 61, 55)` | `#403d37` | Secondary text, card hover bg |
| `stone-400` | `rgb(156, 152, 144)` | `#9c9890` | Placeholder text, muted text |
| `stone-300` | `rgb(204, 201, 193)` | `#ccc9c1` | Border color, form border |
| `white` | `rgb(255, 255, 255)` | `#ffffff` | White text on dark bg, form bg |
| `snow` | — | — | Light background sections |
| `gray-50` | Tailwind `bg-gray-50` | — | Nav bar background |
| `gray-700` | Tailwind `bg-gray-700` | — | Resource cards, dark surfaces |
| `gray-800` | Tailwind `bg-gray-800` | — | CTA/FOOTER dark sections |
| `blue` | Tailwind `bg-blue` | `#007bff` | Primary accent, CTAs, slider indicators |
| `blue-dark` | Tailwind `bg-blue-dark` | — | Slider bar bg |
| `blue-dark` | Tailwind `focus-visible:outline-blue-dark` | — | Focus ring |
| `yellow` | — | `#feae0f` | Trust badge accent, focus ring |
| `red` | `#df2036` | `#df2036` | Error states, required field asterisk |
| `green` | `#0dab41` | `#0dab41` | Form valid outline |

### Spacing Scale (Custom tokens from CSS)
| Token | Value | Usage |
|-------|-------|-------|
| `s20` | 20px | Nav padding, gap |
| `s25` | 25px | Column padding |
| `s30` | 30px | Nav px padding (lg+) |
| `s40` | 40px | Nav gap (lg+) |
| `gap2` | ~24px | Row container padding-x |
| `space1` | ~16px | Card bottom padding |
| `container` | `max-width: 1280px` (lg), `1440px` (xl) | Row container |

### Border Radius
| Element | Value |
|---------|-------|
| Nav bar | `rounded` (4px default) |
| Cards (collection/resource) | `rounded-[4px]` = 4px |
| Image wrappers | `rounded-[4px]` |
| Resource card images | `rounded-[5px]` = 5px |
| CTA buttons (pill) | `rounded-full` |
| Accordion border-bottom | `0.5px` |
| Modal | `border-radius: 5px` |
| Card overview link icon | `rx="14.7494"` (circle, ~30x30) |

### Shadows
- No box-shadow values found in CSS — page relies on color contrast, not shadows
- Cards use solid bg + rounded corners only

---

## 2. Global Navigation Bar

```
┌──────────────────────────────────────────────────────────────┐
│ [fixed, top-16px, z-50, left-0, w-full, px-20 lg:px-30]    │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ Nav bar: flex, h-[56px], max-w-[1408px], mx-auto,       │ │
│ │   rounded (4px), bg-gray-50, items-center                │ │
│ │                                                          │ │
│ │ [Logo 198x30px] [Products] [Solutions] [Resources]      │ │
│ │ [About] [Pricing]              [Company] [Cloud] [CTA]  │ │
│ └──────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

**Structure:**
- `position: fixed`, `top: 16px`, `z-index: 50`, `left: 0`, `w-full`
- Inner bar: `max-w-[1408px]`, `mx-auto`, `h-[56px]`, `rounded`, `bg-gray-50`
- Logo: `block h-[30px] w-[198px]` SVG (Samsara wordmark + icon)
- Nav items: `gap-s20 lg:gap-s40`, `h-full`, `min-w-0`, `items-center`
- Right side: `gap-s20`, cloud link, "Check our prices" CTA button

**Trust badges in nav (below bar):**
- G2 4.5 stars, 4,800+ reviews
- #1 rated driver app
- #1 for fleet management

---

## 3. Page Sections (Top → Bottom)

### SECTION 1: Hero (Video Background)
**Layout:** Full-width video background with centered text overlay

```
┌─────────────────────────────────────────────────────────────┐
│  VIDEO BACKGROUND (Hero.mp4)                                │
│  min-height: min(var(--hero-min-height-desktop), 640px)     │
│  object-position: var(--hero-object-position-desktop)        │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  BREADCRUMBS (above hero content):                     │ │
│  │  1. Products > Fleet Telematics > GPS Fleet Tracking   │ │
│  │                                                        │ │
│  │  G2 Badge: [4.5 stars img] G2 4.5 stars               │ │
│  │  4,800+ reviews                                        │ │
│  │                                                        │ │
│  │  H1: "GPS Fleet Tracking"                              │ │
│  │  max-w: 450px (mobile), 1126px (desktop)               │ │
│  │  font: Samsara Sans, 700, uppercase                    │ │
│  │  font-size: calc(48px * scale) → calc(72px * scale)    │ │
│  │  color: white, text-align: center                      │ │
│  │  margin-bottom: 24px                                   │ │
│  │                                                        │ │
│  │  P: "Monitor your fleet, adapt to changing conditions, │ │
│  │     and keep drivers on schedule with full visibility   │ │
│  │     and extended tracking."                             │ │
│  │  font-size: 20px → 24px, weight: 500, line-height: 120%│ │
│  │  color: white, max-w: 450px → 894px                    │ │
│  │  margin-bottom: 24px                                   │ │
│  │                                                        │ │
│  │  CTA: [Check our prices]  ← btn style                  │ │
│  │  margin: 12px each side, margin-bottom: -12px          │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**CSS specifics:**
- `.hero--main`: `min-height: min(var(--hero-min-height-desktop), 640px)`
- `.hero__rows h1`: `font-weight: 700`, `text-transform: uppercase`, `color: white`
- `.hero__rows p:not(:has(.btn))`: `font-size: 20px`, `font-weight: 500`, `line-height: 120%`
- Container inside hero: `max-width: 1126px`, `padding: 0`
- Rows have `justify-content: center`

---

### SECTION 2: "Complete Visibility" — Tabbed Accordion with Video
**Eyebrow:** "Complete Visibility" (pill badge)
**H2:** "Track your fleet with real-time GPS data"
**Layout:** Full-width row with accordion blade pattern (desktop: side-by-side, mobile: stacked tabs)

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px, mx-auto                     │
│                                                              │
│  ┌─ Eyebrow pill ──────────────────────────────────────────┐ │
│  │  bg-blue, text-white, rounded-full, px, py, text-sm     │ │
│  │  "Complete Visibility"                                   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  H2: "Track your fleet with real-time GPS data"             │
│  font-size: ~40px (h2 style), weight: 700, color: #1c1917  │
│  margin-bottom: 16px                                        │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  ACCORDION BLADES (4 tabs, horizontal scroll mobile)   │  │
│  │                                                        │  │
│  │  Desktop (≥1280px): CSS Grid with                     │  │
│  │    `grid-template-columns: var(--accordion-blades-grid-template)`│  │
│  │    Active blade wider, others collapsed                │  │
│  │                                                        │  │
│  │  Tabs:                                                │  │
│  │  1. "Helicopter view"                                  │  │
│  │  2. "Smart map overlays"                               │  │
│  │  3. "Live location sharing"                            │  │
│  │  4. "Geofencing"                                       │  │
│  │                                                        │  │
│  │  Tab nav: inline-flex, h-full, items-center,           │  │
│  │    justify-center, gap-x-[16px]                       │  │
│  │  Tab item: w-[190px], grow, font-semibold             │  │
│  │  Active indicator: h-[5px], w-full, bg-blue           │  │
│  │  Inactive bar: h-[5px], bg-blue-dark, opacity-[0.06]  │  │
│  │                                                        │  │
│  │  Image frame: border-radius: 4px, height: 88px        │  │
│  │    → active: height: calc(56.25vw - 27px) on mobile   │  │
│  │    → active: height: 100% on desktop                   │  │
│  │                                                        │  │
│  │  Active content panel: grid-template-rows: 0fr → 1fr   │  │
│  │  transition: grid-template-rows .3s ease-in-out        │  │
│  │                                                        │  │
│  │  Example tab content:                                  │  │
│  │  H4: "Helicopter view"                                 │  │
│  │  P: "Eliminate guesswork with an instant aerial view   │  │
│  │     of your assets. Easily aid route navigation and    │  │
│  │     help authorities track stolen property."            │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  VIDEO: Helicopter_View.mp4 (preloaded)                      │
│  Object-fit: cover, Object-position: 50% 50%                │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 3: "Track Everything" — Card Carousel
**Eyebrow:** "Track Everything" (pill badge)
**H2:** "Extended tracking of your most mobile assets"
**Layout:** Horizontal scrollable card track (3 cards)

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│                                                              │
│  ┌─ Eyebrow pill ──────────────────────────────────────────┐ │
│  │  "Track Everything"                                     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  H2: "Extended tracking of your most mobile assets"         │
│                                                              │
│  CARD TRACK:                                                 │
│  - display: flex, overflow-x: auto, scrollbar-width: none    │
│  - gap: 16px (mobile), 20px (tablet), 24px (desktop)        │
│  - scroll-snap-type: x mandatory                             │
│  - padding: 48px top/bottom, 50px right (mobile)             │
│  - padding-left/right: 11px (desktop)                        │
│                                                              │
│  CARDS (3x):                                                 │
│  Each card:                                                   │
│  - min-h-[250px], flex-col, rounded-[4px]                    │
│  - bg-white (or light), overflow-hidden                       │
│  - Image: w-full, aspect-ratio maintained, rounded-[4px]     │
│  - Content padding: p-1 (mobile) → p-[1.75rem] (desktop)     │
│                                                              │
│  Card 1: "Vehicles"                                          │
│    Image: Vehicles.png (676x962, webp)                       │
│    P: "Optimize your field services with real-time vehicle   │
│       locations, prevent breakdowns by monitoring engine     │
│       diagnostics and fault codes, and cut costs by         │
│       reducing idle time."                                    │
│                                                              │
│  Card 2: "Trailers"                                          │
│    Image: Trailers.png (676x962, webp)                       │
│    P: "Improve trailer utilization, reduce detention time,   │
│       and protect sensitive loads with door, cargo,          │
│       temperature, and humidity sensing..."                  │
│                                                              │
│  Card 3: "Equipment"                                         │
│    Image: Equipment.png (676x962, webp)                      │
│    P: "Get the most out of your equipment. Identify what is  │
│       being underutilized and make data part of your         │
│       procurement planning process."                         │
│                                                              │
│  Top gradient overlay: linear-gradient(180deg, #1f1f1f,      │
│    #3e3e3e00), height: 50px                                  │
│  Bottom gradient: linear-gradient(0deg, #1f1f1f, #3e3e3e00), │
│    height: 240px                                             │
└──────────────────────────────────────────────────────────────┘
```

**Card CSS specifics:**
- `.c--collection-card`: `relative flex h-full flex-col transition-all duration-300 ease-in-out rounded-[4px]`
- Image wrapper: `overflow-hidden rounded-[4px] pb-[12px]`
- Card text: `font-size: 1rem`, `line-height: 1.5rem`, `padding: 0 20px 20px`

---

### SECTION 4: "Routing & Dispatch" — Slider Section
**Eyebrow:** "Routing & Dispatch" (blue pill badge)
**H2:** "Finish routes with fewer miles and vehicles"
**Layout:** Default slider with tabbed navigation

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│                                                              │
│  ┌─ Eyebrow pill (eyebrow-blue) ─────────────────────────┐  │
│  │  "Routing & Dispatch"                                   │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  H2: "Finish routes with fewer miles and vehicles"          │
│                                                              │
│  SLIDER NAV (tabs, horizontal bar indicators):               │
│  - inline-flex, h-full, items-center, justify-center         │
│  - gap-x-[16px], whitespace-nowrap                           │
│  - Tab items: w-[190px], grow, flex-col, self-end            │
│  - Active: pointer-events-none                               │
│  - Indicator: absolute bottom-0, z-10, h-[5px], w-full      │
│  - Animated: h-[5px], w-full, origin-left, scale-0, bg-blue  │
│  - Inactive bar: h-[5px], bg-blue-dark, opacity-[0.06]       │
│                                                              │
│  Tabs:                                                       │
│  1. "Route optimization" (active by default)                 │
│  2. "Dispatch"                                               │
│  3. "Commercial navigation"                                  │
│                                                              │
│  SLIDER CONTENT:                                             │
│  - Full-width image/video: Optimizes-routes.mp4              │
│  - Below image: Title + description text                     │
│  - Description for "Route optimization":                     │
│    "Cut fuel consumption, minimize vehicle wear, maximize    │
│     fleet utilization, and improve worker productivity with  │
│     optimized routes tailored to your organization's         │
│     operational priorities."                                  │
│  - CTA: [Learn more] → /products/telematics/routing          │
│    btn--tertiary style, arrow-right icon                     │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 5: "Custom Reporting" — Content Row with Image
**Eyebrow:** "Custom Reporting" (pill badge)
**H2:** "Analyze performance and uncover cost reduction opportunities"
**Layout:** Two-column (6/12 text + 6/12 image) or image left, text right

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│                                                              │
│  ┌─ Eyebrow pill ──────────────────────────────────────────┐ │
│  │  "Custom Reporting"                                     │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  H2: "Analyze performance and uncover cost reduction         │
│       opportunities"                                         │
│                                                              │
│  IMAGE:                                                      │
│  Custom_Reporting__1_.png (2160x1422, webp)                  │
│  Aerial view of white semi-truck on highway                  │
│  border-radius: 4px (from content module)                    │
│                                                              │
│  SUB-SECTIONS (3 feature items):                             │
│                                                              │
│  1. "Trip history"                                           │
│     P: "Analyze trip data for vehicles in your fleet.        │
│        Identify driver trends to improve upon, and increase  │
│        driver and trip efficiency."                           │
│                                                              │
│  2. "Speeding"                                               │
│     P: "Monitor vehicle speed and harsh driving events.      │
│        Uncover ways to prevent accidents and increase        │
│        driver safety."                                        │
│                                                              │
│  3. "Time on site"                                           │
│     P: "Optimize time spent at each location. With          │
│        visibility into how your drivers are spending their   │
│        days, you can quickly flag unauthorized or            │
│        inefficient activity."                                 │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 6: "Resources" — Card Carousel (2 Report Cards)
**Eyebrow:** "Resources" (pill badge)
**H2:** "Keep up with our latest news & updates"
**Layout:** Horizontal card carousel, 2 cards side by side

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│                                                              │
│  ┌─ Eyebrow pill ──────────────────────────────────────────┐ │
│  │  "Resources"                                            │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                              │
│  H2: "Keep up with our latest news & updates"               │
│                                                              │
│  CARD CAROUSEL (horizontal flex, gap: 16px):                 │
│                                                              │
│  Card 1 (horizontal stack layout):                           │
│  ┌──────────────────────────────────────────────┐           │
│  │  Eyebrow: "Report" (pill badge, small)       │           │
│  │  ┌──────────┐  ┌──────────────────────────┐  │           │
│  │  │  Image   │  │  Content wrapper          │  │           │
│  │  │  (square)│  │  border-left: 1px solid   │  │           │
│  │  │          │  │    hsla(0,0%,100%,.5)     │  │           │
│  │  │  rounded │  │  padding-left: 20px       │  │           │
│  │  │  [5px]   │  │  → 1.75rem (lg)           │  │           │
│  │  │          │  │                           │  │           │
│  │  │          │  │  H3: "Samsara AI helps    │  │           │
│  │  │          │  │  reduce crash rates by    │  │           │
│  │  │          │  │  nearly 75%"              │  │           │
│  │  │          │  │  font-size: 1.25rem       │  │           │
│  │  │          │  │  line-height: 1.75rem     │  │           │
│  │  │          │  │  margin-bottom: 0.75rem   │  │           │
│  │  │          │  │                           │  │           │
│  │  │          │  │  P: description text      │  │           │
│  │  │          │  │  font-size: 0.875rem      │  │           │
│  │  │          │  │  line-height: 1.25rem     │  │           │
│  │  │          │  │                           │  │           │
│  │  │          │  │  CTA: [See the report →]  │  │           │
│  │  │          │  └──────────────────────────┘  │           │
│  │  └──────────┘                                │           │
│  └──────────────────────────────────────────────┘           │
│                                                              │
│  Card 2 (same layout):                                       │
│  Eyebrow: "Report"                                           │
│  Image: fleet-tech-report cover                              │
│  H3: "Samsara rated No. 1: Satisfaction, support & service" │
│  P: "Market research firm Endeavor Business Intelligence     │
│     partnered with FleetOwner and Fleet Maintenance to       │
│     survey more than 500 U.S. fleet professionals;           │
│     Samsara consistently outperformed the competition."      │
│  CTA: [See the report →]                                     │
│                                                              │
│  Card layout CSS:                                            │
│  .card--layout-horizontal-stack:                             │
│    display: flex, flex-direction: column                      │
│  .content--wrapper-outer:                                    │
│    display: flex, flex-direction: row                         │
│  .image--wrapper:                                            │
│    margin-right: 20px → 1.75rem, max-w: 167px (lg)          │
│  .content--wrapper:                                          │
│    border-left: 1px solid hsla(0,0%,100%,.5)                 │
│    padding-left: 20px → 1.75rem                              │
│    flex-grow: 1, justify-content: center                     │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 7: FAQ Accordion
**Eyebrow:** "FAQs" (pill badge, centered)
**H2:** "Learn more about GPS tracking" (centered)
**Layout:** Centered column (8/12 width), accordion items

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│  Column: md:w-8/12, ml-2/12, text-center                     │
│                                                              │
│  ┌─ Eyebrow pill (eyebrow-blue, center-align) ───────────┐  │
│  │  "FAQs"                                                 │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  H2: "Learn more about GPS tracking"                        │
│  text-align: center                                         │
│                                                              │
│  ACCORDION ITEMS (border-bottom: 0.5px solid #ccc9c1):      │
│  First item: border-top also 0.5px                           │
│                                                              │
│  Each accordion:                                             │
│  - Button: flex, w-full, cursor-pointer, items-center        │
│    justify-between, gap-[40px], py-[16px], text-left         │
│  - H4: h5 class, mb-0, w-10/12 md:w-8/12, text-left         │
│    font-size: ~1.25rem, weight: 500                           │
│  - Icon: w-[28px], h-[24px], plus/minus SVG                  │
│    fill: black                                               │
│  - Panel: height: 0 → auto, overflow: hidden                  │
│    transition: height .25s ease-in-out                       │
│  - Panel content: rc__rich-content, width: 100%              │
│    → md: 83.333333%                                          │
│  - Links in content: underline, text-underline-offset: 2px   │
│    color: #1c1917                                            │
│                                                              │
│  FAQ Items:                                                   │
│  1. "What is GPS fleet tracking software and how does it     │
│      work?"                                                  │
│  2. "How is your solution different from other providers?"   │
│  3. "How will my drivers react to being tracked?"            │
│  4. "What are the advantages of a telematics solution for    │
│      fleet managers and drivers?"                            │
│  5. "How much does a telematics solution cost? Will it save  │
│      me money?"                                              │
│  6. "How can we improve driver safety or asset utilization   │
│      with Samsara?"                                          │
│  7. "What is the hardware warranty?"                         │
│  8. "Is it easy to install and set up?"                      │
│  9. "My existing GPS fleet tracking devices are on 3G, what  │
│      are my options?"                                        │
│  10. "Is Samsara ELD compliant?"                             │
│                                                              │
│  FAQ Schema JSON-LD included (FAQPage structured data)        │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 8: "Products That Transform Your Business" — Card Grid
**H2:** "Products that transform your business"
**Subtext:** "Connect your fleet, equipment, sites, and people on the open platform that users love."
**Layout:** Horizontal scrollable card grid (6 cards)

```
┌──────────────────────────────────────────────────────────────┐
│  [Row Container] max-w: 1280px                               │
│                                                              │
│  H2: "Products that transform your business"                │
│  P: "Connect your fleet, equipment, sites, and people on    │
│     the open platform that users love."                      │
│                                                              │
│  CARD GRID (horizontal flex, overflow-x: auto):              │
│  - gap: 16px                                                 │
│  - scroll-snap-type: x mandatory                             │
│                                                              │
│  6 CARDS (c--card-overview):                                 │
│  Each card:                                                  │
│  - min-h-[250px], flex-col, gap-s20, overflow-hidden         │
│  - rounded-[4px], bg-white (or light)                        │
│  - border: 1px solid transparent                              │
│  - transition: all 300ms ease-in-out                         │
│  - Image: w-full, rounded-[4px]                              │
│  - Content: p-1 (mobile), p-[1.75rem] (lg)                   │
│  - H3: font-size: 1.25rem, font-weight: 500, line-height: 1.75rem │
│  - P: font-size: 0.875rem, line-height: 1.25rem             │
│  - CTA link: absolute bottom-[20px] right-[20px]            │
│    "See details" text + arrow circle icon (30x30, rx: 14.75)│
│    Icon bg: #403d37, hover: bg-blue                         │
│  - Link block: absolute bottom-0, left-auto, right-0, top-0 │
│    w-0 → w-full on hover, origin-left, rounded, bg-charcoal │
│    → bg-blue on hover, transition: 300ms                     │
│                                                              │
│  Cards:                                                      │
│  1. Fleet Telematics — icon SVG, "Real-time GPS, routing,    │
│     fuel, compliance"                                        │
│  2. Cameras and Video — icon SVG, "AI dash cams, site        │
│     security, in-cab alerts"                                 │
│  3. Equipment Management — icon SVG, "Diagnostics,           │
│     maintenance, location tracking"                          │
│  4. Samsara Platform — icon SVG, "Open platform, workflows   │
│     & reporting, AI insights"                                │
│  5. Maintenance — icon SVG, "Maximize vehicle uptime and     │
│     minimize costs with real-time diagnostics"              │
│  6. Workforce Management — icon SVG, "Driver & fleet apps,   │
│     worker safety, coaching & training"                     │
│                                                              │
│  Icon SVGs: inline SVGs, viewBox 0 0 2160 2160              │
│  Card images: SVG from ctfassets.net                        │
└──────────────────────────────────────────────────────────────┘
```

---

### SECTION 9: CTA Footer ("Ready to Team Up?")
**Background:** `bg-gray-800` (dark)
**Text color:** `text--all-white`
**Layout:** Full-width dark section with centered content

```
┌──────────────────────────────────────────────────────────────┐
│  bg-gray-800, text--all-white                                │
│                                                              │
│  H2: "Ready to team up?"                                    │
│  color: white, centered                                     │
│                                                              │
│  P: "We're ready when you are. Let's talk about what your   │
│     operation needs."                                        │
│  color: white, centered                                     │
│                                                              │
│  CTA: [Talk to sales] → /pricing                            │
│  White/outlined button style                                │
│                                                              │
│  Noise texture overlay: background-image: url(noise-light.svg)│
│  background-size: auto, repeat                               │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. Component Library Reference

### Eyebrow Pill Badge
```css
.eyebrow-pill {
  display: inline-block;
  padding: 4px 12px;        /* approximate */
  border-radius: 9999px;     /* rounded-full */
  font-size: 14px;           /* text-sm */
  font-weight: 500;
  margin-bottom: 8px;
  background: #007bff;       /* blue */
  color: white;
}
.eyebrow-blue { background: #007bff; }
```

### Primary CTA Button
```css
.btn {
  display: inline-flex;
  align-items: center;
  padding: 12px 24px;
  border-radius: 4px;         /* rounded */
  font-weight: 500;
  font-size: 16px;
  transition: all 0.2s;
}
.btn--primary {
  background: #007bff;
  color: white;
  border: none;
}
.btn--primary:hover {
  background: #0056b3;
}
```

### Tertiary Button (Arrow Link)
```css
.btn--tertiary {
  background: transparent;
  border: none;
  color: currentColor;
  text-decoration: underline;
  text-underline-offset: 2px;
}
.btn--tertiary .btn__icon-wrap svg {
  width: 16px;
  height: 16px;
}
```

### Form Input
```css
.inline-cta-form .formkit-wrapper {
  border-radius: 4px;
  border-width: 1px;
  border-color: #ccc9c1;
  background: white;
  overflow: hidden;
}
.inline-cta-form input {
  border-radius: 0;
  min-height: 48px;
  padding: 0 12px;
  font-size: 1rem;
  font-weight: 500;
  line-height: 1.4;
  color: #1c1917;
}
.inline-cta-form input::placeholder {
  color: #9c9890;
}
.inline-cta-form label {
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 4px;
  color: #403d37;
}
```

### Collection Card (Resource Cards)
```css
.c--collection-card {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  transition: all 300ms ease-in-out;
  border-radius: 4px;
}
.card-image img {
  border-radius: 4px;
}
```

### Overview Card (Product Cards)
```css
.c--card-overview {
  position: relative;
  min-height: 250px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow: hidden;
  border-radius: 4px;
  border: 1px solid transparent;
  transition: all 300ms ease-in-out;
}
.c--card-overview:hover {
  border-color: rgba(0,0,0,0.1);
}
.c--card-overview-link {
  position: absolute;
  bottom: 0; left: 0; right: 0; top: 0;
  z-index: 40;
  width: 100%; height: 100%;
}
.c--card-overview-link-block {
  position: absolute;
  bottom: 0; top: 0; right: 0;
  width: 0;
  transform-origin: left;
  border-radius: 4px;
  background: #403d37;        /* charcoal */
  transition: all 300ms ease-in-out;
}
.group:hover .c--card-overview-link-block {
  width: 100%;
  background: #007bff;        /* blue */
}
```

### Accordion
```css
.ba--base-accordion {
  position: relative;
  width: 100%;
  border-bottom: 0.5px solid #ccc9c1;
}
.ba--base-accordion:first-child {
  border-top: 0.5px solid #ccc9c1;
}
.ba--base-accordion button {
  display: flex;
  width: 100%;
  cursor: pointer;
  align-items: center;
  justify-content: space-between;
  gap: 40px;
  padding: 16px 0;
  text-align: left;
}
.ba--base-accordion h4 {
  font-size: 1.25rem;
  margin-bottom: 0;
  width: 80%;
}
.ba--base-accordion__panel {
  height: 0;
  overflow: hidden;
  transition: height 0.25s ease-in-out;
}
```

### Horizontal Resource Card
```css
.card--layout-horizontal-stack {
  display: flex;
  flex-direction: column;
}
.card--layout-horizontal-stack .content--wrapper-outer {
  display: flex;
  flex-direction: row;
}
.card--layout-horizontal-stack .image--wrapper {
  margin-right: 20px;
  max-width: 167px;
}
.card--layout-horizontal-stack .content--wrapper {
  border-left: 1px solid hsla(0,0%,100%,.5);
  padding-left: 20px;
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
@media (min-width: 1280px) {
  .card--layout-horizontal-stack .image--wrapper {
    margin-right: 1.75rem;
  }
  .card--layout-horizontal-stack .content--wrapper {
    padding-left: 1.75rem;
  }
}
```

---

## 5. Layout Grid System

### Row Container
```css
.row-container {
  width: 100%;
  padding-left: 24px;        /* px-gap2 */
  padding-right: 24px;
  max-width: 1280px;          /* lg: container */
  margin: 0 auto;
  position: relative;
  z-index: 10;
}
```

### Column System
```css
.columns-wrapper {
  display: flex;
  flex-wrap: wrap;
  margin-left: -25px;
  margin-right: -25px;
}
.column {
  padding-left: 25px;
  padding-right: 25px;
}
/* Column widths: */
.column.w-12/12 { width: 100%; }
.column.w-6/12  { width: 50%; }
.column.w-8/12  { width: 66.667%; }
.column.w-4/12  { width: 33.333%; }
```

### Responsive Breakpoints
| Prefix | Min-Width | Usage |
|--------|-----------|-------|
| `sm:` | 640px | Small phones |
| `md:` | 768px | Tablets |
| `lg:` | 1000px | Desktop |
| `xl:` | 1280px | Large desktop |
| `2xl:` | 1440px | Ultra-wide |

---

## 6. Complete Page Wireframe (Dimensions Approximate)

```
┌─────────────────────────────────────────────────────────────────────┐
│ NAV: fixed top-16px, z-50, max-w-1408px, h-56px, bg-gray-50       │
│ [Logo 198x30] [Products] [Solutions] [Resources] [About] [Pricing]│
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ HERO (min-h: 640px) — Video BG                                      │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [G2 Badge 4.5 stars, 4800+ reviews]                        │     │
│ │  H1: "GPS Fleet Tracking" (72px bold white uppercase)       │     │
│ │  P: "Monitor your fleet..." (24px medium white)             │     │
│ │  [Check our prices] CTA button                              │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Complete Visibility" (bg-white, py-16)                    │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: Complete Visibility]                              │     │
│ │  H2: "Track your fleet with real-time GPS data"             │     │
│ │  [4-tab accordion blades: Helicopter view | Smart map       │     │
│ │   overlays | Live location sharing | Geofencing]            │     │
│ │  Image/Video area (border-radius: 4px)                      │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Track Everything" (bg-light, py-16)                       │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: Track Everything]                                │     │
│ │  H2: "Extended tracking of your most mobile assets"         │     │
│ │  [3x horizontal scroll cards: Vehicles | Trailers |        │     │
│ │   Equipment] (min-w: 215px, max-w: 245px each)             │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Routing & Dispatch" (bg-white, py-16)                     │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: Routing & Dispatch]                              │     │
│ │  H2: "Finish routes with fewer miles and vehicles"          │     │
│ │  [3-tab slider: Route optimization | Dispatch |             │     │
│ │   Commercial navigation]                                    │     │
│ │  Full-width video/image area                                │     │
│ │  Description text + [Learn more] CTA                        │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Custom Reporting" (bg-light, py-16)                       │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: Custom Reporting]                                │     │
│ │  H2: "Analyze performance and uncover cost reduction..."   │     │
│ │  [Image: semi-truck aerial] + [3 sub-features:             │     │
│ │   Trip history | Speeding | Time on site]                   │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Resources" (bg-white, py-16)                              │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: Resources]                                       │     │
│ │  H2: "Keep up with our latest news & updates"               │     │
│ │  [2x horizontal resource cards with report covers]          │     │
│ │  Each: Image (167px) | border-left divider | Content        │     │
│ │  [See the report →] CTAs                                    │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: FAQ (bg-white, py-16)                                      │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  [Eyebrow: FAQs] (centered)                                 │     │
│ │  H2: "Learn more about GPS tracking" (centered)             │     │
│ │  [10x accordion items, border-bottom: 0.5px]                │     │
│ │  Each: question (h4) + chevron/plus icon + expandable panel │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ SECTION: "Products That Transform" (bg-light, py-16)                │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  H2: "Products that transform your business"               │     │
│ │  P: "Connect your fleet, equipment, sites..."              │     │
│ │  [6x product cards in horizontal scroll]                   │     │
│ │  Each: SVG icon | H3 title | P description | "See details" │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ CTA FOOTER (bg-gray-800, py-16)                                    │
│ ┌─────────────────────────────────────────────────────────────┐     │
│ │  H2: "Ready to team up?" (white, centered)                 │     │
│ │  P: "We're ready when you are..." (white, centered)        │     │
│ │  [Talk to sales] CTA button (white/outlined)               │     │
│ └─────────────────────────────────────────────────────────────┘     │
│                                                                     │
│ FOOTER (bg-gray-800, text-white)                                   │
│ [Products] [Industries] [Integrations] [Developers]                │
│ [Resources] [Company] [Language selector]                          │
│ [Social icons: Facebook, Twitter, Instagram, LinkedIn, YouTube]    │
│ [Legal links row]                                                   │
│ © 2026 Samsara Inc. All rights reserved.                          │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Image Asset URLs

| Image | CDN URL | Dimensions |
|-------|---------|------------|
| Hero video | `//videos.ctfassets.net/.../Hero.mp4` | — |
| Helicopter view video | `//videos.ctfassets.net/.../Helicopter_View.mp4` | — |
| Routes video | `//videos.ctfassets.net/.../Optimizes-routes.mp4` | — |
| Vehicles card | `https://images.ctfassets.net/.../Vehicles.png` | 676×962 |
| Trailers card | `https://images.ctfassets.net/.../Trailers.png` | 676×962 |
| Equipment card | `https://images.ctfassets.net/.../Equipment.png` | 676×962 |
| Custom Reporting | `https://images.ctfassets.net/.../Custom_Reporting__1_.png` | 2160×1422 |
| Safety Report cover | `https://images.ctfassets.net/.../Report__-_reduce_crash_rates_by_75-.png` | 2160×2160 |
| Fleet Tech Report | `https://images.ctfassets.net/.../fleet-tech-report-resources-image.png` | 2160×2160 |
| G2 Badge | `https://images.ctfassets.net/.../Screenshot_2024-12-18_at_4.42.54_PM-removebg-preview.png` | 2160×674 |
| Meta OG Image | `https://images.ctfassets.net/.../Meta-Image_T2_products__telematics__gps-fleet-tracking.png` | — |
| Product icons (SVG) | `https://images.ctfassets.net/.../Vehicle_Telematics.svg` | 2160×2160 |
| | `https://images.ctfassets.net/.../AI_Camera.svg` | 2160×2160 |
| | `https://images.ctfassets.net/.../Equipment_Monitoring.svg` | 2160×2160 |
| | `https://images.ctfassets.net/.../Open_Platform.svg` | 2160×2160 |
| | `https://images.ctfassets.net/.../Maintenance.svg` | 2160×2160 |
| | `https://images.ctfassets.net/.../Partnership.svg` | 2160×2160 |
| Noise texture | `/_nuxt/textures/noise-light.svg` | — |

---

## 8. Typography Scale Summary

| Element | Size (mobile) | Size (desktop) | Weight | Line Height | Transform |
|---------|---------------|----------------|--------|-------------|-----------|
| H1 | 48px * scale | 72px * scale | 700 | — | uppercase |
| H2 | ~32px | ~40px | 700 | — | normal |
| H3 | 1.25rem (20px) | 1.25rem (20px) | 500 | 1.75rem (28px) | normal |
| H4 | ~1.125rem | ~1.125rem | 500 | 1.5 | normal |
| P (hero) | 20px | 24px | 500 | 120% | normal |
| P (body) | 1rem (16px) | 1rem (16px) | 400 | 1.5rem (24px) | normal |
| P (card-sm) | 0.875rem (14px) | 0.875rem (14px) | 400 | 1.25rem (20px) | normal |
| Eyebrow | 14px (text-sm) | 14px | 500 | — | normal |
| Nav link | 16px | 16px | 500 | — | normal |
| Footer link | ~14px | ~14px | 400 | — | normal |

---

## 9. Animation & Transitions

| Element | Property | Duration | Easing |
|---------|----------|----------|--------|
| Accordion panel | `height` | 0.25s | ease-in-out |
| Accordion mobile content | `grid-template-rows` | 0.3s | ease-in-out |
| Card hover link block | `width`, `background` | 300ms | ease-in-out |
| Card hover border | `border-color` | 300ms | ease-in-out |
| Slider indicator | `scale` | — | — |
| Hero nav `top` | `transition: [top]` | — | — |
| Image frame expand | `height` | 0.3s | ease-in-out |
| Fade enter/leave | `opacity` | 0.15s | ease-in / ease-out |

---

*Document generated from live source inspection of samsara.com/products/telematics/gps-fleet-tracking*
