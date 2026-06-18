/**
 * Phase grouping for the step timeline.
 *
 * A pipeline's steps are grouped into four product phases so the timeline reads
 * as a story (Setup → Dev → Review → Delivery) rather than a flat list. The
 * mapping is heuristic on the step's action_type / name; hero screens can also
 * pass an explicit phase per step to override.
 */

export type Phase = 'setup' | 'dev' | 'review' | 'delivery'

export interface PhaseDef {
  key: Phase
  label: string
  icon: string
}

/** Ordered phase definitions (the canonical timeline order). */
export const PHASES: readonly PhaseDef[] = [
  { key: 'setup', label: 'Setup', icon: 'pi pi-cog' },
  { key: 'dev', label: 'Dev', icon: 'pi pi-code' },
  { key: 'review', label: 'Review', icon: 'pi pi-eye' },
  { key: 'delivery', label: 'Delivery', icon: 'pi pi-send' },
] as const

/** Keyword → phase heuristics, checked in order against action_type then name. */
const KEYWORDS: Array<{ phase: Phase; match: RegExp }> = [
  { phase: 'setup', match: /\b(clone|checkout|setup|init|prepare|provision|container|workspace)\b/i },
  { phase: 'review', match: /\b(reviews?|hitl|approvals?|approve|gate|lint|tests?|verify|qa|checks?)\b/i },
  { phase: 'delivery', match: /\b(pr|pull_request|merge|deploy|push|commit|publish|deliver|release)\b/i },
  { phase: 'dev', match: /\b(dev|code|implement|agent|build|generate|edit|write|fix)\b/i },
]

/** Normalize snake_case / kebab-case into space-separated tokens for matching. */
function normalize(s: string): string {
  return s.replace(/[_-]+/g, ' ').toLowerCase()
}

/**
 * Classify a step into a phase from its action type / name.
 * Defaults to `dev` (the work bucket) when nothing matches.
 */
export function phaseForStep(input: {
  actionType?: string | null
  name?: string | null
}): Phase {
  const haystacks = [input.actionType ?? '', input.name ?? '']
  for (const h of haystacks) {
    if (!h) continue
    const normalized = normalize(h)
    for (const { phase, match } of KEYWORDS) {
      if (match.test(normalized)) return phase
    }
  }
  return 'dev'
}

/** Label for a phase key. */
export function phaseLabel(phase: Phase): string {
  return PHASES.find((p) => p.key === phase)?.label ?? phase
}
