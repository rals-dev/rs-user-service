package services

import (
	"context"
	"errors"
	"testing"
	"user-service/constants"
	errConstant "user-service/constants/error"
	"user-service/domain/dto"
	"user-service/domain/models"
	"user-service/repositories"
	repoUser "user-service/repositories/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// fakeUserRepository is a hand-written test double for repoUser.IUserRepository.
// The project has no mocking library wired up with a full, buildable go.sum
// entry (only transitive go.mod-only hashes for testify/gomock are present),
// so tests use plain function-field fakes instead.
type fakeUserRepository struct {
	findByUUID     func(ctx context.Context, id string) (*models.User, error)
	findByUsername func(ctx context.Context, username string) (*models.User, error)
	findByEmail    func(ctx context.Context, email string) (*models.User, error)
	update         func(ctx context.Context, req *dto.UpdateRequest, id string) (*models.User, error)
	register       func(ctx context.Context, req *dto.RegisterRequest) (*models.User, error)

	findByUsernameCalls int
	findByEmailCalls    int
	updateCalls         int
}

func (f *fakeUserRepository) Register(ctx context.Context, req *dto.RegisterRequest) (*models.User, error) {
	if f.register != nil {
		return f.register(ctx, req)
	}
	return nil, errors.New("Register not stubbed")
}

func (f *fakeUserRepository) Update(ctx context.Context, req *dto.UpdateRequest, id string) (*models.User, error) {
	f.updateCalls++
	if f.update != nil {
		return f.update(ctx, req, id)
	}
	return nil, errors.New("Update not stubbed")
}

func (f *fakeUserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	f.findByUsernameCalls++
	if f.findByUsername != nil {
		return f.findByUsername(ctx, username)
	}
	return nil, errConstant.ErrUserNotFound
}

func (f *fakeUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	f.findByEmailCalls++
	if f.findByEmail != nil {
		return f.findByEmail(ctx, email)
	}
	return nil, errConstant.ErrUserNotFound
}

func (f *fakeUserRepository) FindByUUID(ctx context.Context, id string) (*models.User, error) {
	if f.findByUUID != nil {
		return f.findByUUID(ctx, id)
	}
	return nil, errConstant.ErrUserNotFound
}

var _ repoUser.IUserRepository = (*fakeUserRepository)(nil)

type fakeRepositoryRegistry struct {
	user repoUser.IUserRepository
}

func (f *fakeRepositoryRegistry) GetUser() repoUser.IUserRepository {
	return f.user
}

var _ repositories.IRepositoryRegistry = (*fakeRepositoryRegistry)(nil)

func newService(repo *fakeUserRepository) IUserService {
	return NewUserService(&fakeRepositoryRegistry{user: repo})
}

func ctxAsUser(u *dto.UserResponse) context.Context {
	return context.WithValue(context.Background(), constants.UserLogin, u)
}

// --- IDOR authorization: GetUserByUUID ---

func TestGetUserByUUID_ForbiddenForOtherCustomer(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()

	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			t.Fatal("repository must not be queried once authorization denies access")
			return nil, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: owner, Role: "customer"})
	_, err := svc.GetUserByUUID(ctx, other.String())
	if !errors.Is(err, errConstant.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetUserByUUID_AllowedForOwner(t *testing.T) {
	owner := uuid.New()
	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			return &models.User{UUID: owner, Username: "self", Email: "self@example.com"}, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: owner, Role: "customer"})
	resp, err := svc.GetUserByUUID(ctx, owner.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UUID != owner {
		t.Fatalf("expected uuid %s, got %s", owner, resp.UUID)
	}
}

func TestGetUserByUUID_AllowedForAdminOnAnyUser(t *testing.T) {
	admin := uuid.New()
	target := uuid.New()
	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			if id != target.String() {
				t.Fatalf("expected lookup for target uuid %s, got %s", target, id)
			}
			return &models.User{UUID: target, Username: "someone-else"}, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: admin, Role: "admin"})
	resp, err := svc.GetUserByUUID(ctx, target.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UUID != target {
		t.Fatalf("expected uuid %s, got %s", target, resp.UUID)
	}
}

func TestGetUserByUUID_InvalidUUIDFormatIsRejected(t *testing.T) {
	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			t.Fatal("repository must not be queried for a malformed uuid")
			return nil, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: uuid.New(), Role: "customer"})
	_, err := svc.GetUserByUUID(ctx, "not-a-uuid")
	if !errors.Is(err, errConstant.ErrRequestValidation) {
		t.Fatalf("expected ErrRequestValidation, got %v", err)
	}
}

func TestGetUserByUUID_UnauthenticatedContextIsRejected(t *testing.T) {
	repo := &fakeUserRepository{}
	svc := newService(repo)

	_, err := svc.GetUserByUUID(context.Background(), uuid.New().String())
	if !errors.Is(err, errConstant.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// --- IDOR authorization: Update ---

func TestUpdate_ForbiddenForOtherCustomer(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()

	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			t.Fatal("repository must not be queried once authorization denies access")
			return nil, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: owner, Role: "customer"})
	_, err := svc.Update(ctx, &dto.UpdateRequest{Name: "x", Username: "x", Email: "x@example.com", PhoneNumber: "0800"}, other.String())
	if !errors.Is(err, errConstant.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected Update to never be called, got %d calls", repo.updateCalls)
	}
}

