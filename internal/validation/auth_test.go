package validation

import "testing"

func TestValidateRegistrationValidInput(t *testing.T) {
	input := RegistrationInput{
		Email:    "  Lefteris@Example.COM  ",
		Username: "  lefteris  ",
		Password: "strong-password-123",
	}

	got, err := ValidateRegistration(input)
	if err != nil {
		t.Fatalf("ValidateRegistration() error: %v", err)
	}

	if got.Email != "lefteris@example.com" {
		t.Errorf(
			"Email = %q, want %q",
			got.Email,
			"lefteris@example.com",
		)
	}

	if got.Username != "lefteris" {
		t.Errorf(
			"Username = %q, want %q",
			got.Username,
			"lefteris",
		)
	}

	if got.Password != "strong-password-123" {
		t.Errorf(
			"Password = %q, want original password",
			got.Password,
		)
	}
}
func TestValidateRegistrationRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input RegistrationInput
	}{
		{
			name: "blank email",
			input: RegistrationInput{
				Email:    "",
				Username: "lefteris",
				Password: "strong-password-123",
			},
		},
		{
			name: "whitespace email",
			input: RegistrationInput{
				Email:    "   ",
				Username: "lefteris",
				Password: "strong-password-123",
			},
		},
		{
			name: "malformed email",
			input: RegistrationInput{
				Email:    "not-an-email",
				Username: "lefteris",
				Password: "strong-password-123",
			},
		},
		{
			name: "blank username",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "",
				Password: "strong-password-123",
			},
		},
		{
			name: "whitespace username",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "   ",
				Password: "strong-password-123",
			},
		},
		{
			name: "username too short",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "ab",
				Password: "strong-password-123",
			},
		},
		{
			name: "username too long",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "abcdefghijklmnopqrstuvwxyz1234567",
				Password: "strong-password-123",
			},
		},
		{
			name: "blank password",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "lefteris",
				Password: "",
			},
		},
		{
			name: "password too short",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "lefteris",
				Password: "12345",
			},
		},
		{
			name: "password too long",
			input: RegistrationInput{
				Email:    "lefteris@example.com",
				Username: "lefteris",
				Password: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateRegistration(tt.input)

			if err == nil {
				t.Fatal("ValidateRegistration() error = nil, want error")
			}
		})
	}
}
