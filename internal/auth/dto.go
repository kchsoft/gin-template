package auth

type SignupRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=20"`
	Email       string `json:"email" binding:"required,email,max=50"`
	PhoneNumber string `json:"phoneNumber" binding:"required,phone"`
	Password    string `json:"password" binding:"required,min=8,max=15"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=15"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
