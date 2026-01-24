# Database Cleanup Scripts

## cleanup_employees.go

This script deletes all employees from the database except those with the `admin` role.

### What it does:
- Deletes all related data for non-admin employees:
  - Leaves
  - Leave accruals
  - Leave taken records
  - Leave carry over records
  - Identity information
  - Employment details
  - Employment history
  - Position assignments
  - Documents
  - Work lifecycle events
  - Onboarding/offboarding processes
  - Compliance records
  - Audit logs
  - Leave audits
- Deletes all non-admin employees
- Preserves all admin users

### How to run:

```bash
cd /home/andrea/Documents/Sources/hrms-api
go run scripts/cleanup_employees.go
```

The script will ask for confirmation. Type `DELETE` to proceed.

### ⚠️ WARNING:
This operation is **irreversible**. Make sure you have a database backup before running this script.
