# Simplified Leave Management System

This document describes the simplified admin-only leave management system implementation.

## Overview

The system has been simplified to:
- **Admin-only model**: Employees do not apply for leave; admins record leave taken
- **2 days per month accrual**: Each employee earns 2 days per month automatically
- **Simple balance calculation**: Total Accrued - Total Taken = Balance
- **No approval workflow**: Admin has full control to record leave

## Schema Changes

### 1. `leave_accruals` Table (Simplified)

```sql
leave_accruals
--------------
id (PK)
employee_id (FK)
leave_type_id (FK)
year (INT)              -- Year (e.g., 2026)
month (INT)             -- Month (1-12)
days_accrued (FLOAT)    -- Days accrued (e.g., 2.0)
created_at
updated_at
```

**Changes:**
- Removed `accrual_month` (date field) → Replaced with `year` and `month` (integers)
- Removed `days_used` → Calculated from `leave_taken` table
- Removed `days_balance` → Calculated dynamically
- Removed `is_processed`, `processed_at`, `notes` → Simplified

### 2. `leave_taken` Table (New)

```sql
leave_taken
-----------
id (PK)
employee_id (FK)
leave_type_id (FK)
start_date (DATE)
end_date (DATE)
days_taken (FLOAT)      -- Calculated days taken
recorded_by (FK)        -- Admin user ID
remarks (TEXT)
recorded_at (TIMESTAMP)
created_at
updated_at
```

**Features:**
- No approval workflow
- Admin records actual leave taken
- Clean audit trail with `recorded_by` and `recorded_at`

### 3. `leave_types` Table (Updated)

Added:
- `accrual_rate` (FLOAT) - Days per month (default: 2.0 for Annual Leave)

### 4. `employees` Table (Updated)

Added:
- `employee_number` (VARCHAR, UNIQUE) - Employee number
- `date_joined` (DATE) - Date employee joined
- `status` (VARCHAR) - active, inactive

## Balance Calculation

### Formula

```text
Total Accrued – Total Taken = Balance
```

### Implementation

```go
// Get total accrued
SELECT COALESCE(SUM(days_accrued), 0) 
FROM leave_accruals 
WHERE employee_id = ? AND leave_type_id = ?

// Get total taken
SELECT COALESCE(SUM(days_taken), 0) 
FROM leave_taken 
WHERE employee_id = ? AND leave_type_id = ?

// Balance = Total Accrued - Total Taken
```

**Benefits:**
- Always correct (no sync issues)
- Fully traceable
- Excel-migration friendly

## Monthly Accrual Job

### Automatic Processing

Runs on the 1st of each month at 2:00 AM (cron: `0 0 2 1 * *`)

**Process:**
1. For each active employee:
   - Check if employee was active during the month
   - If employed during the month:
     - Insert `leave_accruals` record with 2.0 days
   - Optional: Prorate for mid-month joins

### Manual Processing

Admins can manually trigger accrual processing via:
```
POST /api/hr/leaves/process-accruals
```

## Admin Workflow

### 1. System Auto-Adds 2 Days/Month
- Automatic monthly accrual job runs on 1st of each month
- Creates `leave_accruals` records for all active employees

### 2. Employee Goes on Leave
- Employee takes leave (no application needed)

### 3. Admin Records Leave
- Admin selects employee
- Enters start date and end date
- System calculates `days_taken` automatically
- Balance updates automatically (calculated on-the-fly)

## API Endpoints

### Simplified Admin Endpoints

#### Record Leave Taken
```
POST /api/admin/leave-taken
Body: {
  "employee_id": 1,
  "leave_type_id": 1,
  "start_date": "2026-01-15",
  "end_date": "2026-01-17",
  "remarks": "Annual vacation"
}
```

#### Get Employee Leave Balance
```
GET /api/admin/employees/{id}/leave-balance?leave_type_id=1
Response: {
  "employee_id": 1,
  "employee_name": "John Doe",
  "leave_type_id": 1,
  "total_accrued": 24.0,
  "total_taken": 5.0,
  "balance": 19.0
}
```

#### Get Employee Leave History
```
GET /api/admin/employees/{id}/leave-taken?leave_type_id=1
Response: [array of leave_taken records]
```

#### Get All Employees Leave Balances
```
GET /api/admin/employees/leave-balances?leave_type_id=1
Response: [array of balance objects]
```

## Migration Notes

### Database Migration

The system uses GORM AutoMigrate, which will:
1. Add new columns to existing tables
2. Create new `leave_taken` table
3. Update `leave_accruals` table structure

**Important:** Existing data in `leave_accruals` with `accrual_month` will need manual migration to `year` and `month` fields if you have existing data.

### Backward Compatibility

The old endpoints (`/api/hr/leaves/*`) are still available for backward compatibility, but the new simplified endpoints (`/api/admin/*`) are recommended for the admin-only workflow.

## Future Enhancements

The simplified schema makes it easy to add:
- Carry-forward rules
- Max caps
- Leave requests (if policy changes)
- Sick leave
- Other leave types

## Example Usage

### 1. Monthly Accrual (Automatic)
```go
// Runs automatically on 1st of each month
// Creates: leave_accruals record with year=2026, month=1, days_accrued=2.0
```

### 2. Admin Records Leave
```bash
curl -X POST http://localhost:8080/api/admin/leave-taken \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "employee_id": 1,
    "leave_type_id": 1,
    "start_date": "2026-01-15",
    "end_date": "2026-01-17",
    "remarks": "Annual vacation"
  }'
```

### 3. Check Balance
```bash
curl -X GET http://localhost:8080/api/admin/employees/1/leave-balance \
  -H "Authorization: Bearer <admin_token>"
```

## Summary

✅ **Simplified Schema**: Clean, minimal tables
✅ **No Approval Workflow**: Admin records leave directly
✅ **Automatic Accrual**: 2 days/month automatically added
✅ **Dynamic Balance**: Always calculated correctly
✅ **Audit Trail**: Full tracking with `recorded_by` and `recorded_at`
✅ **Excel-Friendly**: Easy to export and migrate data
