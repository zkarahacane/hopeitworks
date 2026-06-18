import { definePreset } from '@primevue/themes'
import Aura from '@primevue/themes/aura'
import { semanticTokens, fontTokens, statusColorTokens } from './tokens'

/**
 * HopeTheme — the hopeitworks PrimeVue preset.
 *
 * Layers on top of Aura:
 *  - Custom surface ramps (dark phosphor-on-near-black, light off-white) from
 *    the design palette in tokens.ts.
 *  - Space Grotesk as the UI font; JetBrains Mono exposed as `--font-mono`.
 *  - The 5 status families exposed as CSS custom properties per color scheme
 *    (`--status-<family>-color` / `--status-<family>-surface`) so the unified
 *    `statusToken` system can resolve them at runtime, with dark/light parity.
 *  - `primary` anchored to the informational blue accent ONLY (blue is never a
 *    status — see statusToken.ts).
 *
 * The `extend` block defines arbitrary tokens; PrimeVue emits them as CSS vars
 * scoped to `:root` (light) and the `.dark` selector (dark), so light/dark
 * parity holds automatically via the existing `darkModeSelector: '.dark'`.
 */

function statusVars(scheme: 'light' | 'dark') {
  const s = statusColorTokens[scheme]
  return {
    runningColor: s.running.color,
    runningSurface: s.running.surface,
    doneColor: s.done.color,
    doneSurface: s.done.surface,
    gateColor: s.gate.color,
    gateSurface: s.gate.surface,
    failedColor: s.failed.color,
    failedSurface: s.failed.surface,
    queuedColor: s.queued.color,
    queuedSurface: s.queued.surface,
    accent: s.accent,
  }
}

export const HopeTheme = definePreset(Aura, {
  primitive: {
    // Make the informational blue accent available as a primitive ramp.
    accent: {
      50: '#EFF5FF',
      100: '#DBE8FF',
      200: '#BFD6FF',
      300: '#93BAFF',
      400: '#5B9DFF',
      500: '#3B82F6',
      600: '#2563EB',
      700: '#1D4ED8',
      800: '#1E40AF',
      900: '#1E3A8A',
      950: '#172554',
    },
  },
  semantic: {
    // Blue accent = informational only. Status meaning comes from statusToken.
    primary: {
      50: '{accent.50}',
      100: '{accent.100}',
      200: '{accent.200}',
      300: '{accent.300}',
      400: '{accent.400}',
      500: '{accent.500}',
      600: '{accent.600}',
      700: '{accent.700}',
      800: '{accent.800}',
      900: '{accent.900}',
      950: '{accent.950}',
    },
    // Reusable token surface for the design's status families + fonts.
    status: statusVars('light'),
    font: {
      family: fontTokens.sans,
      mono: fontTokens.mono,
    },
    colorScheme: {
      light: {
        ...semanticTokens.colorScheme.light,
        status: statusVars('light'),
      },
      dark: {
        ...semanticTokens.colorScheme.dark,
        status: statusVars('dark'),
      },
    },
  },
})
