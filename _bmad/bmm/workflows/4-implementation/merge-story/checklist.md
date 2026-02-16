---
title: 'Merge Story - Validation Checklist'
validation-target: 'Story branch merge into target branch'
validation-criticality: 'HIGH'
---

# Merge Story Validation Checklist

## Pre-Merge Gates

- [ ] Story status is "done" or "review" (code review passed)
- [ ] PR exists targeting the correct base branch (wave-X or main)
- [ ] All CI checks are green
- [ ] Branch is rebased onto latest target (no conflicts)
- [ ] No uncommitted changes on the branch

## Merge Execution

- [ ] PR squash-merged successfully
- [ ] Feature branch deleted after merge
- [ ] Target branch pulled with merged changes

## Post-Merge

- [ ] sprint-status.yaml updated: story → done
- [ ] Story file Status updated to "done"
- [ ] Change Log entry added with merge details
