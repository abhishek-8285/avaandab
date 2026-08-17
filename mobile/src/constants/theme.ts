// Avandab Driver Pro - Dense, data-first UI tokens for professional drivers.

export const Colors = {
  // Brand
  primary: '#0f766e',          // Teal-700 - saturated accent for active states
  primaryDark: '#115e59',
  primaryLight: '#ccfbf1',
  primarySubtle: '#99f6e4',

  // Dark chrome (headers, top bars, status banners)
  chrome: '#0f172a',           // Slate 900
  chromeLight: '#1e293b',      // Slate 800
  chromeBorder: '#334155',     // Slate 700

  // Backgrounds & surfaces
  background: '#f1f5f9',       // Slate 100
  surface: '#ffffff',
  surfaceSecondary: '#f8fafc', // Slate 50

  // Text
  textPrimary: '#0f172a',      // Slate 900
  textSecondary: '#475569',    // Slate 600
  textMuted: '#94a3b8',        // Slate 400
  textOnPrimary: '#ffffff',
  textOnChrome: '#e2e8f0',     // Slate 200 (on dark chrome)
  textOnChromeMuted: '#94a3b8',

  // Status - high contrast
  success: '#15803d',
  successBg: '#dcfce7',
  warning: '#b45309',
  warningBg: '#fef3c7',
  danger: '#dc2626',
  dangerBg: '#fee2e2',
  info: '#0369a1',
  infoBg: '#e0f2fe',

  // Borders - sharper visibility
  border: '#cbd5e1',           // Slate 300
  borderLight: '#e2e8f0',      // Slate 200
};

export const Font = {
  mono: 'monospace',
  sans: 'system-ui',
};

export const Radius = {
  none: 0,
  sm: 2,
  md: 4,
  lg: 6,
  xl: 8,
};

export const Spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  xxl: 24,
};

export const Shadows = {
  card: {
    shadowColor: '#0f172a',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.06,
    shadowRadius: 2,
    elevation: 1,
  },
  modal: {
    shadowColor: '#0f172a',
    shadowOffset: { width: 0, height: 8 },
    shadowOpacity: 0.2,
    shadowRadius: 16,
    elevation: 8,
  },
};
