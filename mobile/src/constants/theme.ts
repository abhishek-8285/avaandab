// Avandab Brand & UI Color Palette (Matching Web App)

export const Colors = {
  // Brand Colors
  primary: '#00685f',          // Main Avandab Teal
  primaryDark: '#005049',      // Dark Teal hover/pressed
  primaryLight: '#e6f4f1',     // Soft Teal badge background
  primarySubtle: '#ccece7',    // Light Teal borders

  // Background & Surface Colors
  background: '#f7f9fb',       // Main App Background
  surface: '#ffffff',          // Card & Container Background
  surfaceSecondary: '#f1f5f9', // Sub-card & neutral background

  // Text Colors
  textPrimary: '#0f172a',      // Main Headings (Slate 900)
  textSecondary: '#565e74',    // Body text & labels
  textMuted: '#94a3b8',        // Subtitles & placeholder text
  textOnPrimary: '#ffffff',    // Text over Teal buttons

  // Status & Accent Colors
  success: '#15803d',          // Green status
  successBg: '#dcfce7',        // Green badge background
  warning: '#b45309',          // Amber status
  warningBg: '#fef3c7',        // Amber badge background
  danger: '#dc2626',           // Red status/error
  dangerBg: '#fee2e2',         // Red badge background

  // Border & Divider Colors
  border: '#e2e8f0',           // Card border
  borderLight: '#f1f5f9',      // Divider line
};

export const Shadows = {
  card: {
    shadowColor: '#0f172a',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.05,
    shadowRadius: 8,
    elevation: 2,
  },
  modal: {
    shadowColor: '#0f172a',
    shadowOffset: { width: 0, height: 10 },
    shadowOpacity: 0.15,
    shadowRadius: 20,
    elevation: 10,
  },
};
