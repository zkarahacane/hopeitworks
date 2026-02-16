/**
 * Design token hierarchy:
 *
 * 1. Primitive tokens - Raw values (colors, spacing, font sizes)
 * 2. Semantic tokens  - Contextual meaning (primary, surface, text)
 * 3. Component tokens - Component-specific overrides
 *
 * PrimeVue Aura preset provides the primitive and semantic layers.
 * Component tokens can be overridden via definePreset() in theme/index.ts.
 */

export const primitiveTokens = {
  borderRadius: {
    sm: '0.25rem',
    md: '0.375rem',
    lg: '0.5rem',
    xl: '0.75rem',
  },
  spacing: {
    xs: '0.25rem',
    sm: '0.5rem',
    md: '1rem',
    lg: '1.5rem',
    xl: '2rem',
  },
} as const

export const semanticTokens = {
  colorScheme: {
    light: {
      surface: {
        0: '#ffffff',
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
    dark: {
      surface: {
        0: '#ffffff',
        50: '#fafafa',
        100: '#f4f4f5',
        200: '#e4e4e7',
        300: '#d4d4d8',
        400: '#a1a1aa',
        500: '#71717a',
        600: '#52525b',
        700: '#3f3f46',
        800: '#27272a',
        900: '#18181b',
        950: '#09090b',
      },
    },
  },
} as const
