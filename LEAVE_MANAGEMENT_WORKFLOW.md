# Leave Management Workflow

## Overview

The system uses an **Admin-Centered Workflow** where only administrators manage all employee leave data. This provides centralized control and ensures consistency across the organization.

**Note**: While employee self-service leave request functionality exists in the system (`/api/leaves`), it is **not used** in this workflow. All leave management is handled exclusively by administrators.

---

## Admin-Centered Workflow

### Core Principle
**Only administrators can create, update, and manage leave records for employees.** This ensures:
- Centralized control over leave data
- Consistent leave management practices
- Direct balance management without approval workflows
- Complete audit trail of all changes

---

## Workflow Steps

### Step 1: Monthly Accrual Processing (Admin)

**When**: Typically at the beginning of each month (or end of previous month)

**What Admin Does**:
1. Navigate to "Process Accruals" in admin dashboard
2. Select month (defaults to current month)
3. Choose "All Employees" or "Selected Employees"
4. Click "Process Accruals"

**What System Does Automatically**:
- Adds 2.0 days to each employee's balance
- Calculates days used from approved leaves in that month
- Updates balance: `New Balance = Previous Balance + 2.0 - Days Used`
- Creates/updates accrual record for that month

**Example**:
- Employee had 10 days at end of January
- Admin processes February accruals
- Employee gets +2.0 days = 12 days
- If employee used 3 days in February (approved leaves), balance = 12 - 3 = 9 days

### Step 2: Creating Leave Records (Admin)

**Admin Actions**:
1. Admin creates leave record directly for any employee via admin dashboard
2. Admin selects:
   - Employee
   - Leave type
   - Start date and end date
   - Reason (optional)
   - Status (defaults to "Approved" for admin-created leaves)
3. System automatically:
   - Checks for overlapping leaves
   - Validates leave balance (if status is Approved)
   - Updates carry-over usage if applicable
   - Records in audit log

**API Endpoint**:
```
POST /api/hr/leaves
{
  "employee_id": 1,
  "leave_type_id": 1,
  "start_date": "2025-12-01",
  "end_date": "2025-12-05",
  "reason": "Annual vacation",
  "status": "Approved"  // Optional, defaults to Approved
}
```

**What Happens**:
- Leave record is created immediately
- If status is "Approved", balance is checked and carry-over usage is updated
- When monthly accrual is processed, system automatically deducts days used from balance
- Complete audit trail is maintained

### Step 3: Managing Leave Records (Admin)

**Admin Can**:
- **View all leaves** for any employee: `GET /api/hr/employees/{id}/leaves`
- **Update leave records**: `PUT /api/hr/leaves/{id}` (change dates, status, reason)
- **Delete leave records**: `DELETE /api/hr/leaves/{id}` (soft delete)
- **Filter leaves** by status, leave type, date range

**Update Leave Example**:
```
PUT /api/hr/leaves/{id}
{
  "start_date": "2025-12-02",  // Change dates
  "end_date": "2025-12-06",
  "status": "Approved",        // Change status
  "reason": "Updated reason"
}
```

### Step 4: Balance Management (Admin)

**Admin Can Directly Adjust Balances**:

**Option A: Adjust Balance Directly**
```
POST /api/hr/employees/{id}/annual-leave-balance/adjust
{
  "days": -5.0,  // Deduct 5 days
  "reason": "Unauthorized absence"
}
```
- Instantly updates balance
- Updates DaysUsed automatically
- Records in audit log

**Option B: Add Manual Accrual**
```
POST /api/hr/employees/{id}/annual-leave-balance/accrual
{
  "month": "2025-02",
  "days": 2.0,
  "reason": "Manual accrual"
}
```
- Adds accrual for specific month
- Useful for corrections or bonuses

**Option C: Set Initial Balance**
```
POST /api/hr/employees/{id}/annual-leave-balance/set-initial
{
  "balance": 15.5,
  "days_used": 8.5,
  "days_accrued": 24.0,
  "reason": "Initial balance from old system"
}
```
- Sets absolute balance value
- Useful for onboarding employees from old system

### Step 5: End of Month (Admin)

**Generate Monthly Report**:
1. Admin exports monthly report (same format as Excel): `GET /api/hr/leaves/monthly-report/export?month=2025-02`
2. Review balances
3. Make any manual adjustments if needed

---

## Complete Monthly Cycle Example

### January 2025

**Jan 1-31**: Normal operations
- Admin creates leave records for employees as needed
- Admin updates leave records if dates change
- System tracks everything automatically

**Feb 1** (or last day of Jan): Admin processes January accruals
- System adds 2.0 days to all employees
- System calculates days used in January from approved leaves
- System updates balances automatically
- Example: Employee with 8 days + 2.0 accrued - 3 days used = 7 days balance

**Feb 2**: Admin generates January monthly report
- Exports Excel file
- Reviews all balances
- Makes any necessary adjustments

**Throughout February**: Repeat cycle

---

## Key Points

### ✅ What's Automatic:
1. **Balance calculation** when accruals are processed
2. **Days used calculation** from approved leaves
3. **Carry-over tracking** (if enabled)
4. **Audit logging** of all changes
5. **Overlap checking** when creating/updating leaves

### ✅ What Admin Controls:
1. **Creating leave records** for any employee
2. **Updating leave records** (dates, status, reason)
3. **Deleting leave records** when needed
4. **When to process accruals** (can be automated with cron job)
5. **Manual balance adjustments** when needed
6. **Report generation** for management

