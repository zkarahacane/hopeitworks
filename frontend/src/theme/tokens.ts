/**
 * Design Tokens - 3-level hierarchy
 *
 * Primitive tokens: Raw values (colors, sizes, fonts)
 * Semantic tokens: Purpose-driven aliases (primary, surface, text)
 * Component tokens: Component-specific overrides
 *
 * PrimeVue Aura preset provides the full token system.
 * This file documents the token hierarchy for reference
 * and provides any project-specific token extensions.
 */

// Primitive tokens are defined in the Aura preset (blue, green, red, etc.)
// Semantic tokens map primitives to purposes (primary, secondary, success, etc.)
// Component tokens are auto-derived from semantic tokens by PrimeVue

export const tokenReference = {
  primitive: {
    description: 'Raw values from Aura preset: blue, green, red, yellow, etc.',
  },
  semantic: {
    description: 'Purpose mappings configured in HopeTheme preset',
    primary: 'Maps to blue palette',
  },
  component: {
    description: 'Auto-derived from semantic tokens by PrimeVue',
  },
} as const
