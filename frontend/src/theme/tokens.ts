/**
 * Design token hierarchy:
 *
 * 1. Primitive tokens - Raw values (colors, spacing, font sizes, fonts)
 * 2. Semantic tokens  - Contextual meaning (surface, status families)
 * 3. Component tokens - Component-specific overrides (in theme/index.ts)
 *
 * PrimeVue Aura provides baseline primitive/semantic layers; HopeTheme
 * (theme/index.ts) overrides them with the values below. Status family colors
 * are exposed as CSS custom properties in assets/main.css and consumed via the
 * unified `statusToken` system — never hard-code these hexes in components.
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

/**
 * Type families.
 * - Space Grotesk: headings, KPIs, UI chrome (the human voice).
 * - JetBrains Mono: IDs, durations, costs, branch names, container ids, logs
 *   (the machine voice). Exposed as the `--font-mono` token + `.font-mono`
 *   utility so any component can speak "machine".
 */
export const fontTokens = {
  sans: "'Space Grotesk', ui-sans-serif, system-ui, sans-serif",
  mono: "'JetBrains Mono', ui-monospace, 'SFMono-Regular', 'Menlo', monospace",
} as const

/**
 * Surface ramp — dark base → raised → overlay → border, then a lighter pair.
 * Darkest-first per the design palette. PrimeVue maps surface.950..0.
 */
const darkSurfaces = {
  base: '#0B0F0D', // app background
  raised: '#131A16', // cards / panels
  overlay: '#1A221D', // popovers / hovered rows
  border: '#28332C', // borders / dividers
} as const

const lightSurfaces = {
  base: '#F6F8F6', // app background
  raised: '#FFFFFF', // cards / panels
  overlay: '#EFF2EF', // popovers / hovered rows
  border: '#D7DCD8', // borders / dividers
} as const

export const surfaceTokens = { dark: darkSurfaces, light: lightSurfaces } as const

/**
 * Status family colors — the 5 product families.
 *
 * - running: phosphor green (vivid, animated)
 * - done:    calm/solid green (same hue family, no animation)
 * - gate:    amber (breathing while awaiting a human)
 * - failed:  red
 * - queued:  gray (neutral)
 *
 * `accent` (blue) is RESERVED for non-status informational accents only and is
 * never a run/step/story status — it lives here only so the theme primary can
 * reference one consistent blue.
 */
export const statusColorTokens = {
  dark: {
    running: { color: '#39FF8B', surface: 'rgba(57, 255, 139, 0.14)' },
    done: { color: '#37B66B', surface: 'rgba(55, 182, 107, 0.14)' },
    gate: { color: '#F5B342', surface: 'rgba(245, 179, 66, 0.16)' },
    failed: { color: '#F56565', surface: 'rgba(245, 101, 101, 0.16)' },
    queued: { color: '#8A958E', surface: 'rgba(138, 149, 142, 0.16)' },
    accent: '#5B9DFF',
  },
  light: {
    running: { color: '#16A34A', surface: 'rgba(22, 163, 74, 0.12)' },
    done: { color: '#15803D', surface: 'rgba(21, 128, 61, 0.12)' },
    gate: { color: '#B7791F', surface: 'rgba(183, 121, 31, 0.14)' },
    failed: { color: '#DC2626', surface: 'rgba(220, 38, 38, 0.12)' },
    queued: { color: '#6B7280', surface: 'rgba(107, 114, 128, 0.14)' },
    accent: '#2563EB',
  },
} as const

/**
 * Full PrimeVue surface ramps (950..0) derived from the 4-step palette.
 * Intermediate stops are interpolated to keep PrimeVue's components coherent
 * while pinning the 4 design anchors.
 */
export const semanticTokens = {
  colorScheme: {
    light: {
      surface: {
        0: lightSurfaces.raised, // #FFFFFF
        50: lightSurfaces.base, // #F6F8F6
        100: lightSurfaces.overlay, // #EFF2EF
        200: lightSurfaces.border, // #D7DCD8
        300: '#C2C9C4',
        400: '#9BA39D',
        500: '#717A74',
        600: '#535B55',
        700: '#3B423D',
        800: '#262B28',
        900: '#161A17',
        950: '#0B0F0D',
      },
    },
    dark: {
      surface: {
        0: '#FFFFFF',
        50: '#E8EDE9',
        100: '#C2CAC5',
        200: '#9AA39D',
        300: '#6E7A73',
        400: '#4C5851',
        500: '#3A453E',
        600: darkSurfaces.border, // #28332C
        700: darkSurfaces.overlay, // #1A221D
        800: darkSurfaces.raised, // #131A16
        900: darkSurfaces.base, // #0B0F0D
        950: '#070A09',
      },
    },
  },
} as const
