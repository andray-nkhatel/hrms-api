# HRMS Leave Management API

A comprehensive Leave Management System (LMS) API built with Go, Gin, and PostgreSQL. This RESTful API enables employees to apply for leaves, track leave balances, and allows managers/HR to approve or reject leave requests.

## Features

- **Employee Self-Service**: Apply for leave, view leave history, and check leave balance
- **Managerial Workflows**: View pending leave requests and approve/reject them
- **Admin Management**: Manage employees, leave types, and system settings
- **Role-Based Authentication**: JWT-based authentication with role-based access control
- **Business Logic Validation**: Prevents overlapping leaves and ensures sufficient leave balance
- **Secure**: Password hashing with bcrypt, JWT tokens with expiration

## Tech Stack

- **Backend**: Go (Golang)
- **Framework**: Gin
- **Database**: PostgreSQL
- **ORM**: GORM
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Validation**: go-playground/validator
- **Configuration**: godotenv

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for PostgreSQL)
- Make (optional, for convenience commands)

## Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd hrms-api
```

### 2. Start PostgreSQL Database

```bash
docker-compose up -d
```

This will start a PostgreSQL container on port 5432.

### 3. Configure Environment Variables

Create a `.env` file in the root directory:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=hrms_db

JWT_SECRET=your-secret-key-change-this-in-production
JWT_EXPIRATION_HOURS=24

PORT=8080
GIN_MODE=debug
```

### 4. Install Dependencies

```bash
go mod download
```

### 5. Run the Application

```bash
go run main.go
```

The server will start on `http://localhost:8080` (or the port specified in `.env`).

The application will automatically:
- Connect to the database
- Run migrations
- Seed initial leave types (Sick, Casual, Annual, Maternity, Paternity)

## API Endpoints

### Authentication

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "nrc": "123456/78/9",
  "password": "yourpassword"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "employee": {
    "id": 1,
    "nrc": "123456/78/9",
    "firstname": "John",
    "lastname": "Doe",
    "email": "john@example.com",
    "department": "IT",
    "role": "employee"
  }
}
```

#### Register
```http
POST /auth/register
Content-Type: application/json

{
  "nrc": "123456/78/9",
  "firstname": "John",
  "lastname": "Doe",
  "email": "john@example.com",
  "password": "securePassword123",
  "department": "IT",
  "role": "employee"
}
```

**Note:** Role can be `employee`, `manager`, or `admin`. Defaults to `employee` if not specified.

### Employee Endpoints

All employee endpoints require authentication. Include the JWT token in the Authorization header:
```
Authorization: Bearer <your-token>
```

#### Apply for Leave
```http
POST /api/leaves
Authorization: Bearer <token>
Content-Type: application/json

{
  "leave_type_id": 1,
  "start_date": "2024-02-01",
  "end_date": "2024-02-05",
  "reason": "Family vacation"
}
```

#### View Leave History
```http
GET /api/leaves
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "id": 1,
    "employee_id": 1,
    "leave_type_id": 1,
    "start_date": "2024-02-01T00:00:00Z",
    "end_date": "2024-02-05T00:00:00Z",
    "reason": "Family vacation",
    "status": "Pending",
    "created_at": "2024-01-15T10:30:00Z",
    "leave_type": {
      "id": 1,
      "name": "Annual",
      "max_days": 20
    }
  }
]
```

#### Check Leave Balance
```http
GET /api/leaves/balance
Authorization: Bearer <token>
```

**Response:**
```json
[
  {
    "leave_type_id": 1,
    "leave_type_name": "Annual",
    "max_days": 20,
    "used_days": 5,
    "balance": 15
  },
  {
    "leave_type_id": 2,
    "leave_type_name": "Sick",
    "max_days": 10,
    "used_days": 2,
    "balance": 8
  }
]
```

### Manager Endpoints

Manager endpoints require authentication with `manager` or `admin` role.

#### View Pending Leaves
```http
GET /api/leaves/pending
Authorization: Bearer <token>
```

#### Approve Leave
```http
PUT /api/leaves/{id}/approve
Authorization: Bearer <token>
```

#### Reject Leave
```http
PUT /api/leaves/{id}/reject
Authorization: Bearer <token>
```

### Admin Endpoints

Admin endpoints require authentication with `admin` role.

#### Leave Types Management

**Get All Leave Types**
```http
GET /api/leave-types
Authorization: Bearer <token>
```

**Create Leave Type**
```http
POST /api/leave-types
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Sabbatical",
  "max_days": 30
}
```

**Update Leave Type**
```http
PUT /api/leave-types/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Sabbatical",
  "max_days": 45
}
```

**Delete Leave Type**
```http
DELETE /api/leave-types/{id}
Authorization: Bearer <token>
```

#### Employee Management

**Get All Employees**
```http
GET /api/employees
Authorization: Bearer <token>
```

**Get Employee by ID**
```http
GET /api/employees/{id}
Authorization: Bearer <token>
```

**Create Employee**
```http
POST /api/employees
Authorization: Bearer <token>
Content-Type: application/json

