package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"vps-panel/internal/database"
	"vps-panel/internal/middleware"
	"vps-panel/internal/models"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type LoginResponse struct {
	Token        string `json:"token"`
	User         *UserResponse `json:"user"`
	Requires2FA  bool   `json:"requires_2fa,omitempty"`
}

type UserResponse struct {
	ID               uint   `json:"id"`
	Username         string `json:"username"`
	Email            string `json:"email"`
	Role             string `json:"role"`
	TwoFactorEnabled bool   `json:"two_factor_enabled"`
}

func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var user models.User
	result := database.DB.Where("username = ?", req.Username).First(&user)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if !user.CheckPassword(req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Check 2FA
	if user.TwoFactorEnabled {
		if req.TOTPCode == "" {
			return c.Status(fiber.StatusOK).JSON(LoginResponse{
				Requires2FA: true,
			})
		}

		valid := totp.Validate(req.TOTPCode, user.TwoFactorSecret)
		if !valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid 2FA code",
			})
		}
	}

	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// Set cookie
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		MaxAge:   86400,
		Path:     "/",
	})

	return c.JSON(LoginResponse{
		Token: token,
		User: &UserResponse{
			ID:               user.ID,
			Username:         user.Username,
			Email:            user.Email,
			Role:             user.Role,
			TwoFactorEnabled: user.TwoFactorEnabled,
		},
	})
}

func Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

func GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(UserResponse{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Email,
		Role:             user.Role,
		TwoFactorEnabled: user.TwoFactorEnabled,
	})
}

type Setup2FAResponse struct {
	Secret string `json:"secret"`
	QRCode string `json:"qr_code"`
}

func Setup2FA(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "VPS Panel",
		AccountName: user.Username,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate 2FA secret",
		})
	}

	// Save secret temporarily (user needs to verify before enabling)
	user.TwoFactorSecret = key.Secret()
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save 2FA secret",
		})
	}

	return c.JSON(Setup2FAResponse{
		Secret: key.Secret(),
		QRCode: key.URL(),
	})
}

type Verify2FARequest struct {
	Code string `json:"code"`
}

func Verify2FA(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var req Verify2FARequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.TwoFactorSecret == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "2FA not set up",
		})
	}

	valid := totp.Validate(req.Code, user.TwoFactorSecret)
	if !valid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid 2FA code",
		})
	}

	user.TwoFactorEnabled = true
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to enable 2FA",
		})
	}

	return c.JSON(fiber.Map{
		"message": "2FA enabled successfully",
	})
}

func Disable2FA(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	user.TwoFactorEnabled = false
	user.TwoFactorSecret = ""
	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable 2FA",
		})
	}

	return c.JSON(fiber.Map{
		"message": "2FA disabled successfully",
	})
}
