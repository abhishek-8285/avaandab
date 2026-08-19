# Premium B2B SaaS Design System
*Extracted from Stripe, Linear, Vercel, Notion — ready for Tailwind CSS implementation*

---

## 1. Color Palette

### Core Colors
| Token | Hex | Usage |
|-------|-----|-------|
| `--color-bg-primary` | `#ffffff` / `#08090a` (dark) | Page background |
| `--color-bg-secondary` | `#f9f8f9` / `#1c1c1f` (dark) | Card/section bg |
| `--color-bg-tertiary` | `#f4f2f4` / `#232326` (dark) | Subtle bg |
| `--color-bg-quaternary` | `#eeedef` / `#28282c` (dark) | Hover states |
| `--color-bg-marketing` | `#010102` | Linear's dark hero |
| `--color-bg-translucent` | `rgba(0,0,0,0.02)` | Glass effect |

### Text Colors
| Token | Hex | Usage |
|-------|-----|-------|
| `--color-text-primary` | `#282a30` / `#f7f8f8` (dark) | Headlines |
| `--color-text-secondary` | `#3c4149` / `#d0d6e0` (dark) | Body text |
| `--color-text-tertiary` | `#6f6e77` / `#8a8f98` (dark) | Captions |
| `--color-text-quaternary` | `#86848d` / `#62666d` (dark) | Muted labels |

### Brand / Accent Colors
| Token | Hex | Source |
|-------|-----|--------|
| Stripe Primary (Slate) | `#0a2540` | Stripe |
| Stripe Purple | `#635bff` | Stripe |
| Stripe Teal | `#11efe3` | Stripe |
| Stripe Cyan | `#02bcf5` | Stripe |
| Stripe Pink | `#ff5996` | Stripe |
| Linear Indigo | `#5e6ad2` / `#7070ff` | Linear |
| Linear Accent (dark) | `#828fff` | Linear |
| Vercel Black | `#000000` | Vercel |
| Notion Red accent | `#ff5996` | Notion |

### Border / Surface Colors
| Token | Hex | Usage |
|-------|-----|-------|
| `--color-border-primary` | `#e9e8ea` / `#23252a` (dark) | Card borders |
| `--color-border-secondary` | `#e4e2e4` / `#34343a` (dark) | Dividers |
| `--color-border-tertiary` | `#dcdbdd` / `#3e3e44` (dark) | Subtle lines |
| `--color-border-translucent` | `rgba(255,255,255,0.05)` | Glass borders |

---

## 2. Typography Scale

### Font Stacks
```css
--font-regular: "Inter Variable", "SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
--font-monospace: "Berkeley Mono", "SF Mono", "Menlo", monospace;
--font-serif-display: "Tiempos Headline", ui-serif, Georgia, Cambria, serif;
```

### Size Scale (Linear's exact values)
| Token | Size | Line Height | Letter Spacing | Usage |
|-------|------|-------------|----------------|-------|
| `text-tiny` | `10px` (.625rem) | 1.5 | -.015em | Micro labels |
| `text-micro` | `12px` (.75rem) | 1.4 | 0 | Tag/badge |
| `text-mini` | `13px` (.8125rem) | 1.5 | -.01em | Caption |
| `text-small` | `14px` (.875rem) | 1.5 | -.013em | Small body |
| `text-regular` | `15px` (.9375rem) | 1.6 | -.011em | Body |
| `text-large` | `18px` (1.0625rem) | 1.6 | 0 | Large body |
| `title-1` | `17px` (1.0625rem) | 1.4 | -.012em | Card title |
| `title-2` | `20px` (1.25rem) | 1.33 | -.012em | Section sub |
| `title-3` | `24px` (1.5rem) | 1.33 | -.012em | H3 |
| `title-4` | `32px` (2rem) | 1.125 | -.022em | H2 |
| `title-5` | `40px` (2.5rem) | 1.1 | -.022em | H1 (small) |
| `title-6` | `48px` (3rem) | 1 | -.022em | H1 (medium) |
| `title-7` | `56px` (3.5rem) | 1.1 | -.022em | H1 (large) |
| `title-8` | `64px` (4rem) | 1.06 | -.022em | Display |
| `title-9` | `72px` (4.5rem) | 1 | -.022em | Hero |

