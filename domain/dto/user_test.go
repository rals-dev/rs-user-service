package dto

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
)

func validRegisterRequest() RegisterRequest {
	return RegisterRequest{
		Name:            "Test",
		Username:        "testuser",
		Password:        "supersecret",
		ConfirmPassword: "supersecret",
		Email:           "test@example.com",
		PhoneNumber:     "0812345678",
	}
}

func TestRegisterRequest_Valid(t *testing.T) {
	v := validator.New()
	req := validRegisterRequest()
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestRegisterRequest_PasswordConfirmationMustMatch(t *testing.T) {
	v := validator.New()
	req := validRegisterRequest()
	req.ConfirmPassword = "different"
	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation error when confirmPassword does not match password")
	}
}

func TestRegisterRequest_PasswordMinLength(t *testing.T) {
	v := validator.New()
	req := validRegisterRequest()
	req.Password = "short"
	req.ConfirmPassword = "short"
	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation error for password shorter than 8 characters")
	}
}

// TestRegisterRequest_RoleIDNotBoundFromJSON locks in that RoleID can never be
// set by client-supplied JSON (json:"-"): registrations always get the
// customer role, forced by the service layer, never by the request body.
func TestRegisterRequest_RoleIDNotBoundFromJSON(t *testing.T) {
	data := []byte(`{
		"name":"Test","username":"testuser","password":"supersecret",
		"confirmPassword":"supersecret","email":"test@example.com",
		"phoneNumber":"0812345678","RoleID":1,"roleId":1
	}`)
	var req RegisterRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if req.RoleID != 0 {
		t.Fatalf("expected RoleID to remain zero-value when supplied by client JSON, got %d", req.RoleID)
	}
}

func validUpdateRequest() UpdateRequest {
	return UpdateRequest{
		Name:        "Test",
		Username:    "testuser",
		Email:       "test@example.com",
		PhoneNumber: "0812345678",
	}
}

func TestUpdateRequest_PasswordOptional(t *testing.T) {
	v := validator.New()
	req := validUpdateRequest()
	if err := v.Struct(req); err != nil {
		t.Fatalf("expected no validation error when password is omitted, got %v", err)
	}
}

// TestUpdateRequest_ConfirmPasswordRequiredWhenPasswordSet guards the nil-pointer
// panic regression in services/user/user.go Update(): sending "password"
// without "confirmPassword" must be rejected by validation before it ever
// reaches the service's *req.ConfirmPassword dereference.
func TestUpdateRequest_ConfirmPasswordRequiredWhenPasswordSet(t *testing.T) {
	v := validator.New()
	password := "newsecret1"
	req := validUpdateRequest()
	req.Password = &password

	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation error when confirmPassword is missing but password is set")
	}
}

func TestUpdateRequest_PasswordMinLength(t *testing.T) {
	v := validator.New()
	password := "short"
	confirm := "short"
	req := validUpdateRequest()
	req.Password = &password
	req.ConfirmPassword = &confirm

	if err := v.Struct(req); err == nil {
		t.Fatal("expected validation error for password shorter than 8 characters")
	}
}
