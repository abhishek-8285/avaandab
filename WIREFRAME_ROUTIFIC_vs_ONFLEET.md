# Detailed Wireframe Document: Routific vs Onfleet Landing Pages

---

## PAGE 1: ROUTIFIC — Smart Route Optimization

### Color Palette & Typography

- **Primary Colors**: White background, dark navy/black text (#1a1a2e inferred), accent green (CTA buttons)
- **Fonts** (from WebFont config):
  - **Headings**: `Secular One` (weights 300–700)
  - **Body**: `Open Sans` (300–800, italic variants)
- **Card Style**: Clean white cards, no heavy borders, soft shadows implied. Sections alternate between white and light gray backgrounds.
- **Logo**: SVG text logo, black on white

---

### NAVIGATION BAR
```
[Logo: Routific SVG]    How it works | Pricing | Integrations | About us
                                                        [Login] [Book demo] [Start free trial]
```
- Two login links: "Login to old Routific" and "Login to new Routific"
- CTA buttons: "Book demo" (outlined) and "Start free trial" (solid)

---

### HERO SECTION
```
┌─────────────────────────────────────────────────────┐
│  H1: "Routes so smart, drivers actually follow them" │
│                                                     │
│  Subtitle: "Routific's smart route optimization      │
│  engine builds routes that are fast, accurate,       │
│  and visually clean — so your team spends less time  │
│  second-guessing and more time delivering."          │
│                                                     │
│  [Image: Optimized route map with mobile app view]   │
└─────────────────────────────────────────────────────┘
```
- Layout: Full-width, centered text above, hero image below
- Image: Route efficiency screenshot showing mobile app integration

---

### SECTION 1: OPTIMIZED
```
┌─────────────────────────────────────────────────────┐
│  Label: "OPTIMIZED" (small caps/eyebrow text)       │
│                                                     │
│  H2: "No more spaghetti routes!"                    │
│                                                     │
│  Body Copy: "A tangled route erodes trust in the    │
│  plan, and in the software. Drivers reorder stops,   │
│  go off-script, or plan their own path. And once a   │
│  driver stops trusting the route, the optimization   │
│  is lost."                                          │
│                                                     │
│  [BEFORE/AFTER COMPARISON VISUAL — side-by-side]     │
│                                                     │
│  LEFT IMAGE: "Competitor spaghetti routes"           │
│    → Tangled, overlapping colored route lines        │
│    → Alt text: "Competitive route optimization       │
│       solvers typically optimize for mathematical     │
│       efficiency alone, often creating entangled     │
│       spaghetti routes."                             │
│                                                     │
│  RIGHT IMAGE: "Routific clean routes"                │
│    → Clean, non-overlapping, geographically logical  │
│       colored route clusters                         │
│    → Alt text: "Routific's route optimization        │
│       algorithm produces clean, non-overlapping      │
│       routes..."                                     │
└─────────────────────────────────────────────────────┘
```
- **Pattern**: Side-by-side before/after comparison (competitor vs. Routific)
- **Visual style**: Map-based route visualizations, colored lines by route

---

### SECTION 2: ACCURATE
```
┌─────────────────────────────────────────────────────┐
│  Label: "ACCURATE" (eyebrow)                        │
│                                                     │
│  H2: "ETAs built on real-world traffic data"        │
│                                                     │
│  Body: "Distance alone doesn't determine how long    │
│  a delivery takes. Traffic is the missing ingredient │
│  in many route planners. Routific's algorithm        │
│  incorporates historical traffic data to build       │
│  routes and ETAs that reflect how roads actually     │
│  behave, not just how far apart stops are."         │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  SUB-SECTION: 3 accordion/dropdown cards     │   │
│  │  (chevron-left icon on each)                 │   │
│  │                                              │   │
│  │  1. "Traffic-informed routing"               │   │
│  │     Body: "Most routing tools calculate      │   │
│  │     distance, then apply traffic as an       │   │
│  │     afterthought. Routific builds traffic    │   │
│  │     directly into the optimization..."       │   │
│  │                                              │   │
│  │  2. "Accurate ETAs at dispatch"              │   │
│  │     Body: "A delivery window is only useful  │   │
│  │     if it's realistic. When ETAs are based   │   │
│  │     on distance alone, they can be           │   │
│  │     over-optimistic..."                      │   │
│  │                                              │   │
│  │  3. "Driver speeds"                          │   │
│  │     Body: "Every driver moves at a different │   │
│  │     pace – on the road and while carrying    │   │
│  │     out deliveries. Routific lets you        │   │
│  │     adjust each driver's speed               │   │
│  │     individually..."                         │   │
│  └──────────────────────────────────────────────┘   │
│                                                     │
│  [3 stacked images, right-aligned]                   │
│  1. Predicted traffic map visualization              │
│  2. Route list with ETA estimates                    │
│  3. Driver speeds comparison UI                     │
└─────────────────────────────────────────────────────┘
```
- **Layout**: Left text + right stacked images; accordion expandable cards
- **Interactive element**: Collapsible sub-cards with chevron rotation

---

### SECTION 3: FAST
```
┌─────────────────────────────────────────────────────┐
│  Label: "FAST" (eyebrow)                            │
│                                                     │
│  H2: "Built to grow with you"                       │
│                                                     │
│  Body: "As your business grows, so do your route     │
│  planning needs. More orders, more drivers, more    │
│  complexity. Routific scales with you — keeping      │
│  route planning fast and simple no matter how much   │
│  your operation expands.                            │
│                                                     │
│  Our algorithm handles up to 5,000 orders in a      │
│  single optimization. Routes load in seconds, the   │
│  interface stays snappy, and your dispatcher never  │
│  waits on the software to catch up.                 │
│                                                     │
│  Your job is to grow the business. Ours is to keep  │
│  the deliveries efficient."                         │
│                                                     │
│  [Image: Line graph of growing order volumes]        │
│  → SVG, abstract growth curve                        │
└─────────────────────────────────────────────────────┘
```
- **Stat callout**: "up to 5,000 orders in a single optimization"
- **Visual**: Single growth chart (SVG)

---

### SECTION 4: FLEXIBLE
```
┌─────────────────────────────────────────────────────┐
│  Label: "FLEXIBLE" (eyebrow)                        │
│                                                     │
│  H2: "The algorithm is smart. Your team is smarter." │
│                                                     │
│  Body: "Routific's optimization engine doesn't       │
│  replace your dispatcher's expertise — it amplifies  │
│  it. When local knowledge, specialized drivers, or   │
│  specific constraints come into play, you stay in   │
│  control."                                          │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  SUB-SECTION: 3 accordion/dropdown cards     │   │
│  │                                              │   │
│  │  1. "Draw your route"                        │   │
│  │     "Guide the flow of any route directly    │   │
│  │     on the map. Routific re-optimizes around │   │
│  │     your input instantly..."                 │   │
│  │                                              │   │
│  │  2. "Driver tags and skills"                 │   │
│  │     "Match orders to the right driver or     │   │
│  │     vehicle every time. Assign tags for      │   │
│  │     driver preference, vehicle type,         │   │
│  │     certifications..."                       │   │
│  │                                              │   │
│  │  3. "Load and time-window constraints"       │   │
│  │     "Set capacity limits and shift times     │   │
│  │     across your fleet. Routific builds       │   │
│  │     routes that respect your real-world      │   │
│  │     constraints."                            │   │
│  └──────────────────────────────────────────────┘   │
│                                                     │
│  [3 stacked images, right-aligned]                   │
│  1. Draw-route feature (patent-pending)              │
│  2. Route tags assignment UI                         │
│  3. Route constraints UI                             │
└─────────────────────────────────────────────────────┘
```

---

### SECTION 5: CONSTRAINTS GRID (An algorithm that works within your real-world rules)
```
┌─────────────────────────────────────────────────────┐
│  H2: "An algorithm that works within your            │
│       real-world rules"                              │
│                                                     │
│  Body: "Every delivery operation has constraints —   │
│  time windows, vehicle capacities, shift times,      │
│  local geography. Routific's algorithm respects     │
│  them all. It calculates optimal departure times,   │
│  sends drivers to actual delivery entrances, and    │
│  automatically slots last-minute orders into the    │
│  best position on an active route. When you need    │
│  flexibility, soft constraints let you specify      │
│  which rules can bend."                             │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  3×3 GRID OF CONSTRAINT CARDS                │   │
│  │  (icon + title + description + "See how" CTA)│   │
│  │                                              │   │
│  │  1. ⏰ Time windows                          │   │
│  │     "Deliver within the windows your         │   │
│  │     customers expect"                        │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  2. 📦 Capacities and loads                  │   │
│  │     "Match orders to vehicles without        │   │
│  │     overloading your fleet"                  │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  3. 🕐 Flexible start times                  │   │
│  │     "Calculate the optimal departure time    │   │
│  │     to arrive just-in-time for your first    │   │
│  │     stop"                                    │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  4. 📍 Route to delivery location            │   │
│  │     "Routes stop at the right entrance,      │   │
│  │     not just the right address"              │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  5. ⚖️ Balance routes                        │   │
│  │     "Distribute work fairly across your      │   │
│  │     driver fleet"                            │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  6. 📉 Minimize routes                       │   │
│  │     "Use the fewest routes needed to get     │   │
│  │     the job done"                            │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  7. 🏔️ Terrain awareness                     │   │
│  │     "Accounts for mountains, bridges, and    │   │
│  │     rivers that add real travel time"        │   │
│  │                                              │   │
│  │  8. 📌 Best-insert                           │   │
│  │     "Slots last-minute orders into the best  │   │
│  │     position without rebuilding the route"   │   │
│  │     [See how →]                              │   │
│  │                                              │   │
│  │  9. 🔄 Soft constraints                      │   │
│  │     "When routes need flexibility, tell the  │   │
│  │     algorithm which rules can bend"          │   │
│  │     [See how →]                              │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Layout**: 3-column grid (or responsive 3-col → 1-col)
- **Card style**: SVG icon + bold title + short description + link CTA
- **Links**: Route to help.routific.com articles

---

### SECTION 6: COMING SOON — Predictive Traffic
```
┌─────────────────────────────────────────────────────┐
│  Label: "COMING SOON" (eyebrow)                     │
│                                                     │
│  H2: "Routing that knows rush hour before your      │
│       drivers do"                                   │
│                                                     │
│  Body: "Traffic has patterns. A Thursday morning     │
│  route and a Friday evening route cover the same    │
│  roads, but they're not the same drive. Routific's  │
│  predictive traffic feature accounts for that       │
│  difference, building routes and ETAs around how    │
│  roads actually behave at the time your drivers     │
│  are on them.                                       │
│                                                     │
│  Routing tools often underestimate travel time      │
│  between stops. In New York City, that gap          │
│  averages 6 minutes per stop. On a single route     │
│  with 30 orders, that's 3 hours of unplanned       │
│  overtime for one driver — multiplied across your   │
│  entire fleet, every day.                           │
│                                                     │
│  *This R&D work is funded in part by Scale AI and   │
│  the Canada Innovation Corporation.*"               │
│                                                     │
│  [Image: Large predicted traffic visualization]      │
│  → Color-coded traffic pattern map                   │
└─────────────────────────────────────────────────────┘
```
- **Stat callout**: "6 minutes per stop" gap in NYC, "3 hours of unplanned overtime" for one 30-stop route

---

### SECTION 7: TRUST / SOCIAL PROOF
```
┌─────────────────────────────────────────────────────┐
│  H2: "Voted #1 route planning software"             │
│                                                     │
│  ★★★★★  4.9 on Capterra (144 reviews)              │
│                                                     │
│  QUOTE: "The ease of use, simplicity, and customer  │
│  service is second to none. I have tried and demoed  │
│  every routing tool out there and none come close    │
│  to Routific."                                     │
│  — Morgan H, CEO (Capterra review)                  │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  3 BADGE IMAGES (Capterra/SoftwareAdvice/    │   │
│  │  GetApp award badges)                        │   │
│  │                                              │   │
│  │  1. Capterra "Best Ease of Use 2025"         │   │
│  │  2. SoftwareAdvice "Best Customer Support    │   │
│  │     2025"                                    │   │
│  │  3. GetApp "Best Functionality & Features    │   │
│  │     2025"                                    │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Badge style**: Official review site award badges with logos + category text

---

### SECTION 8: CUSTOMER LOGO CAROUSEL
```
┌─────────────────────────────────────────────────────┐
│  [Scrolling logo carousel — horizontal scroll]       │
│                                                     │
│  LOGOS (in order):                                   │
│  1. Reverend Nat's Hard Cider (Portland cidery)     │
│  2. Flourist (Vancouver bakery)                     │
│  3. Hand Up Toronto (non-profit)                    │
│  4. Marché SecondLife (Montreal grocery)            │
│  5. Walden Local (New England meat subscription)    │
│  6. Trunkrs (Dutch parcel carrier)                  │
│  7. Empire Furniture (white-glove delivery)         │
│  8. Logismith (US logistics partner)                │
│  9. Shinsen AG / Bowlz (Zurich meal delivery)       │
│  10. 4P Foods (DC-area farm-to-home)                │
│                                                     │
│  [Logos repeat/loop in carousel]                     │
└─────────────────────────────────────────────────────┘
```
- **Logo style**: Grayscale/low-opacity brand logos, uniform height, horizontally scrollable

---

### SECTION 9: BOTTOM CTA
```
┌─────────────────────────────────────────────────────┐
│  H2: "Smart route optimization for your delivery    │
│       business"                                     │
│                                                     │
│  [Start for free]  [Book a demo]                     │
│  (solid green)    (outlined)                        │
└─────────────────────────────────────────────────────┘
```

---

### FOOTER
```
┌─────────────────────────────────────────────────────┐
│  [Routific logo]  "Proudly part of the Techstars    │
│                    family" [Techstars logo]          │
│                                                     │
│  Product          Developers       Support           │
│  How it works     Integrations     About us          │
│  Route planning   Route optimization API  Help center│
│  Smart route opt  (dev.routific)   Blog              │
│  Customer stories                  Roadmap            │
│  Pricing                           Status page        │
│  Product news                                       │
│                                                     │
│  [LinkedIn] [Instagram] [Facebook]                  │
│  Privacy policy | Terms of service                  │
│  © 2026 Routific Solutions Inc. All rights reserved. │
└─────────────────────────────────────────────────────┘
```

---

### ROUTIFIC PAGE SUMMARY

| Element | Pattern |
|---------|---------|
| **Page structure** | Hero → 4 keyword sections (OPTIMIZED/ACCURATE/FAST/FLEXIBLE) → Constraints grid → Coming soon → Trust badges → Logo carousel → CTA → Footer |
| **Keyword sections** | Eyebrow label (ALL CAPS) + H2 + body text + expandable sub-cards (accordion) + stacked images |
| **Before/after** | Side-by-side map comparison (competitor spaghetti vs. Routific clean) |
| **Trust** | 4.9 stars, 144 reviews, 3 award badges, 1 customer quote |
| **Stat callouts** | "5,000 orders", "6 minutes per stop in NYC", "3 hours unplanned overtime" |
| **CTAs** | "Start free trial" (primary), "Book demo" (secondary) |
| **FAQ** | No explicit FAQ section on this page |
| **Color** | White bg, green CTAs, dark text |
| **Typography** | Secular One (headings), Open Sans (body) |

---
---

## PAGE 2: ONFLEET — Delivery Management

### Color Palette & Typography

- **Primary Colors**: White background, dark text, purple accent (#6B4EAA inferred from SVG paths)
- **Fonts**: System/proprietary sans-serif (not Google Fonts; loaded from CDN)
- **Card Style**: Clean white cards, minimal borders, CSS class `common-Header*` system
- **Logo**: Custom SVG bird/origami mark + "Onfleet" wordmark

---

### NAVIGATION BAR
```
┌─────────────────────────────────────────────────────┐
│  [Onfleet SVG logo (bird mark)]    [Compact nav]    │
└─────────────────────────────────────────────────────┘
```
- Minimal nav: Logo only visible in HTML; other nav items not rendered in page source (likely hamburger menu on mobile)

---

### HERO SECTION
```
┌─────────────────────────────────────────────────────┐
│                                                     │
│  H1: "Power                                        │
│       your                                          │
│       deliveries"                                   │
│  (line-broken, stacked vertically)                  │
│                                                     │
│  Body: "Onfleet is the leading delivery management  │
│  software for all industries.                       │
│  We provide the easiest way to manage your          │
│  deliveries. Route, dispatch, track, and analyze    │
│  your fleet while delighting your customers."       │
│                                                     │
│  [Image: Onfleet truck illustration, @2x PNG]        │
│  → "onfleet-industry-truck@2x.png"                  │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  STICKY CTA BUTTON:                          │   │
│  │  "Contact Sales" [arrow icon →]              │   │
│  │  (purple, sticky, animated)                  │   │
│  └──────────────────────────────────────────────┘   │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  CONTACT FORM (right sidebar / below hero)   │   │
│  │  "Contact sales"                             │   │
│  │  [HubSpot form embed]                        │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Layout**: Left-aligned text + image, right sidebar sticky contact form
- **CTA**: Purple "Contact Sales" button with arrow icon, sticky on scroll
- **Form**: HubSpot-embedded sales form, two variants (desktop sidebar + mobile)

---

### SECTION 1: THE ONSFLEET ADVANTAGE
```
┌─────────────────────────────────────────────────────┐
│  H2: "The Onfleet advantage"                        │
│                                                     │
│  ✓ Cut down on costs by gaining 20-40% efficiency   │
│    from more intelligent routes                     │
│                                                     │
│  ✓ Visibility to goods and drivers in real-time     │
│    for managers                                     │
│                                                     │
│  ✓ The ability to deliver to locations that may not │
│    yet have a real address                          │
│                                                     │
│  ✓ Reduces the manual tasks of routing delivery     │
│    routes                                           │
│                                                     │
│  [Checkmark icons: SVG polyline, 18×18, black]      │
└─────────────────────────────────────────────────────┘
```
- **Layout**: Checklist format, 4 bullet points with SVG checkmarks
- **Stat callout**: "20-40% efficiency"

---

### SECTION 2: TRUSTED BY THOUSANDS (Social Proof)
```
┌─────────────────────────────────────────────────────┐
│  H2: "Trusted by thousands of businesses"           │
│  Sub: "The most successful delivery operations      │
│        choose Onfleet."                             │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  LOGO STRIP (horizontal, 4 logos visible)    │   │
│  │                                              │   │
│  │  1. Kroger (blue SVG logo)                   │   │
│  │  2. The Wonderful Company (colorful SVG)     │   │
│  │  3. Sweetgreen (green gradient SVG)          │   │
│  │  4. FreshDirect (orange/green SVG)           │   │
│  └──────────────────────────────────────────────┘   │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  AWARD CARDS (3 side-by-side)                │   │
│  │                                              │   │
│  │  Card 1:                                     │   │
│  │    [GetApp logo]                             │   │
│  │    4.7 ★★★★★                                │   │
│  │    "95 reviews"                              │   │
│  │    "Top-rated Last Mile Delivery Software"   │   │
│  │                                              │   │
│  │  Card 2:                                     │   │
│  │    [GetApp logo]                             │   │
│  │    4.7 ★★★★★                                │   │
│  │    "95 reviews"                              │   │
│  │    "Top 20 Fleet Management Software"        │   │
│  │                                              │   │
│  │  Card 3:                                     │   │
│  │    [GetApp logo]                             │   │
│  │    4.6 ★★★★★                                │   │
│  │    "136 reviews"                             │   │
│  │    "Route Planning and Fleet Management      │   │
│  │     Leader"                                  │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Logo style**: Full-color SVG logos, uniform height, grayscale-to-color on hover (CSS class `selected_logos_box`)
- **Award cards**: Single card with logo + rating + review count + category title

---

### SECTION 3: CUSTOMER QUOTE STRIP (Under Hero / Social Proof)
```
┌─────────────────────────────────────────────────────┐
│  HORIZONTAL SCROLLING QUOTE CAROUSEL                 │
│                                                     │
│  QUOTE 1:                                           │
│  "Onfleet is a no-brainer for any serious delivery  │
│  operation. We were able to not only increase our   │
│  delivery capacity by 50% using their route         │
│  optimization engine, but we have also improved     │
│  on-time rates and customer satisfaction through    │
│  accurate ETAs and real-time visibility."           │
│  — Emily Ehrlich, eCommerce Coordinator,            │
│    The United Family                                │
│  [Logo: United Supermarkets]                        │
│                                                     │
│  QUOTE 2:                                           │
│  "Onfleet has streamlined our logistics, freeing    │
│  us to focus more time and energy on the big        │
│  picture. Most importantly, our customers love the  │
│  tracking and automated delivery alert features."   │
│  — Zach Nelkin, Chief Strategy Officer,             │
│    Hungry Harvest                                   │
│  [Logo: Hungry Harvest]                             │
│                                                     │
│  QUOTE 3:                                           │
│  "It's been so easy to implement Onfleet and it's   │
│  helped us increase customer satisfaction and       │
│  better manage our deliveries."                     │
│  — Benjamin Chesler, Co-Founder and COO,            │
│    Imperfect Foods                                  │
│  [Logo: Imperfect Foods]                            │
│                                                     │
│  QUOTE 4:                                           │
│  "Freshop powers online grocery for 1,000+ grocery  │
│  stores. Onfleet's great partnership allows stores  │
│  to do the last mile on time and on target."        │
│  — Brian Moyer, CEO, Freshop                        │
│  [Logo: Freshop]                                    │
└─────────────────────────────────────────────────────┘
```
- **Pattern**: Horizontal carousel of testimonial cards, each with: logo + quote text + author name + title + company
- **Key stat from quote**: "50% increase in delivery capacity" (United Family)

---

### SECTION 4: UTILITY BUILT FOR DELIVERY (Feature Grid — Part 1)
```
┌─────────────────────────────────────────────────────┐
│  H2: "Utility built for delivery"                   │
│                                                     │
│  Sub: "Onfleet's software has the features all      │
│  business owners need to launch or upgrade their    │
│  delivery operations:"                              │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  2×2 FEATURE GRID (image left/right swap)    │   │
│  │                                              │   │
│  │  1. Proof of delivery                        │   │
│  │     [Screenshot: proof-of-delivery.jpg]      │   │
│  │     "Enforce completion requirements through │   │
│  │     in-app collection of photos, signatures, │   │
│  │     barcodes and notes."                     │   │
│  │                                              │   │
│  │  2. Notify Customers                         │   │
│  │     [Screenshot: automatic-status-updates]   │   │
│  │     "With automated SMS notifications        │   │
│  │     customers know when a delivery has       │   │
│  │     started, when it should be expected,     │   │
│  │     and when it's arriving."                 │   │
│  │                                              │   │
│  │  3. Private Communications                   │   │
│  │     [Screenshot: customer-communication]     │   │
│  │     "Customers may call or message their     │   │
│  │     driver, dispatcher, or call center with  │   │
│  │     a single tap. Anonymize calls to         │   │
│  │     safeguard privacy."                      │   │
│  │                                              │   │
│  │  4. Contactless Signatures                   │   │
│  │     [Screenshot: contactless-signatures]     │   │
│  │     "Companies can serve signatures for      │   │
│  │     deliveries via SMS, for the recipient to │   │
│  │     sign from the safety of their home..."   │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Layout**: Alternating image/text rows (zig-zag pattern)
- **Image style**: Compact screenshots with shadows

---

### SECTION 5: DISPATCH & MANAGE (Feature Grid — Part 2)
```
┌─────────────────────────────────────────────────────┐
│  H2: "Effortlessly dispatch and manage your         │
│       deliveries"                                   │
│  Sub: "Monitor and control your delivery business   │
│  as it grows with these key features"              │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  3-COLUMN FEATURE CARDS                      │   │
│  │                                              │   │
│  │  1. Dispatching & Routing                    │   │
│  │     [Screenshot: route-optimization.jpg]     │   │
│  │     "Onfleet's integrated route optimization │   │
│  │     engine considers time, location,         │   │
│  │     capacity, and traffic to produce the     │   │
│  │     most efficient routing solutions..."     │   │
│  │                                              │   │
│  │  2. Driver Tracking & Communication          │   │
│  │     [Screenshot: team-communication.jpg]     │   │
│  │     "Provide live driver locations with      │   │
│  │     accurate ETAs for fleetwide              │   │
│  │     transparency and granular product        │   │
│  │     tracking."                               │   │
│  │                                              │   │
│  │  3. Analyze & Export Data                    │   │
│  │     [Screenshot: key-metrics.jpg]            │   │
│  │     "Visualize success rates, on-time rates, │   │
│  │     service times, feedback scores, distance │   │
│  │     traveled, and more. Segment data by      │   │
│  │     teams, drivers, day, week, and even hour │   │
│  │     of the day."                             │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```
- **Layout**: 3-column horizontal card grid
- **Image**: Screenshots in compact format

---

### SECTION 6: IMPROVE DELIVERY EXPERIENCE (Feature Grid — Part 3)
```
┌─────────────────────────────────────────────────────┐
│  H2: "Improve your delivery experience"             │
│  Sub: "Increase customer engagement and satisfaction │
│  with Onfleet delivery software."                   │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  4-ITEM HORIZONTAL GRID (alternating images) │   │
│  │                                              │   │
│  │  1. Status updates                           │   │
│  │     [Screenshot: automatic-status-updates]   │   │
│  │     "Track your fleet from your dispatch     │   │
│  │     center, identify potential delays, and   │   │
│  │     reroute drivers if needed."              │   │
│  │                                              │   │
│  │  2. Real-time GPS driver tracking            │   │
│  │     [Screenshot: real-time-driver-tracking]  │   │
│  │     "Give customers insight into where their │   │
│  │     delivery driver is and when their product│   │
│  │     will arrive."                            │   │
│  │                                              │   │
│  │  3. Customer communications                  │   │
│  │     [Screenshot: customer-communication]     │   │
│  │     "Let customers chat with drivers directly│   │
│  │     through Onfleet."                        │   │
│  │                                              │   │
│  │  4. Feedback collection                      │   │
│  │     [Screenshot: customer-feedback.jpg]      │   │
│  │     "Request feedback via SMS immediately    │   │
│  │     after deliveries."                       │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

---

### SECTION 7: INTEGRATIONS (Logo Strip)
```
┌─────────────────────────────────────────────────────┐
│  H2: "Integrated with the tools you love"           │
│                                                     │
│  Sub: "Onfleet's delivery software is integrated    │
│  out of the box with your favorite point of sale,   │
│  online store ordering, and payments solutions so   │
│  you can be up and running in no time."             │
│                                                     │
│  ┌──────────────────────────────────────────────┐   │
│  │  INTEGRATION PARTNER CARDS (logo + quote)    │   │
│  │                                              │   │
│  │  1. MI9 Retail (partner card)                │   │
│  │     Logo: MI9 SVG                            │   │
│  │     "Mi9 Retail is an enterprise software    │   │
│  │     provider for large retailers, wholesalers│   │
│  │     and brands. Onfleet's API integration    │   │
│  │     enables fast and easy connection between │   │
│  │     the Mi9 / MWG platform..."              │   │
│  │                                              │   │
│  │  2. Freshop (partner card)                   │   │
│  │     Logo: Freshop SVG                        │   │
│  │     [Logo description: "Onfleet Partner      │   │
│  │     Freshop"]                                │   │
│  │                                              │   │
│  │  3. Shopify (integration)                    │   │
│  │     "Onfleet Integration Shopify"            │   │
│  │                                              │   │
│  │  4. Square (integration)                     │   │
│  │     "Onfleet Integration Square"             │   │
│  │                                              │   │
│  │  5. Grocerkey (integration)                  │   │
│  │     "Onfleet Integration Grocerkey"          │   │
│  └──────────────────────────────────────────────┘   │
│                                                     │
│  INTEGRATION LOGO STRIP:                            │
│  [MI9] [Freshop] [Shopify] [Square] [Grocerkey]    │
└─────────────────────────────────────────────────────┘
```
- **Partner logos**: Full SVG logos with descriptive partner callouts
- **Integration logos**: Named integration cards (Shopify, Square, Grocerkey)

---

### FOOTER
```
┌─────────────────────────────────────────────────────┐
│  [Onfleet logo SVG]                                 │
│                                                     │
│  Onfleet, Inc. © 2026. All rights reserved.         │
│  Terms of Service ➜ | Privacy Notice ➜              │
│  Cookie Notice ➜ | Accessibility ➜                  │
│                                                     │
│  [Log In]                                            │
└─────────────────────────────────────────────────────┘
```

---

### ONSFLEET PAGE SUMMARY

| Element | Pattern |
|---------|---------|
| **Page structure** | Hero + sticky contact form → Advantage checklist → Trust logos + awards → Customer quotes → 3 feature grid sections → Integrations → Footer |
| **Hero** | 3-line stacked H1 + truck illustration + sticky sidebar contact form |
| **Trust** | 4.7/4.6 star ratings, 95–136 reviews, 3 award cards (GetApp) |
| **Quote strip** | 4 customer testimonials in horizontal carousel with company logos |
| **Feature grids** | 3 separate sections, alternating zig-zag and column layouts |
| **Integration logos** | MI9, Freshop, Shopify, Square, Grocerkey |
| **Stat callouts** | "20-40% efficiency", "50% delivery capacity increase" (from quote) |
| **CTA** | "Contact Sales" (primary, purple, sticky) |
| **FAQ** | No explicit FAQ section on this page |
| **Color** | White bg, purple CTAs (#6B4EAA), black text |
| **Typography** | System sans-serif |

---
---

## COMPARATIVE ANALYSIS

### Page Structure Comparison

| Feature | Routific | Onfleet |
|---------|----------|---------|
| **Primary CTA** | "Start free trial" / "Book demo" | "Contact Sales" |
| **CTA Color** | Green | Purple |
| **Hero Image** | Route map screenshot | Truck illustration |
| **Social Proof Position** | Bottom (after features) | Top (immediately after hero) |
| **Feature Organization** | By benefit keyword (OPTIMIZED/ACCURATE/FAST/FLEXIBLE) | By use case (Utility/Dispatch/Experience) |
| **Before/After Visual** | Yes (spaghetti vs. clean routes) | No |
| **Customer Quotes** | 1 quote | 4-quote carousel |
| **Trust Badges** | 3 award badges (Capterra × 3) | 3 award cards (GetApp × 3) |
| **Logo Strip Position** | Bottom (10 logos) | Top (4 logos) |
| **FAQ Section** | None | None |
| **Contact Form** | No (links to HubSpot meeting) | Yes (inline HubSpot form) |
| **Sticky Elements** | None visible | Sticky CTA button + sticky form |
| **Collapsible Content** | Yes (accordion cards) | No |
| **Coming Soon Section** | Yes (Predictive Traffic) | No |
| **API/Developer Link** | Yes (dev.routific.com) | No |

### Typography Comparison

| Aspect | Routific | Onfleet |
|--------|----------|---------|
| **Heading Font** | Secular One | System sans-serif |
| **Body Font** | Open Sans | System sans-serif |
| **Font Loading** | Google Fonts (webfont.js) | CDN stylesheet |

### Card & Layout Styles

| Aspect | Routific | Onfleet |
|--------|----------|---------|
| **Card Borders** | None (clean white) | Minimal (CSS-driven) |
| **Section Backgrounds** | Alternating white/gray | White with dividers |
| **Grid System** | 3-column constraint grid | 2×2 and 3-column feature grids |
| **Image Treatment** | Map screenshots + SVG graphics | App screenshots + illustrations |
| **Interactive Elements** | Accordion expand/collapse | Static display |

### Trust & Credibility Elements

| Element | Routific | Onfleet |
|---------|----------|---------|
| **Rating** | 4.9/5 | 4.6–4.7/5 |
| **Review Count** | 144 (Capterra) | 95–136 (GetApp) |
| **Award Badges** | Best Ease of Use, Best Support, Best Features (2025) | Top-rated, Top 20, Leader (GetApp) |
| **Customer Logos** | 10 niche/regional brands | 4 major enterprise brands (Kroger, Sweetgreen, etc.) |
| **Testimonials** | 1 Capterra review | 4 detailed customer quotes |
| **Industry Focus** | Food delivery, grocery, meal kits | Grocery, food delivery, general retail |

---
---

*Document generated from live page analysis of routific.com/smart-route-optimization and onfleet.com/delivery-management as of August 2026.*
