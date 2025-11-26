package handlers

import (
	"bytes"
	"encoding/json"
	"hrms-api/config"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) {
	// Load test config
	config.LoadConfig()
	config.AppConfig.DBName = "hrms_test_db"

	// Connect to test database
	if err := database.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Migrate
	if err := database.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Clean up
	database.DB.Exec("TRUNCATE TABLE employees CASCADE")
}

func TestLogin(t *testing.T) {
	setupTestDB(t)
	defer database.DB.Exec("TRUNCATE TABLE employees CASCADE")

	// Create test employee
	hashedPassword, _ := utils.HashPassword("testpass123")
	employee := models.Employee{
		NRC:          "123456/78/9",
		Firstname:    "Test",
		Lastname:     "User",
		Email:        "test@example.com",
		PasswordHash: hashedPassword,
		Role:         models.RoleEmployee,
	}
	database.DB.Create(&employee)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/login", Login)

	// Test successful login
	loginReq := LoginRequest{
		NRC:      "123456/78/9",
		Password: "testpass123",
	}
	jsonValue, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response AuthResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, employee.NRC, response.Employee.NRC)

	// Test invalid credentials
	loginReq.Password = "wrongpassword"
	jsonValue, _ = json.Marshal(loginReq)
	req, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRegister(t *testing.T) {
	setupTestDB(t)
	defer database.DB.Exec("TRUNCATE TABLE employees CASCADE")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/auth/register", Register)

	// Test successful registration
	registerReq := RegisterRequest{
		NRC:        "987654/32/1",
		Firstname:  "New",
		Lastname:   "User",
		Email:      "newuser@example.com",
		Password:   "password123",
		Department: "IT",
		Role:       models.RoleEmployee,
	}
	jsonValue, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var response AuthResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, registerReq.NRC, response.Employee.NRC)

	// Test duplicate NRC
	req, _ = http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}