### Font Weights
| Token | Value | Usage |
|-------|-------|-------|
| Light | 300 | Subtle text |
| Normal | 400 | Body |
| Medium | 510 | Labels, buttons |
| Semibold | 590 | Subheadings, nav |
| Bold | 680 | Emphasis, strong |

---

## 3. Spacing System

### Base Spacing (Linear's grid)
| Value | Pixels | Usage |
|-------|--------|-------|
| `gap-1` | 2px | Tight inline |
| `gap-2` | 4px | Icon-label |
| `gap-3` | 6px | Compact |
| `gap-4` | 8px | Default compact |
| `gap-5` | 10px | Card internal |
| `gap-6` | 12px | Form elements |
| `gap-8` | 16px | Standard |
| `gap-10` | 20px | Section internal |
| `gap-12` | 24px | Between sections |
| `gap-16` | 32px | Large gap |
| `gap-20` | 40px | Section spacing |
| `gap-24` | 48px | Major sections |
| `gap-32` | 64px | Page padding |

### Page-Level Spacing
```css
--page-padding-inline: 24px;          /* Mobile */
--page-padding-block: 64px;           /* Section vertical */
--homepage-outer-padding: 46px;       /* Desktop outer */
--homepage-padding-inset: 32px;       /* Content inset */
--page-max-width: 1024px;             /* Prose container */
--homepage-max-width: 1344px;         /* Full-width */
```

---

## 4. Border Radius Scale

| Token | Value | Usage |
|-------|-------|-------|
| `radius-4` | 4px | Badges, small buttons |
| `radius-6` | 6px | Inputs, buttons |
| `radius-8` | 8px | Cards, dropdowns |
| `radius-12` | 12px | Large cards |
| `radius-16` | 16px | Feature cards (Linear) |
| `radius-24` | 24px | Icons, large avatars |
| `radius-32` | 32px | Image containers |
| `radius-rounded` | 9999px | Pills, avatars |

---

## 5. Shadow System

### Linear (exact values)
```css
--shadow-tiny: 0px 1px 1px 0px rgba(0,0,0,0.09);
--shadow-low: 0px 1px 4px -1px rgba(0,0,0,0.09);
--shadow-medium: 0px 3px 12px rgba(0,0,0,0.09);
--shadow-high: 0px 7px 24px rgba(0,0,0,0.06);

/* Dark theme */
--shadow-low: 0px 2px 4px rgba(0,0,0,0.1);
--shadow-medium: 0px 4px 24px rgba(0,0,0,0.12);
--shadow-high: 0px 7px 32px rgba(0,0,0,0.35);
```

### Stripe Card Shadow
```css
box-shadow: 0 0 0 1px rgba(255,255,255,0.12),
            0 4px 24px 0 rgba(0,0,0,0.12),
            0 0 0 1px rgba(0,0,0,0.06);
```

### Stripe Sticky Header Shadow
```css
box-shadow: 0 0 60px rgba(50,50,93,0.18);
```

### Stripe Chat Widget
```css
box-shadow: rgba(0,0,0,0.05) 0px 12px 15px,
            rgba(0,0,0,0.05) 0px 0px 0px 1px,
            rgba(0,0,0,0.08) 0px 5px 9px;
```

---

## 6. Card Styles

### Linear Feature Card
```css
background: var(--color-bg-primary);
border-radius: 16px;
padding: 32px;
box-shadow: var(--shadow-medium);
transition: box-shadow 0.25s, transform 0.25s;

&:hover {
  box-shadow: var(--shadow-large);
  transform: scale(1.02);
}
&:active {
  transform: scale(0.98);
}
```

### Stripe Accented Card
```css
--accentedCardAccentColor: linear-gradient(90deg, #FF5996 0%, #635BFF 50%, #02BCF5 100%);
border-radius: 12px;
```

### Vercel Grid Card
```css
/* Rounded corners with gradient overlay */
rounded-[inherit]
bg-[radial-gradient(circle_at_top_left,var(--ds-background-200)_0%,transparent_30%),
   radial-gradient(circle_at_top_right,var(--ds-background-200)_0%,transparent_30%),
   radial-gradient(circle_at_bottom_right,var(--ds-background-200)_0%,transparent_30%)]
```

---

## 7. Button Styles