{
  "nrc": "987654/32/1",
  "firstname": "Jane",
  "lastname": "Smith",
  "email": "jane@example.com",
  "password": "securePassword123",
  "department": "HR",
  "role": "manager"
}
```

**Update Employee**
```http
PUT /api/employees/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "firstname": "Jane",
  "lastname": "Doe",
  "department": "Finance",
  "role": "admin"
}
```

**Delete Employee**
```http
DELETE /api/employees/{id}
Authorization: Bearer <token>
```

## Database Schema

### Employees Table
- `id` (SERIAL PRIMARY KEY)
- `nrc` (VARCHAR(20), UNIQUE, NOT NULL)
- `firstname` (VARCHAR(50), NOT NULL)
- `lastname` (VARCHAR(50), NOT NULL)
- `email` (VARCHAR(100), UNIQUE)
- `password_hash` (VARCHAR(256), NOT NULL)
- `department` (VARCHAR(50))
- `role` (VARCHAR(50), DEFAULT 'employee')
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)
- `deleted_at` (TIMESTAMP, soft delete)

### Leave Types Table
- `id` (SERIAL PRIMARY KEY)
- `name` (VARCHAR(50), NOT NULL)
- `max_days` (INT, NOT NULL)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)
- `deleted_at` (TIMESTAMP, soft delete)

### Leaves Table
- `id` (SERIAL PRIMARY KEY)
- `employee_id` (INT, FOREIGN KEY → employees.id)
- `leave_type_id` (INT, FOREIGN KEY → leave_types.id)
- `start_date` (DATE, NOT NULL)
- `end_date` (DATE, NOT NULL)
- `reason` (TEXT)
- `status` (VARCHAR(20), DEFAULT 'Pending')
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)
- `deleted_at` (TIMESTAMP, soft delete)

## Business Rules

1. **Leave Balance Validation**: Employees cannot apply for more days than their available balance.
2. **Overlapping Leaves**: The system prevents overlapping leave requests (for approved or pending leaves).
3. **Past Dates**: Employees cannot apply for leave with start dates in the past.
4. **Date Range**: Start date must be before or equal to end date.
5. **Leave Status**: Only pending leaves can be approved or rejected.

## Testing

Run unit tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

**Note:** Tests require a test database. Update the `DB_NAME` in your test configuration to use a separate test database.

## Project Structure

```
hrms-api/
├── config/          # Configuration management
├── database/        # Database connection and migrations
├── handlers/        # HTTP request handlers
├── middleware/      # Authentication and authorization middleware
├── models/          # Database models
├── routes/          # Route definitions
├── utils/           # Utility functions (JWT, validation)
├── main.go          # Application entry point
├── go.mod           # Go module file
├── docker-compose.yml # PostgreSQL container setup
└── README.md        # This file
```

## Security Features

- **Password Hashing**: Passwords are hashed using bcrypt before storage
- **JWT Authentication**: Secure token-based authentication
- **Role-Based Access Control**: Different endpoints accessible based on user role
- **Input Validation**: Request validation using go-playground/validator
- **SQL Injection Protection**: GORM provides parameterized queries

## Error Handling

The API returns appropriate HTTP status codes:

- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Missing or invalid authentication token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource conflict (e.g., duplicate NRC, overlapping leaves)
- `500 Internal Server Error`: Server error

## Example Usage

### 1. Register a new employee
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nrc": "123456/78/9",
    "firstname": "John",
    "lastname": "Doe",
    "email": "john@example.com",
    "password": "password123",
    "department": "IT",
    "role": "employee"
  }'
```

### 2. Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "nrc": "123456/78/9",
    "password": "password123"
  }'
```

### 3. Apply for leave (using token from login)
```bash
curl -X POST http://localhost:8080/api/leaves \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "leave_type_id": 1,
    "start_date": "2024-02-01",
    "end_date": "2024-02-05",
    "reason": "Vacation"
  }'
```

## Future Enhancements

- Email notifications on leave approval/rejection
- Reports dashboard for HR and management
- Audit trail for leave actions
- Leave cancellation by employees
- Bulk leave operations
- Leave calendar view
- Integration with external calendar systems

## License

This project is part of an HRMS system. All rights reserved.

## Support

For issues and questions, please contact the development team or create an issue in the repository.

