# PR #156 Feedback Transcript

## Pull Request: Fix non-deterministic table ordering in topological sort

### Timeline of Comments and Feedback

---

#### **p-c-h-b** | November 8, 2025, 03:00
**Initial PR Description:**

Outlined the fix for FK constraint ordering issues when referenced tables don't exist yet. The contributor explained that "FK-heavy tables could be reordered alphabetically" causing creation failures, and proposed deterministic cycle breaking with deferred constraints applied via ALTER TABLE.

---

#### **Copilot AI** | November 8, 2025, 03:00
**Automated Review:**

Generated pull request overview noting:
- Refactored topological sort to handle cycles using insertion order as tiebreaker
- Added deferred constraint mechanism
- Deferred RLS policy creation

---

#### **p-c-h-b** | November 8, 2025, 22:15
**Author Response to Copilot:**

- Added validation for empty constraint names in generateDeferredConstraintsSQL
- Fixed the comment to accurately reflect insertion order instead of alphabetical order

---

#### **tianzhou** | November 8, 2025, 03:43
**Reviewer Feedback #1:**

Requested two actions:
1. Enhance the table_to_table test case to validate the fix
2. Remove test artifact files (`*_actual.sql`, `*_expected.sql`)

---

#### **p-c-h-b** | November 8, 2025, 22:28
**Author Response:**

Addressed feedback by enhancing dependency test with circular FK dependencies between departments and users tables to validate constraint deferral.

---

#### **tianzhou** | November 9, 2025, 02:55
**Reviewer Question:**

"have you forgot to push the update?"

---

#### **p-c-h-b** | November 9, 2025, 09:37
**Author Update:**

Pushed corrections:
- Removed test artifacts
- Fixed employee_status_log constraint to stay inline (one-way dependency)
- Updated shouldDeferConstraint() logic to check existing tables from old schema

---

#### **tianzhou** | November 10, 2025, 02:55
**Reviewer Feedback #2 (Latest):**

Identified remaining issues:

1. **Policy Ordering Problem**: In `testdata/dump/employee/pgschema.sql`, RLS policies should be positioned alongside their corresponding `ALTER TABLE...ENABLE ROW LEVEL SECURITY` statements rather than separated. Currently policies appear much later after functions and procedures.

2. **File Exclusion**: Directed the contributor to "exclude this from the PR" regarding the `cmd/dump/employee_expected.sql` file - it shouldn't be committed.

Noted overall progress with "Almost there," indicating the submission is nearly complete pending these structural adjustments.

---

#### **tianzhou** | November 10, 2025, 15:45
**Reviewer Feedback #3:**

- Reported that emitting `CREATE POLICY` immediately after enabling RLS causes migrations to fail when policies reference helper functions defined later in the diff.
- Requested restoring deferred policy creation until after all new functions/procedures exist.

---

#### **p-c-h-b** | November 10, 2025, 18:02
**Author Update:**

- Added selective policy deferral: policies stay co-located with their tables unless they reference helper functions introduced in the same migration, in which case they're executed after the corresponding functions/procedures are created.
- Updated the employee dump fixture to match the dependency-safe ordering.

---

## Summary of Key Issues Addressed

### Round 1 (Nov 8):
- Enhanced test coverage for circular FK dependencies
- Removed test artifact files

### Round 2 (Nov 9):
- Fixed constraint deferral logic
- Cleaned up test artifacts

### Round 3 (Nov 10 - Current):
- **Policy Ordering**: Policies must be co-located with RLS enable statements
- **Test Artifacts**: Remove `cmd/dump/employee_expected.sql`
- **Policy Deferral**: Ensure CREATE POLICY statements execute only after dependent functions/procedures when helper functions are newly added

---

## Current Status

**Status**: Feedback addressed in latest commits
- ✅ Removed test artifact file (`cmd/dump/employee_expected.sql`)
- ✅ RLS enable statements remain co-located with their tables, with policies only deferred when they depend on helper functions added in the same migration
- ✅ Updated test fixture (`testdata/dump/employee/pgschema.sql`)
- ✅ All tests passing
- ✅ Restored dependency-safe policy creation without sacrificing readability
