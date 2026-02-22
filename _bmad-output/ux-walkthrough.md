# UX Walkthrough -- hopeitworks (2026-02-22)

## Summary

- **Total issues found: 22**
- Critical: 3 | High: 5 | Medium: 8 | Low: 6
- **Tested as:** admin@hopeitworks.dev (admin) and dev@hopeitworks.dev (user)
- **Stack:** Frontend http://localhost:5173, API http://localhost:8080, MailHog http://localhost:8025

---

## 1. Login Page (`/login`)

### What works
- Page renders correctly with centered layout
- Title "hopeitworks" displayed
- Email field with placeholder "you@example.com"
- Password field with eye toggle (show/hide)
- "Forgot password?" link navigates to `/forgot-password`
- "Sign In" button submits form
- Nav/sidebar is correctly **hidden** on login page (fix-8 verified)
- Login with `admin@hopeitworks.dev` / `admin123` (8 chars) **succeeds**
- Redirects to dashboard after login with `?redirect=/` parameter honored

### Issues
- **[CRITICAL]** Password frontend validation enforces min 8 characters, but seed user `dev@hopeitworks.dev` has password `dev123` (6 chars). Login is impossible via the UI for this user. The API accepts 6-char passwords. Mismatch between frontend validation and backend/seed data.
- **[LOW]** "Enter a password" hint text displayed below empty password field -- disappears on input. Not a bug per se, but could be confusing since it looks like an error.
- **[LOW]** Console warning: `Input elements should have autocomplete attributes` -- missing autocomplete on email and password fields (accessibility/UX best practice).
- **[LOW]** Console error: 401 on `/api/v1/auth/me` on page load -- expected but noisy in dev console.

---

## 2. Forgot Password Page (`/forgot-password`)

