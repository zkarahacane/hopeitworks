# Story R-0-1: [FRONT] Rename "Users" → "Administration" in sidebar

Status: ready-for-dev

## Story

As a **platform administrator**,
I want to see "Administration" with a shield icon in the sidebar instead of "Users" with a users icon,
so that the navigation item better reflects its purpose as an admin control panel rather than just user listing.

## Acceptance Criteria (BDD)

### Scenario 1: Admin user sees the renamed sidebar item

```gherkin
Given I am logged in as an admin user
When I view the application sidebar
Then I see "Administration" as the navigation label (not "Users")
  And the icon displayed is a shield icon (pi pi-shield)
  And the item is still only visible to admin users
```

### Scenario 2: Non-admin user does not see the item

```gherkin
Given I am logged in as a non-admin user
When I view the application sidebar
Then the "Administration" item is not visible
```

### Scenario 3: Navigation still works

```gherkin
Given I am logged in as an admin user
When I click the "Administration" sidebar item
Then I am navigated to the same admin/user management page as before
```

## Tasks / Subtasks

- [ ] **1.1** [FRONT] In `frontend/src/ui/layout/AppSidebar.vue`, update the `adminNavItems` entry: change `label` from `"Users"` to `"Administration"` and `icon` from `"pi pi-users"` to `"pi pi-shield"` (AC: #1, #2, #3)

## Dev Notes

### Dependencies

None. No backend or API changes required.

### Architecture Requirements

This is a pure frontend label and icon change. The routing target, RBAC visibility logic, and all other behaviour remain unchanged.

### Technical Specifications

**File to edit:** `frontend/src/ui/layout/AppSidebar.vue`

Locate the `adminNavItems` array (or equivalent constant) that defines the admin navigation entry. Apply two changes to that entry:

```diff
-  { label: 'Users',           icon: 'pi pi-users',  to: '/admin/users' }
+  { label: 'Administration',  icon: 'pi pi-shield', to: '/admin/users' }
```

The `to` route path is unchanged. Only `label` and `icon` are modified.

### Testing Requirements

- Visual verification: sidebar shows "Administration" with shield icon for admin users.
- No unit test required for a two-field string change.
- Existing E2E tests that assert on the "Users" label text must be updated to assert on "Administration".

Check for any test files that match:

```bash
grep -r "Users" frontend/e2e/ frontend/src/
```

Update any snapshot or text assertions that reference the old label.

### References

- `frontend/src/ui/layout/AppSidebar.vue` — navigation item definitions
- PrimeIcons reference: `pi pi-shield` is available in PrimeIcons v7+

## Dev Agent Record

## Change Log