### Primary Button (Linear)
```css
display: inline-flex;
align-items: center;
justify-content: center;
padding: 10px 16px;
border-radius: 6px;           /* or 8px */
font-size: 14px;
font-weight: 510;             /* medium */
line-height: 1;
background: var(--color-brand-bg);  /* #5e6ad2 */
color: var(--color-brand-text);     /* #fff */
transition: background-color 0.15s;

&:hover {
  background: var(--color-accent-hover);
}
```

### Stripe CTA Button
```css
padding: 16px 24px;          /* or 3px 0 6px for compact */
border-radius: 16.5px;       /* pill shape */
font: var(--fontWeightSemibold) 15px/1.6 var(--fontFamily);
background-color: var(--buttonColor);  /* #635bff */
color: var(--knockoutColor);           /* #fff */
transition: background-color 0.2s ease-in-out;

&:hover {
  background-color: var(--buttonHoverColor);
}
```

### Linear Invert Button
```css
background: var(--color-button-invert-bg);  /* #e5e5e6 */
color: var(--color-button-invert-bg-hover); /* #fff on hover */
```

### Vercel Deploy Button
```css
/* shape="rounded", size="large" */
border-radius: 12px;
padding: 14px 28px;
font-size: 16px;
font-weight: 600;
```

---

## 8. Hero Treatment

### Stripe Hero Pattern
```css
/* Gradient mesh background */
background: linear-gradient(180deg, var(--gradientColorZero) 0%, var(--gradientColorOne) 100%);
/* Content centered */
text-align: center;
/* Overlay for readability */
background: linear-gradient(transparent, rgba(236,239,241,0.8));
backdrop-filter: blur(5px);
```

### Linear Hero (dark)
```css
background: #08090a;
color: #f7f8f8;
/* Title: title-8 (64px), weight-medium, line-height 1.06, letter-spacing -.022em */
/* Subtitle: text-large (18px), color tertiary (#8a8f98), line-height 1.6 */
```

### Vercel Hero
```css
/* Black background with glow effect */
bg-gray-1000
blur-xl @md:blur-[120px]
dark:opacity-15 opacity-0
/* Content layout: 12-column grid */
grid grid-cols-12
```

### Notion Hero
```css
/* Light hero with product showcase */
text-heading-32 @sm:text-heading-48 @lg:text-heading-56
tracking-tighter
text-balance
```

---

## 9. Section Alternating Patterns

### Stripe's Theme System
```css
.theme--White  { background: #ffffff; }
.theme--Light  { background: #f6f9fc; }
.theme--Dark   { background: #0a2540; }
.theme--SemiDark { background: #0d2e4f; }
```

### Linear's Pattern
```css
/* Alternates between levels */
--color-bg-level-0: #fff;     /* Card bg */
--color-bg-level-1: #f8f8f8;  /* Section bg */
--color-bg-level-2: #f4f4f4;  /* Alternate section */
--color-bg-level-3: #f0f0f0;  /* Deepest level */
```

### Vercel's Pattern
```css
/* Light/Dark theme toggle */
.light-theme { background: #FAFAFA; }
.dark-theme { background: #000; }
```

---

## 10. Grid Patterns

### Linear's Grid
```css
/* Feature grid: 2 columns, 400px min rows */
grid-template-columns: repeat(2, minmax(0, 1fr));
grid-auto-rows: minmax(400px, auto);
gap: 24px;

/* Mobile: single column */
@media (max-width: 768px) {
  grid-template-columns: 1fr;
  grid-auto-rows: minmax(360px, auto);
}
```

### Vercel's 12-Column Grid
```css
grid grid-cols-12 gap-x-0 @xl:gap-x-5

/* Card layout */
col-start-1 col-span-12 row-start-1
@lg:col-start-5 @lg:col-span-8 @lg:row-start-1

/* Text column */
col-start-1 col-span-12 row-start-2
@lg:col-start-10 @lg:col-span-3 @lg:row-start-1
```

### Stripe's Layout
```css
max-width: calc(1080px + var(--columnPaddingNormal) * 2);
margin: 0 auto;
```

---

## 11. Animation / Motion

