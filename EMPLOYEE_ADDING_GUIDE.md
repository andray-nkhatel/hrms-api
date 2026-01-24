# Employee Adding Process - Mockup Guide

After cleaning the database (keeping only admin users), you can add employees using the following methods:

## API Endpoints

### 1. Create Employee (Employee or Manager role)

**Endpoint:** `POST /api/employees`

**Authentication:** Required (Admin role)

**Request Body:**
```json
{
  "nrc": "123456/78/9",
  "firstname": "John",
  "lastname": "Doe",
  "email": "john.doe@example.com",
  "password": "password123",
  "department": "IT",
  "role": "employee",  // or "manager"
  "hire_date": "2024-01-15"  // Optional, defaults to today
}
```

**Response:**
```json
{
  "id": 1,
  "nrc": "123456/78/9",
  "firstname": "John",
  "lastname": "Doe",
  "email": "john.doe@example.com",
  "department": "IT",
  "role": "employee",
  "status": "active",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Notes:**
- NRC must be unique
- Email must be unique
- Role can be `employee` or `manager` (use `/api/admins` for admin accounts)
- Automatically creates `EmploymentDetails` with hire date
- Password is hashed before storage

### 2. Create Admin Account

**Endpoint:** `POST /api/admins`

**Authentication:** Required (Admin role)

**Request Body:**
```json
{
  "username": "admin2",
  "firstname": "Admin",
  "lastname": "User",
  "email": "admin2@example.com",
  "password": "password123",
  "department": "Administration"
}
```

**Response:**
```json
{
  "id": 2,
  "username": "admin2",
  "firstname": "Admin",
  "lastname": "User",
  "email": "admin2@example.com",
  "department": "Administration",
  "role": "admin",
  "status": "active",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Notes:**
- Username must be unique
- Email must be unique
- Role is automatically set to `admin`
- Uses `username` instead of `nrc`

### 3. Bulk Upload Employees (CSV)

**Endpoint:** `POST /api/employees/bulk`

**Authentication:** Required (Admin role)

**Content-Type:** `multipart/form-data`

**Request:**
- `file`: CSV file with employee data

**CSV Format:**
```csv
NRC,Firstname,Lastname,Email,Password,Department,Role,HireDate
123456/78/9,John,Doe,john.doe@example.com,password123,IT,employee,2024-01-15
987654/32/1,Jane,Smith,jane.smith@example.com,password123,HR,manager,2024-01-10
```

**Response:**
```json
{
  "total": 2,
  "success": 2,
  "failed": 0,
  "results": [
    {
      "nrc": "123456/78/9",
      "email": "john.doe@example.com",
      "success": true,
      "employee_id": 1
    },
    {
      "nrc": "987654/32/1",
      "email": "jane.smith@example.com",
      "success": true,
      "employee_id": 2
    }
  ]
}
```

### 4. Download CSV Template

**Endpoint:** `GET /api/employees/template`

**Authentication:** Required (Admin role)

**Response:** CSV file download

## Frontend Process

### Using the Admin Employees Page

1. Navigate to `/app/admin/employees` (Admin only)
2. Click "Add Employee" button
3. Fill in the form:
   - NRC (required, must be unique)
   - Firstname (required)
   - Lastname (required)
   - Email (required, must be unique)
   - Password (required)
   - Department (required)
   - Role (employee or manager)
   - Hire Date (optional, defaults to today)
4. Click "Create Employee"

### Bulk Upload via Frontend

1. Navigate to `/app/admin/employees`
2. Click "Bulk Upload" or "Import CSV"
3. Download the template CSV file
4. Fill in employee data
5. Upload the CSV file
6. Review results and fix any errors

## Example cURL Commands

### Create Employee
```bash
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "nrc": "123456/78/9",
    "firstname": "John",
    "lastname": "Doe",
    "email": "john.doe@example.com",
    "password": "password123",
    "department": "IT",
    "role": "employee",
    "hire_date": "2024-01-15"
  }'
```

### Create Manager
```bash
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -d '{
    "nrc": "987654/32/1",
    "firstname": "Jane",
    "lastname": "Manager",
    "email": "jane.manager@example.com",
    "password": "password123",
    "department": "HR",
    "role": "manager"
  }'
```

## Testing the Process

After running the cleanup script:

1. **Verify admin exists:**
   ```bash
   # Login as admin (username: admin, password: password123)
   ```

2. **Add test employees:**
   - Use the frontend at `/app/admin/employees`
   - Or use the API endpoints above

3. **Verify employees were created:**
   - Check the employees list in the frontend
   - Or call `GET /api/employees`

## Important Notes

- **NRC** is required for employees and managers (not for admins)
- **Username** is required for admins (not for employees/managers)
- **Email** must be unique across all users
- **Password** is hashed using bcrypt before storage
- **EmploymentDetails** are automatically created with the hire date
- **Leave accruals** will be automatically processed by the scheduler for active employees
