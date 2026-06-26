package identity

import "github.com/google/uuid"

type CreateUserRequest struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
	Status   string `json:"status"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GetAllUserResponse struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	CreateAt int64  `json:"create_at"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Role     string `json:"role"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type UpdateUserRequest struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	FullName string    `json:"full_name"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
}

type UpdateUserPasswordRequest struct {
	Email           string `json:"-"`
	OldPasswrd      string `json:"old_passwrd"`
	NewPasswrd      string `json:"new_passwrd"`
	ConfirmPassword string `json:"confirm_password"`
}