### Linear Transitions
```css
/* Standard transitions */
--speed-quickTransition: 0.1s;
--speed-regularTransition: 0.25s;

/* Easing functions */
--ease-out-quad: cubic-bezier(0.25, 0.46, 0.45, 0.94);
--ease-out-cubic: cubic-bezier(0.215, 0.61, 0.355, 1);
--ease-out-expo: cubic-bezier(0.19, 1, 0.22, 1);

/* Applied to: */
transition-property: transform, opacity, color;
transition-duration: 0.25s;
transition-timing-function: cubic-bezier(0.25, 0.46, 0.45, 0.94);
```

### Stripe Hover Arrow
```css
--arrowHoverTransition: 150ms cubic-bezier(0.215, 0.61, 0.355, 1);
--arrowHoverOffset: translateX(3px);

:hover .HoverArrow__tipPath {
  transform: var(--arrowHoverOffset);
}
:hover .HoverArrow__linePath {
  opacity: 1;
}
```

### Linear Marquee
```css
@keyframes marqueeMove {
  0% { transform: translateX(var(--marquee-start, 0%)); }
  100% { transform: translateX(calc(-100% + var(--marquee-start, 0%))); }
}
animation: marqueeMove var(--duration, 30s) linear infinite;
```

### Reduced Motion
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 12. Premium Design Devices

### 1. Gradient Mesh Background (Stripe)
```css
/* Multi-color gradient blobs */
background: 
  radial-gradient(circle at 20% 30%, #a960ee 0%, transparent 50%),
  radial-gradient(circle at 80% 70%, #ff333d 0%, transparent 50%),
  radial-gradient(circle at 50% 50%, #0048e5 0%, transparent 50%);
```

### 2. Radial Fade Mask (Linear)
```css
/* Fade content at edges */
-webkit-mask-image: radial-gradient(50% 50%, #000 20%, transparent 100%);
mask-image: radial-gradient(50% 50%, #000 20%, transparent 100%);
```

### 3. Glass / Frosted Header (Linear)
```css
backdrop-filter: saturate(1.8) blur(20px);
background: var(--header-bg);  /* rgba with alpha */
border-bottom: 1px solid var(--header-border);
```

### 4. Backdrop Blur Overlay (Stripe)
```css
background: linear-gradient(transparent, rgba(236,239,241,0.8));
backdrop-filter: blur(5px);
```

### 5. Animated Gradient Text (Stripe)
```css
background: linear-gradient(90deg, #e18638, #e17a38);
background-clip: text;
-webkit-background-clip: text;
-webkit-text-fill-color: transparent;
background-size: 300% 100%;
```

### 6. Radial Glow Under Image (Vercel)
```css
bg-gray-1000
blur-xl @md:blur-[120px]
dark:opacity-15 opacity-0
rounded-full
/* Positioned absolutely behind image */
```

### 7. Content Fade at Bottom (Vercel)
```css
pointer-events-none absolute inset-x-0 bottom-0 z-2 h-40
bg-linear-to-t from-background-200 to-transparent
```

### 8. Card Hover Scale with Shadow (Linear)
```css
transition: box-shadow 0.25s, transform 0.25s;
&:hover {
  box-shadow: var(--shadow-large);
  transform: scale(1.02);
}
```

### 9. Dot Grid / Guide Lines (Stripe)
```css
background: linear-gradient(180deg, var(--guideDashedColor), var(--guideDashedColor) 50%, transparent 0, transparent);
background-size: 1px 8px;
```

### 10. Feature Marquee (Linear)
```css
overflow: hidden;
display: flex;
justify-content: center;
.marquee-content {
  animation: marqueeMove 30s linear infinite;
}
```

---

## 13. Tailwind CSS Utility Classes Cheat Sheet

Based on the extracted values, here are the Tailwind config extensions:

