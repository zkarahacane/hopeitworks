# Sprint Planning - Sprint Status Generator

<critical>The workflow execution engine is governed by: {project-root}/_bmad/core/tasks/workflow.xml</critical>
<critical>You MUST have already loaded and processed: {project-root}/_bmad/bmm/workflows/4-implementation/sprint-planning/workflow.yaml</critical>

## 📚 Document Discovery - Full Epic Loading

**Strategy**: Sprint planning needs ALL epics and stories to build complete status tracking.

**Epic Discovery Process:**

1. **Search for whole document first** - Look for `epics.md`, `bmm-epics.md`, or any `*epic*.md` file
2. **Check for sharded version** - If whole document not found, look for `epics/index.md`
3. **If sharded version found**:
   - Read `index.md` to understand the document structure
   - Read ALL epic section files listed in the index (e.g., `epic-1.md`, `epic-2.md`, etc.)
   - Process all epics and their stories from the combined content
   - This ensures complete sprint status coverage
4. **Priority**: If both whole and sharded versions exist, use the whole document

**Fuzzy matching**: Be flexible with document names - users may use variations like `epics.md`, `bmm-epics.md`, `user-stories.md`, etc.

<workflow>

<step n="1" goal="Parse epic files and extract all work items">
<action>Communicate in {communication_language} with {user_name}</action>
<action>Look for all files matching `{epics_pattern}` in {epics_location}</action>
<action>Could be a single `epics.md` file or multiple `epic-1.md`, `epic-2.md` files</action>

<action>For each epic file found, extract:</action>

- Epic numbers from headers like `## Epic 1:` or `## Epic 2:`
- Story IDs and titles from patterns like `### Story 1.1: User Authentication`
- Convert story format from `Epic.Story: Title` to kebab-case key: `epic-story-title`

**Story ID Conversion Rules:**

- Original: `### Story 1.1: User Authentication`
- Replace period with dash: `1-1`
- Convert title to kebab-case: `user-authentication`
- Final key: `1-1-user-authentication`

<action>Build complete inventory of all epics and stories from all epic files</action>
</step>

  <step n="0.5" goal="Discover and load project documents">
    <invoke-protocol name="discover_inputs" />
    <note>After discovery, these content variables are available: {epics_content} (all epics loaded - uses FULL_LOAD strategy)</note>
  </step>

<step n="2" goal="Build sprint status structure">
<action>For each epic found, create entries in this order:</action>

1. **Epic entry** - Key: `epic-{num}`, Default status: `backlog`
2. **Story entries** - Key: `{epic}-{story}-{title}`, Default status: `backlog`
3. **Retrospective entry** - Key: `epic-{num}-retrospective`, Default status: `optional`

**Example structure:**

```yaml
development_status:
  epic-1: backlog
  1-1-user-authentication: backlog
  1-2-account-management: backlog
  epic-1-retrospective: optional
```

</step>

<step n="3" goal="Apply intelligent status detection">
<action>For each story, detect current status by checking files:</action>

**Story file detection:**

- Check: `{story_location_absolute}/{story-key}.md` (e.g., `stories/1-1-user-authentication.md`)
- If exists → upgrade status to at least `ready-for-dev`

**Preservation rule:**