### ✅ What Employees Can Do (Not Used in This Workflow):
- The system includes employee self-service endpoints (`/api/leaves`) but these are **not used** in the admin-centered workflow
- Employees can view their own leave balance and history (if needed)
- All leave creation and management is handled by administrators

---

## Recommended Setup

### Option 1: Fully Automated Accruals
1. **Set up cron job** to process accruals automatically on 1st of each month
2. Admin creates leave records as needed throughout the month
3. System automatically calculates days used when accruals are processed
4. Admin reviews reports and handles exceptions

### Option 2: Manual Accrual Processing (Current Setup)
1. Admin manually processes accruals monthly (one click for all employees)
2. Admin creates leave records for employees as needed
3. System automatically calculates days used from approved leaves
4. Admin reviews and adjusts as needed

### Option 3: Fully Manual Management
1. Admin processes accruals monthly
2. Admin creates leave records and manually adjusts balances
3. Admin generates reports

---

## Important Notes

### Admin Leave Management:
- **Only admins** can create, update, or delete leave records
- Admin-created leaves default to "Approved" status (can be changed)
- System automatically checks for overlapping leaves
- System validates balance before approving leaves
- All changes are logged in audit trail

### When Accruals Are Processed:
- **System calculates days used** from all **approved leaves** in that month
- If a leave spans multiple months, it calculates the portion in each month
- Manual adjustments are preserved (won't be overwritten)

### Balance Calculation Formula:
```
New Balance = Previous Month Balance + 2.0 (Accrued) - Days Used (from approved leaves)
```

### Days Used Calculation:
- System looks at all **approved leaves** (not pending)
- Calculates overlap with the accrual month
- Automatically deducts from balance

### Manual Adjustments:
- If admin adjusts balance manually, it's preserved
- System won't overwrite manual adjustments when processing accruals
- All adjustments are logged with reason

---

## Example Scenarios

### Scenario 1: Normal Month (Admin-Centered)
1. Employee starts with 10 days
2. Admin creates leave record: 3 days leave, status "Approved"
3. Admin processes monthly accrual:
   - Adds 2.0 days (now 12 days)
   - System sees approved 3-day leave in this month
   - Deducts 3 days (now 9 days)
4. **Result**: 9 days balance

### Scenario 2: Leave Spans Two Months
1. Admin creates leave: Jan 28 - Feb 2 (5 days total), status "Approved"
2. Admin processes January accrual:
   - System calculates: 4 days in January (Jan 28-31)
   - Deducts 4 days from January balance
3. Admin processes February accrual:
   - System calculates: 1 day in February (Feb 1-2)
   - Deducts 1 day from February balance

### Scenario 3: Admin Updates Leave
1. Admin creates leave: Dec 1-5 (5 days), status "Pending"
2. Later, admin updates leave to status "Approved"
3. System checks balance and updates carry-over usage
4. When accrual is processed, days are deducted from balance

### Scenario 4: Manual Adjustment
1. Employee has 10 days
2. Employee takes unauthorized leave (no leave record)
3. Admin manually adjusts balance: -3 days
4. **Result**: 7 days balance, DaysUsed increased by 3
5. When next accrual is processed, this adjustment is preserved

---

## API Endpoints Summary

### Admin Leave Management (Admin Only)
- `POST /api/hr/leaves` - Create leave record for employee
- `PUT /api/hr/leaves/{id}` - Update leave record
- `DELETE /api/hr/leaves/{id}` - Delete leave record
- `GET /api/hr/employees/{id}/leaves` - Get all leaves for employee

### Balance Management (Admin/Manager)
- `GET /api/hr/employees/{id}/annual-leave-balance` - Get balance details
- `POST /api/hr/employees/{id}/annual-leave-balance/adjust` - Adjust balance
- `POST /api/hr/employees/{id}/annual-leave-balance/set-initial` - Set initial balance
- `POST /api/hr/employees/{id}/annual-leave-balance/accrual` - Add manual accrual
- `POST /api/hr/leaves/process-accruals` - Process monthly accruals

### Reports (Admin/Manager)
- `GET /api/hr/leaves/monthly-report` - Get monthly report
- `GET /api/hr/leaves/monthly-report/export` - Export monthly report
- `GET /api/hr/employees/annual-leave-balances` - Get all balances
- `GET /api/hr/employees/annual-leave-balances/export` - Export all balances

### Employee Endpoints (Not Used in This Workflow)
- `POST /api/leaves` - Employee applies for leave (exists but not used)
- `GET /api/leaves` - Employee views own leaves (can be used for viewing)
- `GET /api/leaves/balance` - Employee views own balance (can be used for viewing)

---

## Summary

**The workflow is admin-centered**:
- ✅ Only admins create and manage leave records
- ✅ Centralized control over all leave data
- ✅ No approval workflow needed (admin directly approves)
- ✅ Complete audit trail of all changes

**The system handles**:
- ✅ Automatic balance calculations
- ✅ Automatic days used tracking from approved leaves
- ✅ Overlap checking when creating/updating leaves
- ✅ Preservation of manual adjustments
- ✅ Complete audit trail

**Admin responsibilities**:
- Create leave records for employees as needed
- Process monthly accruals (one click)
- Review and adjust balances when needed
- Generate reports for management