```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        'stripe-slate': '#0a2540',
        'stripe-purple': '#635bff',
        'stripe-teal': '#11efe3',
        'linear-indigo': '#5e6ad2',
        'linear-accent': '#7070ff',
        'bg-primary': { light: '#ffffff', dark: '#08090a' },
        'bg-secondary': { light: '#f9f8f9', dark: '#1c1c1f' },
        'text-primary': { light: '#282a30', dark: '#f7f8f8' },
        'text-secondary': { light: '#3c4149', dark: '#d0d6e0' },
        'text-tertiary': { light: '#6f6e77', dark: '#8a8f98' },
        'border-primary': { light: '#e9e8ea', dark: '#23252a' },
      },
      borderRadius: {
        'card': '16px',
        'button': '6px',
        'pill': '9999px',
      },
      boxShadow: {
        'card': '0px 3px 12px rgba(0,0,0,0.09)',
        'card-hover': '0px 7px 24px rgba(0,0,0,0.06)',
        'glow': '0 0 60px rgba(50,50,93,0.18)',
        'glass': '0 4px 24px rgba(0,0,0,0.12)',
      },
      fontFamily: {
        sans: ['"Inter Variable"', '"SF Pro Display"', '-apple-system', 'sans-serif'],
        mono: ['"Berkeley Mono"', '"SF Mono"', 'monospace'],
      },
      fontSize: {
        'hero': ['4rem', { lineHeight: '1.06', letterSpacing: '-0.022em' }],
        'h1': ['3.5rem', { lineHeight: '1.1', letterSpacing: '-0.022em' }],
        'h2': ['2rem', { lineHeight: '1.125', letterSpacing: '-0.022em' }],
        'h3': ['1.5rem', { lineHeight: '1.33', letterSpacing: '-0.012em' }],
        'body': ['0.9375rem', { lineHeight: '1.6', letterSpacing: '-0.011em' }],
        'caption': ['0.8125rem', { lineHeight: '1.5', letterSpacing: '-0.01em' }],
      },
      transitionTimingFunction: {
        'premium': 'cubic-bezier(0.25, 0.46, 0.45, 0.94)',
        'bounce-out': 'cubic-bezier(0.215, 0.61, 0.355, 1)',
      },
      transitionDuration: {
        '250': '250ms',
        '300': '300ms',
      },
    },
  },
};
```

---

## 14. Quick-Start Component Patterns

### Premium Card
```html
<div class="bg-white dark:bg-[#08090a] rounded-[16px] p-8 
            shadow-[0px_3px_12px_rgba(0,0,0,0.09)]
            transition-all duration-300 ease-[cubic-bezier(0.25,0.46,0.45,0.94)]
            hover:shadow-[0px_7px_24px_rgba(0,0,0,0.06)] hover:scale-[1.02]
            active:scale-[0.98]">
  <!-- content -->
</div>
```

### Premium Hero
```html
<section class="relative min-h-screen flex items-center justify-center
                bg-[#08090a] text-white overflow-hidden">
  <div class="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,#5e6ad2_0%,transparent_60%)] opacity-20"></div>
  <div class="relative z-10 max-w-[1024px] mx-auto px-6 text-center">
    <h1 class="text-[4rem] leading-[1.06] tracking-[-0.022em] font-medium mb-6">
      Your Headline
    </h1>
    <p class="text-[1.0625rem] leading-1.6 text-[#8a8f98] max-w-[600px] mx-auto">
      Your subtitle here.
    </p>
  </div>
</section>
```

### Glass Header
```html
<header class="fixed top-0 w-full z-50 
               backdrop-blur-[20px] saturate-[1.8]
               bg-white/80 dark:bg-[#08090a]/80
               border-b border-black/5 dark:border-white/5">
  <nav class="max-w-[1344px] mx-auto px-6 h-[72px] flex items-center justify-between">
    <!-- nav content -->
  </nav>
</header>
```

### Feature Grid
```html
<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
  <div class="min-h-[400px] bg-white dark:bg-[#08090a] rounded-[16px] p-8 shadow-card">
    <!-- feature 1 -->
  </div>
  <div class="min-h-[400px] bg-white dark:bg-[#08090a] rounded-[16px] p-8 shadow-card">
    <!-- feature 2 -->
  </div>
</div>
```

### CTA Section
```html
<section class="py-20 text-center">
  <h2 class="text-[3.5rem] leading-[1.1] tracking-[-0.022em] font-medium mb-8">
    Built by you, or your agents
  </h2>
  <div class="flex gap-4 justify-center">
    <button class="px-8 py-4 rounded-[12px] bg-[#5e6ad2] text-white text-lg font-semibold
                   transition-all duration-200 hover:bg-[#4a54b8]">
      Deploy now
    </button>
  </div>
</section>
```

---

*Generated from analysis of stripe.com/payments, linear.app/features, vercel.com, notion.so/product*
