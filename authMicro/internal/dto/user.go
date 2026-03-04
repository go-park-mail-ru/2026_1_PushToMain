package dto

type SignUpRequest struct {
	Name           string `json:"name"`
	Surname        string `json:"surname"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	PasswordRepeat string `json:"passwordRepeat"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}
