# Testing Carry-Over Functionality

## Quick Start

### Option 1: Using Make (Recommended)

```bash
# 1. Start PostgreSQL database
make docker-up

# 2. Install dependencies (if not already done)
make deps

# 3. Run the application
make run
```

### Option 2: Manual Steps

```bash
# 1. Start PostgreSQL database
docker-compose up -d

# 2. Install Go dependencies
go mod download

# 3. Run the application
go run main.go
```

The server will start on `http://localhost:8070` (or the port in your `.env` file).

## Testing Carry-Over Endpoints

### 1. Get Authentication Token

First, login as an admin or manager to get a JWT token:

```bash
# Login as Admin
curl -X POST http://localhost:8070/auth/admin/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password123"
  }'

# Or login as Manager
curl -X POST http://localhost:8070/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "nrc": "987654/32/1",
    "password": "password123"
  }'
```

Save the `token` from the response.

### 2. Enable Carry-Over for Annual Leave (if not already enabled)

```bash
# Get leave types to find Annual leave ID
curl -X GET http://localhost:8070/api/leave-types \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"

# Update Annual leave type to enable carry-over (replace {id} with actual ID)
curl -X PUT http://localhost:8070/api/leave-types/{id} \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Annual",
    "max_days": 24,
    "allow_carry_over": true,
    "max_carry_over_days": 5.0,
    "carry_over_expiry_months": 3
  }'
```

### 3. Process Year-End Carry-Over

Process carry-over for all employees for a specific year:

```bash
curl -X POST http://localhost:8070/api/hr/leaves/process-carryover \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "leave_type_id": 1,
    "from_year": 2024
  }'
```

**Response:**
```json
{
  "message": "Carry-over processing completed",
  "from_year": 2024,
  "processed": 2,
  "skipped": 0
}
```

### 4. Get Carry-Over History for an Employee

```bash
# Replace {id} with employee ID (e.g., 1)
curl -X GET http://localhost:8070/api/hr/employees/{id}/carryover-history?leave_type_id=1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Response:**
```json
[
  {
    "id": 1,
    "employee_id": 1,
    "leave_type_id": 1,
    "from_year": 2024,
    "to_year": 2025,
    "days_carried_over": 5.0,
    "days_used": 0.0,
    "days_remaining": 5.0,
    "expiry_date": "2025-03-31T00:00:00Z",
    "is_expired": false,
    "processed_at": "2025-01-15T10:30:00Z"
  }
]
```

### 5. Get Carry-Over Balance

```bash
curl -X GET http://localhost:8070/api/hr/employees/{id}/carryover-balance?leave_type_id=1 \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Response:**
```json
{
  "total_balance": 5.0,
  "active_carryovers": [
    {
      "id": 1,
      "from_year": 2024,
      "days_carried_over": 5.0,
      "days_remaining": 5.0,
      "expiry_date": "2025-03-31T00:00:00Z"
    }
  ],
  "expired_carryovers": []
}
```

### 6. Get Annual Leave Balance (includes carry-over)

```bash
curl -X GET http://localhost:8070/api/hr/employees/{id}/annual-leave-balance \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Response includes `carry_over_balance` field:**
```json
{
  "employee_id": 1,
  "employee_name": "John Doe",
  "total_accrued": 24.0,
  "total_used": 5.0,
  "current_balance": 19.0,
  "carry_over_balance": 5.0,
  "accruals": [...],
  "pending_leaves": 0,
  "upcoming_leaves": 0
}
```

### 7. Expire Carry-Overs

Manually expire carry-overs that have passed their expiry date:

```bash
curl -X POST http://localhost:8070/api/hr/leaves/expire-carryovers \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Complete Test Scenario

Here's a complete test scenario to verify carry-over works end-to-end:

### Step 1: Setup
```bash
# Start database and server (see Quick Start above)
# Login and get token
TOKEN="your_token_here"
EMPLOYEE_ID=1
LEAVE_TYPE_ID=1  # Annual leave
```

### Step 2: Create some leave accruals for 2024
The system automatically creates accruals, but you can verify:
```bash
curl -X GET http://localhost:8070/api/hr/employees/$EMPLOYEE_ID/annual-leave-balance \
  -H "Authorization: Bearer $TOKEN"
```

### Step 3: Process carry-over for 2024
```bash
curl -X POST http://localhost:8070/api/hr/leaves/process-carryover \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"leave_type_id\": $LEAVE_TYPE_ID,
    \"from_year\": 2024
  }"
```

### Step 4: Verify carry-over was created
```bash
curl -X GET http://localhost:8070/api/hr/employees/$EMPLOYEE_ID/carryover-balance?leave_type_id=$LEAVE_TYPE_ID \
  -H "Authorization: Bearer $TOKEN"
```

### Step 5: Apply for leave (should use carry-over)
```bash
# Login as employee first to get employee token
EMPLOYEE_TOKEN="employee_token_here"

curl -X POST http://localhost:8070/api/leaves \
  -H "Authorization: Bearer $EMPLOYEE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "leave_type_id": 1,
    "start_date": "2025-02-01",
    "end_date": "2025-02-05",
    "reason": "Test leave using carry-over"
  }'
```

### Step 6: Approve the leave (this will deduct from carry-over)
```bash
LEAVE_ID=1  # From previous response

curl -X PUT http://localhost:8070/api/leaves/$LEAVE_ID/approve \
  -H "Authorization: Bearer $TOKEN"
```

### Step 7: Verify carry-over was used
```bash
curl -X GET http://localhost:8070/api/hr/employees/$EMPLOYEE_ID/carryover-balance?leave_type_id=$LEAVE_TYPE_ID \
  -H "Authorization: Bearer $TOKEN"
```

You should see `days_used` increased and `days_remaining` decreased.

## Using Swagger UI

The easiest way to test is using Swagger UI:

1. Start the server
2. Open browser: `http://localhost:8070/swagger/index.html`
3. Click "Authorize" and enter: `Bearer YOUR_TOKEN_HERE`
4. Test the endpoints interactively

## Troubleshooting

### Database connection error
- Make sure PostgreSQL is running: `docker ps`
- Check database is accessible: `docker-compose ps`

### Migration errors
- The database will auto-migrate on startup
- If you see errors, you may need to reset: `docker-compose down -v` then restart

### No carry-over created
- Check that Annual leave type has `allow_carry_over: true`
- Verify employee has unused balance at year-end
- Check that `from_year` matches the year you want to process

### Balance not including carry-over
- Make sure you're using `GetCurrentLeaveBalance` (it's used automatically in balance endpoints)
- Verify carry-over records exist and are not expired

