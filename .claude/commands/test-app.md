Test the hopeitworks application end-to-end using the automated test infrastructure.

## Workflow

Execute these phases sequentially, delegating each to a Task agent:

### Phase 1 — Setup (Task Haiku)

```
Run: ./scripts/e2e-stack.sh status
If not healthy: Run ./scripts/e2e-stack.sh up
Then: ./scripts/e2e-stack.sh wait
Return: OK/KO + any errors
```

### Phase 2 — Automated Tests (Task Sonnet)

```
Run: ./scripts/e2e-smoke.sh
Parse: frontend/e2e/real-results/results.json
Read: frontend/e2e/real-results/backend-logs.txt (tail last 200 lines if large)
Return: list of bugs found (test failures + backend errors + console errors)
```

### Phase 3 — Interactive Investigation (Task Sonnet, only if bugs found)

For each bug found in Phase 2:
```
Use Playwright MCP to navigate to the affected page
Take screenshot
Read the accessibility tree
Check browser console for errors
Return: detailed diagnostic per bug (probable cause, file, suggested fix)
```

### Phase 4 — Report (main conversation)

Synthesize results from all agents:
- List bugs classified by severity (critical / high / medium / low)
- Include screenshots and diagnostic details
- Propose a fix plan with affected files

## Important

- **NEVER** run tests or analyze logs in the main conversation thread
- **ALWAYS** delegate to Task agents
- If Phase 1 fails, stop and report the stack issue
- If all tests pass with no backend errors, report "All clear" with a summary