### What works
- Clean centered layout with "Reset your password" subtitle
- Email field with placeholder
- "Send reset link" blue button submits the form
- After submission: green success alert "Check your email. If an account exists for that email, you will receive a reset link shortly." -- good security practice (doesn't reveal email existence)
- "Back to login" link with arrow works
- Nav is hidden (correct)

### Issues
- No issues found. This flow works well end-to-end.

---

## 3. Password Reset Email (MailHog)

### What works
- Email received in MailHog inbox within seconds
- From: `noreply@hopeitworks.local`
- Subject: "Reset your HopeItWorks password"
- HTML email with:
  - "Password Reset Request" heading
  - Personalized greeting ("Hi Admin User,")
  - 1-hour expiry notice
  - "Reset my password" CTA button
  - Security disclaimer
- Reset link points to `http://localhost:5173/reset-password?token=...`

### Issues
- No issues found. Email content and delivery work correctly.

---

## 4. Reset Password Page (`/reset-password?token=...`)

### What works
- "Set new password" subtitle
- New password and Confirm new password fields with eye toggles
- "Set new password" blue button
- Nav hidden (correct)

### Issues
- **[MEDIUM]** Missing "Back to login" link (the forgot-password page has one, but reset-password does not)
- **[LOW]** No indication of password requirements (min length, complexity) -- user has to guess and wait for validation error

---

## 5. Dashboard (`/`)

### What works
- "Dashboard" heading displayed
- Sidebar navigation visible: Dashboard (highlighted), Projects, Runs, Approvals, Settings
- Footer: "Connected" status (left), "v0.0.0" (right)
- "Skip to main content" accessibility link present

### Issues
- **[HIGH]** Dashboard is completely empty -- just a heading with no content, no widgets, no stats, no recent activity. Should at minimum show a welcome message or placeholder.
- **[MEDIUM]** Header shows "Hope" instead of "hopeitworks" -- app name is truncated in the header/navbar brand area.

---

## 6. User Menu (Header dropdown)

### What works
- Clicking user avatar/icon in top-right opens a dropdown menu
- Shows user name ("Admin User") and email (both disabled/non-interactive)
- Separator line between info and actions
- "My Profile" link navigates to `/profile`
- "Logout" link logs out and redirects to `/login`

### Issues
- No issues found. User menu works correctly.

---

## 7. Profile Page (`/profile`)

### What works
- Two-column layout: "Profile Information" (left) and "Change Password" (right)
- Name and Email fields are editable
- Role shown as colored badge ("admin" = red, "user" = blue)
- "Member since" date displayed
- "Save Changes" button disabled until changes are made, enables when field is modified
- Change Password section: Current Password, New Password, Confirm New Password (all with eye toggles)
- "Update Password" button disabled until all fields are filled
- Admin users see "Users" link in sidebar (separated by divider line)

### Issues
- **[LOW]** "Member since: 22 fevrier 2026" -- date is in French locale. Should match application language (English) or be configurable.
- **[MEDIUM]** Profile page is not highlighted in the sidebar navigation -- no nav item is active since "Profile" is not in the main nav (accessed via user menu only).

---

## 8. Projects List (`/projects`)

### What works
- "Projects" heading with "New Project" green button
- Table with columns: Name, Description, Created
- 4 seed/test projects displayed with correct data
- Description column truncates long text with ellipsis
- "Created" column shows relative time ("X minutes ago")
- Pagination at bottom (functional)
- Table rows are clickable (navigates to project detail)
- "Projects" nav item correctly highlighted

### Issues
- **[MEDIUM]** Missing columns that would be useful: Owner, GitHub repo URL, Status
- No other issues found. Clean, functional table view.

---

## 9. Create Project Dialog

### What works
- Modal dialog with dimmed overlay
- Name field (required, marked with *)
- Description textarea (resizable)
- Cancel and Create buttons
- Close (X) button

### Issues
- **[MEDIUM]** Missing GitHub repository URL field -- the project model likely supports this but it's not in the creation form
- **[LOW]** No "slug" or "key" field for short project identifier

---

## 10. Project Detail -- Overview Tab (`/projects/:id`)

### What works
- Project title displayed with back button
- Tab menubar: Overview, Board, Pipeline, Templates, Costs, Notifications
- Overview tab shows: Name, Created (formatted date), Description in a card

### Issues
- **[MEDIUM]** Missing: GitHub repo URL, Owner information, project status
- **[MEDIUM]** Missing: Edit project and Delete project buttons -- no way to modify project info after creation
- **[LOW]** Back button icon not visible (exists in DOM as `button "Back to projects"` but icon may not render)

---

## 11. Story Board -- Board Tab (`/projects/:id/board`)

### What works
- "Story Board" heading with "Import Stories" button
- Epic cards displayed in 3-column grid: Foundation, Task Management, User Authentication
- Each card shows: title, description, status counters (Backlog, Running, Done, Failed)
- Color-coded badges (blue=Backlog, yellow=Running, green=Done, red=Failed)
- Cards are clickable, navigating to epic stories view

### Issues
- **[MEDIUM]** All status counters show 0 for all epics, even though stories exist with various statuses (completed, running, backlog, failed) -- counters are not aggregating story statuses correctly
- Missing "Create Epic" button -- no way to add new epics from the UI

---

## 12. Epic Stories View (`/projects/:id/epics/:id`)

### What works
- Master-detail layout: story list (left) and detail panel (right)
- "Epic Stories" heading with "View DAG" button and back button
- Status filter dropdown (All statuses, Backlog, Running, Done, Failed) -- works
- Search box -- works (filters by text instantly)
- "Create story" button opens creation dialog
- Story list shows: key (S-01..S-07), status badge, title
- Clicking a story shows detail panel with: key, status badge, scope tag, title, Objective, Acceptance Criteria
- Create Story dialog has: Key*, Title*, Objective, Acceptance Criteria, Scope dropdown, Target Files section

### Issues
- **[HIGH]** Each story card shows a "Backlog" label with icon at the bottom, regardless of the actual status shown in the badge (e.g., S-01 shows "completed" badge but "Backlog" below). This is confusing and appears to be a duplicate/stale status display.

---

## 13. Epic DAG View (`/projects/:id/epics/:id/dag`)

### What works
- "Epic DAG" heading with "Launch Epic" green button and back button
- Vue Flow DAG framework is loaded with correct story nodes and edges (S-01 -> S-02 visible in DOM)
- Mini map and zoom controls exist in the DOM

### Issues
- **[CRITICAL]** DAG graph is NOT VISIBLE -- the Vue Flow container has zero height/width. Console warnings confirm: `[Vue Flow]: The Vue Flow parent container needs a width and a height to render the graph`. The entire graph is invisible despite data being loaded correctly. This is a CSS/layout bug.
- **[MEDIUM]** Tab bar shows "Overview" as highlighted instead of "Board" -- active tab state is lost when navigating to the DAG sub-view

---

## 14. Pipeline Configuration Tab (`/projects/:id/pipeline`)

### What works
- "Pipeline Configuration" heading with "Add Step" and "Save" buttons
- 3 pre-configured steps: implement, review, merge
- Each step shows: number, name, type badge, model (Claude Sonnet 4.5), trigger type (Manual/Auto)
- Steps are reorderable with move up/down buttons (correctly disabled at boundaries)
- Remove step buttons
- Clicking a step expands inline editing with: Model dropdown, Auto Approve checkbox, Max Retries spinbutton, Retry Type dropdown
- "Save" button disabled when no changes (correct)

### Issues
- No issues found. Well-designed pipeline configuration UI.

---

## 15. Prompt Templates Tab (`/projects/:id/templates`)

### What works
- Empty state: "No prompt templates found for this project." with description and "Create Template" button
- Template editor page with: Template Name field, Type dropdown (Custom), dark code editor with line numbers, Context Variables sidebar
- Context variables are clickable buttons (Handlebars syntax): story_key, story_title, story_objective, target_files, acceptance_criteria, error_context, diff_content, branch_name, repo_url
- Preview, Save, Cancel buttons

### Issues
- **[MEDIUM]** When navigating to template editor (`/templates/new`), the tab bar shows "Overview" as active instead of "Templates" -- active tab state is lost on sub-routes

---

## 16. Costs Tab (`/projects/:id/costs`)

### What works
- Period selector: 7d (active) and 30d toggle buttons
- 3 summary cards: "Total cost this week", "Total cost this month", "Average cost per story" (all $0.00)
- "Cost Over Time" chart area with "No cost data yet" empty state
- "Recent Runs" section with "No runs in this period" empty state
- Good dashboard layout even without data

### Issues
- **[HIGH]** Console errors (404) on `/costs/chart?period=7d` and `/costs/runs?period=7d` -- these API endpoints appear to not be implemented yet, causing errors on page load

---

## 17. Notifications Tab (`/projects/:id/settings/notifications`)

### What works
- "Notification Channels" heading with description and "Add Channel" button
- Empty state with dashed border: "No notification channels configured" + "Add your first channel" link
- Good visual design for empty state

### Issues
- No issues found.

---

## 18. Runs Page (`/runs`)

### Issues
- **[HIGH]** Page returns 404 "Page Not Found". The route is in the sidebar nav but has no view implemented. The 404 page itself is well-designed with "Go to Dashboard" button.

---

## 19. Approvals Page (`/approvals`)

### What works
- "Approvals" heading
- Table with columns: Story, Title, Project, PR, Waiting Since, Actions
- Empty state: "No pending approvals" (centered in table)

### Issues
- **[HIGH]** Console error (404) on `/api/v1/hitl-requests?status=pending` -- API endpoint not implemented
- Missing filter/search controls for approvals

---

## 20. Settings Page (`/settings`)

### Issues
- **[HIGH]** Page returns 404 "Page Not Found". Same as Runs -- route exists in nav but no view implemented.

---

## 21. User Management (`/admin/users`) -- Admin Only

### What works
- "User Management" heading with "Create User" green button
- Table with columns: Email, Name, Role, Created, Actions
- 3 seed users displayed correctly with role badges (admin=red, user=blue)
- Pagination at bottom
- Edit User dialog: Name, Email fields editable; Role shown but not editable
- Create User dialog: Name, Email, Password (with eye toggle), Role dropdown (default "Member")
- Non-admin users: "Users" link hidden from sidebar, direct URL access redirected to dashboard (access control works)

### Issues
- **[CRITICAL]** Action buttons (Edit/Delete) in the table are invisible -- icons do not render visually. Only a faint circle appears on hover. Users cannot discover these actions.
- **[MEDIUM]** Edit User dialog: Role field is displayed but NOT editable (no dropdown) -- admin cannot change user roles
- **[MEDIUM]** Create User dialog: Role dropdown shows "Member" but the table shows "user" -- inconsistent terminology between creation and display

---

## 22. Logout Flow

### What works
- User menu > Logout redirects to login page
- Nav/sidebar is hidden after logout (correct)
- Session is cleared

### Issues
- No issues found.

---

## 23. Cross-Cutting Observations

### Accessibility
- "Skip to main content" link is present (good)
- Form labels are present
- Console warnings about missing `autocomplete` attributes on input elements
- Keyboard navigation appears functional via tab order

### Responsiveness
- Not tested at mobile breakpoints during this walkthrough (desktop 1440x900 only)

### Performance
- Pages load quickly, no observable lag
- Pinia stores initialize lazily on first navigation to relevant pages (good)

### Security
- Auth token stored in HttpOnly cookie on `/api` path (good)
- Non-admin users correctly blocked from admin routes (both UI and route guard)
- Password reset email doesn't leak account existence
- CSRF protection not evaluated

---

## Issues Summary Table

| # | Page | Severity | Description |
|---|------|----------|-------------|
| 1 | Login | **CRITICAL** | Frontend enforces min 8-char password validation, but seed user `dev@hopeitworks.dev` has 6-char password `dev123`. Login impossible via UI for this user. |
| 2 | DAG View | **CRITICAL** | DAG graph not visible -- Vue Flow container has zero height/width. CSS/layout bug. |
| 3 | User Mgmt | **CRITICAL** | Edit/Delete action button icons are invisible in the Actions column. |
| 4 | Dashboard | HIGH | Dashboard page is completely empty -- no content, widgets, or stats beyond the heading. |
| 5 | Runs | HIGH | `/runs` page returns 404 -- route exists in nav but no view is implemented. |
| 6 | Settings | HIGH | `/settings` page returns 404 -- route exists in nav but no view is implemented. |
| 7 | Approvals | HIGH | API endpoint `/api/v1/hitl-requests` returns 404 -- not implemented. |
| 8 | Costs | HIGH | API endpoints `/costs/chart` and `/costs/runs` return 404 -- not implemented. |
| 9 | Epic Stories | HIGH | Story cards show duplicate/conflicting status -- badge says "completed" but a "Backlog" label also appears on each card. |
| 10 | Header | MEDIUM | App name shows "Hope" instead of "hopeitworks" -- truncated brand text. |
| 11 | Board | MEDIUM | Epic status counters all show 0 even though stories have various statuses. |
| 12 | DAG View | MEDIUM | Tab bar shows "Overview" active instead of "Board" when on DAG sub-route. |
| 13 | Template Editor | MEDIUM | Tab bar shows "Overview" active instead of "Templates" when on editor sub-route. |
| 14 | Project Detail | MEDIUM | No Edit/Delete buttons for project -- cannot modify project after creation. |
| 15 | Project Detail | MEDIUM | Missing GitHub repo URL, Owner, Status in overview. |
| 16 | Create Project | MEDIUM | Missing GitHub repository URL field in creation dialog. |
| 17 | User Mgmt | MEDIUM | Edit User dialog: Role is not editable -- admin cannot change user roles. |
| 18 | User Mgmt | MEDIUM | Role terminology inconsistency: "Member" in create dialog vs "user" in table display. |
| 19 | Reset Password | MEDIUM | Missing "Back to login" link on reset password page. |
| 20 | Profile | MEDIUM | Profile page has no active nav item highlighted in sidebar. |
| 21 | Profile | LOW | "Member since" date shown in French locale instead of English. |
| 22 | Reset Password | LOW | No password requirements shown to user (min length, complexity). |