func TestUpdate_AllowedForAdminOnAnyUser(t *testing.T) {
	admin := uuid.New()
	target := uuid.New()

	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			return &models.User{UUID: target, Username: "original", Email: "original@example.com"}, nil
		},
		findByUsername: func(ctx context.Context, username string) (*models.User, error) {
			return nil, errConstant.ErrUserNotFound
		},
		findByEmail: func(ctx context.Context, email string) (*models.User, error) {
			return nil, errConstant.ErrUserNotFound
		},
		update: func(ctx context.Context, req *dto.UpdateRequest, id string) (*models.User, error) {
			return &models.User{UUID: target, Username: req.Username, Email: req.Email}, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: admin, Role: "admin"})
	_, err := svc.Update(ctx, &dto.UpdateRequest{Name: "New", Username: "newname", Email: "new@example.com", PhoneNumber: "0800"}, target.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected exactly one Update call, got %d", repo.updateCalls)
	}
}

// TestUpdate_PasswordWithoutConfirmPasswordDoesNotPanic locks in the fix for a
// nil-pointer dereference regression: the old code always evaluated
// *req.ConfirmPassword whenever req.Password was set, which panicked (500) if
// the client sent "password" without "confirmPassword".
func TestUpdate_PasswordWithoutConfirmPasswordDoesNotPanic(t *testing.T) {
	self := uuid.New()
	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			return &models.User{UUID: self, Username: "self", Email: "self@example.com"}, nil
		},
	}
	svc := newService(repo)

	password := "newsecret1"
	ctx := ctxAsUser(&dto.UserResponse{UUID: self, Role: "customer"})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Update panicked instead of returning an error: %v", r)
		}
	}()

	_, err := svc.Update(ctx, &dto.UpdateRequest{
		Name: "self", Username: "self", Email: "self@example.com", PhoneNumber: "0800",
		Password: &password, ConfirmPassword: nil,
	}, self.String())
	if !errors.Is(err, errConstant.ErrPasswordDoesNotMatch) {
		t.Fatalf("expected ErrPasswordDoesNotMatch, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected Update to never be called when confirmation is missing, got %d calls", repo.updateCalls)
	}
}

// TestUpdate_DuplicateEmailReturnsEmailError locks in a bug fix: the previous
// code returned ErrUserNameExist for a duplicate *email*, which is both wrong
// and confusing to API clients.
func TestUpdate_DuplicateEmailReturnsEmailError(t *testing.T) {
	self := uuid.New()
	otherOwner := uuid.New()

	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			return &models.User{UUID: self, Username: "self", Email: "self@example.com"}, nil
		},
		findByEmail: func(ctx context.Context, email string) (*models.User, error) {
			return &models.User{UUID: otherOwner, Username: "someone-else", Email: email}, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: self, Role: "customer"})
	_, err := svc.Update(ctx, &dto.UpdateRequest{
		Name: "self", Username: "self", Email: "taken@example.com", PhoneNumber: "0800",
	}, self.String())
	if !errors.Is(err, errConstant.ErrEmailAlreadyExist) {
		t.Fatalf("expected ErrEmailAlreadyExist, got %v", err)
	}
}

// TestUpdate_NoDuplicateLookupWhenFieldsUnchanged locks in a fix for a
// duplicate-query bug: previously, checking whether a username/email was
// "taken by someone else" always issued a second, redundant FindByUsername/
// FindByEmail call even when the value hadn't changed from the current user's own.
func TestUpdate_NoDuplicateLookupWhenFieldsUnchanged(t *testing.T) {
	self := uuid.New()
	repo := &fakeUserRepository{
		findByUUID: func(ctx context.Context, id string) (*models.User, error) {
			return &models.User{UUID: self, Username: "self", Email: "self@example.com"}, nil
		},
		update: func(ctx context.Context, req *dto.UpdateRequest, id string) (*models.User, error) {
			return &models.User{UUID: self, Username: req.Username, Email: req.Email}, nil
		},
	}
	svc := newService(repo)

	ctx := ctxAsUser(&dto.UserResponse{UUID: self, Role: "customer"})
	_, err := svc.Update(ctx, &dto.UpdateRequest{
		Name: "self", Username: "self", Email: "self@example.com", PhoneNumber: "0800",
	}, self.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.findByUsernameCalls != 0 {
		t.Fatalf("expected no FindByUsername lookup when username is unchanged, got %d calls", repo.findByUsernameCalls)
	}
	if repo.findByEmailCalls != 0 {
		t.Fatalf("expected no FindByEmail lookup when email is unchanged, got %d calls", repo.findByEmailCalls)
	}
}

// --- Login: enumeration-safe errors ---

func TestLogin_UnknownUsernameAndWrongPasswordReturnSameError(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password for fixture: %v", err)
	}

	repoUnknownUser := &fakeUserRepository{
		findByUsername: func(ctx context.Context, username string) (*models.User, error) {
			return nil, errConstant.ErrUserNotFound
		},
	}
	repoWrongPassword := &fakeUserRepository{
		findByUsername: func(ctx context.Context, username string) (*models.User, error) {
			return &models.User{Username: username, Password: string(hashed)}, nil
		},
	}

	_, errUnknownUser := newService(repoUnknownUser).Login(context.Background(), &dto.LoginRequest{Username: "ghost", Password: "whatever"})
	_, errWrongPassword := newService(repoWrongPassword).Login(context.Background(), &dto.LoginRequest{Username: "real-user", Password: "not-the-password"})

	if !errors.Is(errUnknownUser, errConstant.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown username, got %v", errUnknownUser)
	}
	if !errors.Is(errWrongPassword, errConstant.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got %v", errWrongPassword)
	}
	if errUnknownUser.Error() != errWrongPassword.Error() {
		t.Fatalf("expected identical error messages to avoid user enumeration, got %q vs %q", errUnknownUser.Error(), errWrongPassword.Error())
	}
}
