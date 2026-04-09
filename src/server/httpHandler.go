package singal

import (
	// "fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const (
	CODE_SUCCESS          = int(0)
	CODE_ERROR            = int(400)
	CODE_INVALID_PARAM    = int(401)
	CODE_FORBIDDEN_PARAM  = int(403)
	CODE_REPEAT_PARAM     = int(404)
	CODE_SERVERBUSY_PARAM = int(501)
)

var errMaps map[int]string

type HttpHandler struct {
}

var gHttpHandler *HttpHandler

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Username string `json:"username" binding:"required,min=2,max=50"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func validateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func NewHttpHandler(router *gin.Engine) (gHttpHandler *HttpHandler) {
	gHttpHandler = &HttpHandler{}
	// api
	router.POST("/echo", gHttpHandler.EchoTest)
	router.POST("/register", gHttpHandler.Register)
	router.POST("/verify", gHttpHandler.VerifyEmail)
	router.POST("/completeRegistration", gHttpHandler.CompleteRegistration)
	router.POST("/login", gHttpHandler.Login)

	errMaps = make(map[int]string)
	errMaps[CODE_SUCCESS] = "success"
	errMaps[CODE_INVALID_PARAM] = "invalid param"
	errMaps[CODE_FORBIDDEN_PARAM] = "forbidden param"
	errMaps[CODE_REPEAT_PARAM] = "repeat param"
	errMaps[CODE_SERVERBUSY_PARAM] = "server busy"

	return
}

func (hh *HttpHandler) EchoTest(c *gin.Context) {
	logger.Debugf("[EchoTest] c.Request.Method: %v", c.Request.Method)
	logger.Debugf("[EchoTest] c.Request.ContentType: %v", c.ContentType())

	c.Request.ParseForm()
	logger.Debugf("[EchoTest] c.Request.Form: %v", c.Request.PostForm)

	for k, v := range c.Request.PostForm {
		logger.Debugf("[EchoTest] k:%v\n", k)
		logger.Debugf("[EchoTest] v:%v\n", v)
	}

	logger.Debugf("[EchoTest] c.Request.ContentLength: %v", c.Request.ContentLength)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "echotest", "traceId": "mwkjt"})
}

func (hh *HttpHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid request parameters",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validateEmail(email) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid email format",
		})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Password must be at least 6 characters",
		})
		return
	}

	db := GetAuthDB()
	var existingUser AuthUser
	if err := db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_REPEAT_PARAM,
			Message: "Email already registered",
		})
		return
	}

	emailService := GetEmailService()
	if emailService == nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Email service not available",
		})
		return
	}

	if err := emailService.SendVerificationCode(email); err != nil {
		logger.Errorf("Failed to send verification code: %v", err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Failed to send verification code",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    CODE_SUCCESS,
		Message: "Verification code sent to email",
	})
}

func (hh *HttpHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid request parameters",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validateEmail(email) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid email format",
		})
		return
	}

	emailService := GetEmailService()
	if emailService == nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Email service not available",
		})
		return
	}

	if !emailService.VerifyCode(email, req.Code) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid or expired verification code",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    CODE_SUCCESS,
		Message: "Email verified successfully",
	})
}

func (hh *HttpHandler) CompleteRegistration(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Code     string `json:"code" binding:"required,len=6"`
		Password string `json:"password" binding:"required,min=6"`
		Username string `json:"username" binding:"required,min=2,max=50"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid request parameters",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validateEmail(email) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid email format",
		})
		return
	}

	emailService := GetEmailService()
	if emailService == nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Email service not available",
		})
		return
	}

	if !emailService.VerifyCode(email, req.Code) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid or expired verification code",
		})
		return
	}

	db := GetAuthDB()
	var existingUser AuthUser
	if err := db.Where("email = ?", email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_REPEAT_PARAM,
			Message: "Email already registered",
		})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Failed to process password",
		})
		return
	}

	user := AuthUser{
		Email:    email,
		Password: string(hashedPassword),
		Username: strings.TrimSpace(req.Username),
		IsActive: true,
	}

	if err := db.Create(&user).Error; err != nil {
		logger.Errorf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, AuthResponse{
			Code:    CODE_SERVERBUSY_PARAM,
			Message: "Failed to create user",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    CODE_SUCCESS,
		Message: "User registered successfully",
		Data: gin.H{
			"user_id":  user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}

func (hh *HttpHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid request parameters",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !validateEmail(email) {
		c.JSON(http.StatusBadRequest, AuthResponse{
			Code:    CODE_INVALID_PARAM,
			Message: "Invalid email format",
		})
		return
	}

	db := GetAuthDB()
	var user AuthUser
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Code:    CODE_ERROR,
			Message: "Invalid email or password",
		})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, AuthResponse{
			Code:    CODE_FORBIDDEN_PARAM,
			Message: "Account not activated",
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, AuthResponse{
			Code:    CODE_ERROR,
			Message: "Invalid email or password",
		})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		Code:    CODE_SUCCESS,
		Message: "Login successful",
		Data: gin.H{
			"user_id":  user.ID,
			"email":    user.Email,
			"username": user.Username,
		},
	})
}
