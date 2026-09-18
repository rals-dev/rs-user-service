package error

import (
	"errors"
	"testing"
)

func TestErrMapping_KnownErrorsAreMapped(t *testing.T) {
	all := make([]error, 0, len(GeneralErrors)+len(UserErrors))
	all = append(all, GeneralErrors...)
	all = append(all, UserErrors...)

	for _, err := range all {
		if !ErrMapping(err) {
			t.Errorf("expected %q to be recognized by ErrMapping", err.Error())
		}
	}
}

func TestErrMapping_UnknownErrorIsNotMapped(t *testing.T) {
	if ErrMapping(errors.New("some completely unrelated error")) {
		t.Error("expected an unregistered error to not be mapped")
	}
}

// TestErrMapping_DoesNotMutateGeneralErrors guards a regression where
// ErrMapping built its combined error list via
// append(append(GeneralErrors[:], UserErrors[:]...)), which could grow/mutate
// the shared GeneralErrors backing array as a side effect of just checking an
// error.
func TestErrMapping_DoesNotMutateGeneralErrors(t *testing.T) {
	before := make([]error, len(GeneralErrors))
	copy(before, GeneralErrors)

	ErrMapping(ErrUserNotFound)
	ErrMapping(errors.New("noise"))

	if len(GeneralErrors) != len(before) {
		t.Fatalf("GeneralErrors length changed from %d to %d after calling ErrMapping", len(before), len(GeneralErrors))
	}
	for i, err := range GeneralErrors {
		if err.Error() != before[i].Error() {
			t.Fatalf("GeneralErrors[%d] changed from %q to %q after calling ErrMapping", i, before[i].Error(), err.Error())
		}
	}
}

// TestUserErrors_IncludesAllDeclaredUserFacingErrors ensures every user-facing
// sentinel error is registered so response.HttpResponse can surface its real
// message instead of falling back to a generic 500 message.
func TestUserErrors_IncludesAllDeclaredUserFacingErrors(t *testing.T) {
	mustInclude := []error{
		ErrEmailAlreadyExist,
		ErrUserAlreadyExist,
		ErrUserNameExist,
		ErrUserNotFound,
		ErrWrongPassword,
		ErrPasswordDoesNotMatch,
		ErrInvalidToken,
		ErrInvalidCredentials,
	}

	for _, err := range mustInclude {
		found := false
		for _, item := range UserErrors {
			if item.Error() == err.Error() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected UserErrors to include %q", err.Error())
		}
	}
}