- If existing `{status_file}` exists and has more advanced status, preserve it
- Never downgrade status (e.g., don't change `done` to `ready-for-dev`)

**Status Flow Reference:**

- Epic: `backlog` → `in-progress` → `done`
- Story: `backlog` → `ready-for-dev` → `in-progress` → `review` → `done`
- Retrospective: `optional` ↔ `done`
  </step>

<step n="4" goal="Generate sprint status file">
<action>Create or update {status_file} with:</action>

**File Structure:**

```yaml
# generated: {date}
# project: {project_name}
# project_key: {project_key}
# tracking_system: {tracking_system}
# story_location: {story_location}

# STATUS DEFINITIONS:
# ==================
# Epic Status:
#   - backlog: Epic not yet started
#   - in-progress: Epic actively being worked on
#   - done: All stories in epic completed
#
# Epic Status Transitions:
#   - backlog → in-progress: Automatically when first story is created (via create-story)
#   - in-progress → done: Manually when all stories reach 'done' status
#
# Story Status:
#   - backlog: Story only exists in epic file
#   - ready-for-dev: Story file created in stories folder
#   - in-progress: Developer actively working on implementation
#   - review: Ready for code review (via Dev's code-review workflow)
#   - done: Story completed
#
# Retrospective Status:
#   - optional: Can be completed but not required
#   - done: Retrospective has been completed
#
# WORKFLOW NOTES:
# ===============
# - Epic transitions to 'in-progress' automatically when first story is created
# - Stories can be worked in parallel if team capacity allows
# - SM typically creates next story after previous one is 'done' to incorporate learnings
# - Dev moves story to 'review', then runs code-review (fresh context, different LLM recommended)

generated: { date }
project: { project_name }
project_key: { project_key }
tracking_system: { tracking_system }
story_location: { story_location }

development_status:
  # All epics, stories, and retrospectives in order
```

<action>Write the complete sprint status YAML to {status_file}</action>
<action>CRITICAL: Metadata appears TWICE - once as comments (#) for documentation, once as YAML key:value fields for parsing</action>
<action>Ensure all items are ordered: epic, its stories, its retrospective, next epic...</action>
</step>

<step n="5" goal="Validate and report">
<action>Perform validation checks:</action>

- [ ] Every epic in epic files appears in {status_file}
- [ ] Every story in epic files appears in {status_file}
- [ ] Every epic has a corresponding retrospective entry
- [ ] No items in {status_file} that don't exist in epic files
- [ ] All status values are legal (match state machine definitions)
- [ ] File is valid YAML syntax

<action>Count totals:</action>

- Total epics: {{epic_count}}
- Total stories: {{story_count}}
- Epics in-progress: {{in_progress_count}}
- Stories done: {{done_count}}

<action>Display intermediate summary to {user_name} in {communication_language} before parallel analysis</action>
</step>

<step n="6" goal="Analyze story dependencies and build parallel execution waves">
<critical>This step identifies which stories can run simultaneously in separate containers</critical>
<critical>Dependency analysis uses: explicit depends_on, story ordering within epics, and domain tags [BACK]/[FRONT]/[SHARED]</critical>

**Dependency Sources (in priority order):**

1. **Explicit depends_on** — If a created story file exists in {story_location}, read its frontmatter for `depends_on: [story-keys]`
2. **Sequential within same domain** — Within an epic, stories of the same domain ([BACK] or [FRONT]) are sequential by default: 1.1 → 1.2 → 1.3
3. **Cross-domain independence** — [BACK] and [FRONT] stories in the same epic are independent unless explicit depends_on says otherwise
4. **[SHARED] blocks both** — A [SHARED] story is a dependency barrier: both domains must wait for it, and it waits for both domains' prior stories
5. **Cross-epic dependencies** — A story in Epic N that depends on a story in Epic M creates an inter-epic dependency

**Algorithm:**

<action>For each story in development_status:
  1. Extract domain tag from epic file: [BACK], [FRONT], [SHARED], or [FULL]
  2. If story file exists in {story_location}, read frontmatter for depends_on and target_files
  3. Build dependency list:
     - Previous same-domain story in same epic (implicit sequential)
     - All stories listed in depends_on (explicit)
     - If [SHARED]: previous story of EVERY domain in same epic
  4. If no dependencies resolved → wave 1 candidate
</action>

<action>Build waves using topological sort:
  - **Wave 1**: Stories with zero unresolved dependencies (typically first [BACK] + first [FRONT] of each active epic)
  - **Wave 2**: Stories whose ALL dependencies are in Wave 1
  - **Wave N**: Stories whose ALL dependencies are in Wave 1..N-1
  - Flag circular dependencies as errors
</action>

<action>For each wave, record:
  - Story keys in the wave
  - Domain breakdown (how many [BACK], [FRONT], [SHARED])
  - Estimated container count needed
  - Which stories from previous waves must complete first
</action>

**File conflict detection (optional, for created stories only):**

<action>If story files exist with target_files in frontmatter:
  - Compare target_files across stories in the same wave
  - If overlap detected: flag as potential merge conflict risk
  - Add warning to wave notes
</action>

**Output: Append parallel_waves section to {status_file}:**

```yaml
# PARALLEL EXECUTION WAVES
# ========================
# Stories in the same wave can run simultaneously in separate containers.
# Each wave must complete before the next wave starts.
# Domain tags: [B]=Backend, [F]=Frontend, [S]=Shared
#
# Launch command per story:
#   ./scripts/bmad-dev.sh -p "/bmad-bmm-dev-story" --model sonnet
# Launch all stories in a wave (parallel containers):
#   for story in <wave-stories>; do
#     ./scripts/bmad-dev.sh -p "/bmad-bmm-create-story $story && /bmad-bmm-dev-story" &
#   done

parallel_waves:
  wave-1:
    stories:
      - key: 1-1-go-scaffolding
        domain: back
      - key: 1-7-vue-scaffolding
        domain: front
    container_count: 2
    depends_on: []
    notes: "No dependencies - safe to launch all simultaneously"
  wave-2:
    stories:
      - key: 1-2-openapi-codegen
        domain: back
      - key: 1-8-app-shell-layout
        domain: front
    container_count: 2
    depends_on: [wave-1]
    notes: "Back stories depend on 1-1, Front stories depend on 1-7"
```

<action>Write the parallel_waves section to {status_file}, appended after development_status</action>
<action>Preserve ALL existing content in {status_file} — only append/update parallel_waves</action>

**Conflict risk warnings:**
<action>If any wave has stories with overlapping target_files, add a conflict_risks entry:
```yaml
    conflict_risks:
      - stories: [1-2-openapi-codegen, 1-13-claude-md-files]
        shared_files: [api/openapi.yaml]
        severity: high
        recommendation: "Run sequentially or ensure stories touch different sections"
```
</action>
</step>

<step n="7" goal="Validate and report">
<action>Perform validation checks:</action>

- [ ] Every epic in epic files appears in {status_file}
- [ ] Every story in epic files appears in {status_file}
- [ ] Every epic has a corresponding retrospective entry
- [ ] No items in {status_file} that don't exist in epic files
- [ ] All status values are legal (match state machine definitions)
- [ ] File is valid YAML syntax
- [ ] parallel_waves covers ALL non-done stories
- [ ] No circular dependencies detected in wave analysis
- [ ] Every story appears in exactly one wave

<action>Count totals:</action>

- Total epics: {{epic_count}}
- Total stories: {{story_count}}
- Epics in-progress: {{in_progress_count}}
- Stories done: {{done_count}}
- Parallel waves: {{wave_count}}
- Max concurrent containers needed: {{max_containers}}

<action>Display completion summary to {user_name} in {communication_language}:</action>

**Sprint Status Generated Successfully**

- **File Location:** {status_file}
- **Total Epics:** {{epic_count}}
- **Total Stories:** {{story_count}}
- **Epics In Progress:** {{epics_in_progress_count}}
- **Stories Completed:** {{done_count}}

**Parallel Execution Plan:**

- **Waves:** {{wave_count}}
- **Max containers per wave:** {{max_containers}}
- **Conflict risks:** {{conflict_risk_count}}

| Wave | Stories | Containers | Depends On |
|------|---------|------------|------------|
{{#each wave in parallel_waves}}
| {{wave.name}} | {{wave.story_count}} | {{wave.container_count}} | {{wave.depends_on}} |
{{/each}}

**Next Steps:**

1. Review the generated {status_file} and parallel_waves
2. For each wave, launch stories in parallel:
   ```bash
   ./scripts/bmad-dev.sh -p "/bmad-bmm-dev-story"
   ```
3. Wait for all stories in a wave to complete (merge) before starting next wave
4. Use `/bmad-bmm-sprint-status` to monitor progress

</step>

</workflow>

## Additional Documentation

### Status State Machine

**Epic Status Flow:**

```
backlog → in-progress → done
```

- **backlog**: Epic not yet started
- **in-progress**: Epic actively being worked on (stories being created/implemented)
- **done**: All stories in epic completed

**Story Status Flow:**

```
backlog → ready-for-dev → in-progress → review → done
```

- **backlog**: Story only exists in epic file
- **ready-for-dev**: Story file created (e.g., `stories/1-3-plant-naming.md`)
- **in-progress**: Developer actively working
- **review**: Ready for code review (via Dev's code-review workflow)
- **done**: Completed

**Retrospective Status:**

```
optional ↔ done
```

- **optional**: Ready to be conducted but not required
- **done**: Finished

### Guidelines

1. **Epic Activation**: Mark epic as `in-progress` when starting work on its first story
2. **Sequential Default**: Stories are typically worked in order, but parallel work is supported
3. **Parallel Work Supported**: Multiple stories can be `in-progress` if team capacity allows
4. **Review Before Done**: Stories should pass through `review` before `done`
5. **Learning Transfer**: SM typically creates next story after previous one is `done` to incorporate learnings
