package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/donation-station/donation-station/internal/database"
	"github.com/donation-station/donation-station/internal/newapi"
	"github.com/donation-station/donation-station/internal/session"
	"github.com/gin-gonic/gin"
)

const currentUserContextKey = "current_user"

type userAuthenticator interface {
	Authenticate(ctx context.Context, username, password string) (*newapi.User, error)
}

type appUserStore interface {
	UpsertAppUser(user *database.AppUser) (*database.AppUser, error)
}

type userCredentialStore interface {
	ListUserCredentials(userID int64, limit, offset int) ([]*database.Credential, int, error)
	RemoveUserCredential(userID, credentialID int64) (bool, error)
}

type userLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *Server) userAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.sessionManager == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "登录系统未初始化"})
			return
		}

		tokenCookie, err := c.Request.Cookie(s.sessionManager.CookieName())
		if err != nil || strings.TrimSpace(tokenCookie.Value) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			return
		}

		claims, err := s.sessionManager.Parse(tokenCookie.Value)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "登录状态已失效，请重新登录"})
			return
		}

		c.Set(currentUserContextKey, claims)
		c.Next()
	}
}

func currentUserFromContext(c *gin.Context) (*session.Claims, bool) {
	value, exists := c.Get(currentUserContextKey)
	if !exists {
		return nil, false
	}

	claims, ok := value.(*session.Claims)
	return claims, ok
}

func (s *Server) userLoginHandler(c *gin.Context) {
	var req userLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入用户名和密码"})
		return
	}
	if s.userAuthenticator == nil || s.appUserStore == nil || s.sessionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录系统未初始化"})
		return
	}

	authUser, err := s.userAuthenticator.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	localUser, err := s.appUserStore.UpsertAppUser(&database.AppUser{
		NewAPIUserID: authUser.ID,
		Username:     authUser.Username,
		Email:        authUser.Email,
		DisplayName:  authUser.DisplayName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存登录用户失败"})
		return
	}

	token, err := s.sessionManager.Issue(&session.Claims{
		LocalUserID:  localUser.ID,
		NewAPIUserID: localUser.NewAPIUserID,
		Username:     localUser.Username,
		Email:        localUser.Email,
		DisplayName:  localUser.DisplayName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建登录状态失败"})
		return
	}

	http.SetCookie(c.Writer, s.sessionManager.BuildCookie(token, isSecureRequest(c)))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":             localUser.ID,
			"newapi_user_id": localUser.NewAPIUserID,
			"username":       localUser.Username,
			"email":          localUser.Email,
			"display_name":   localUser.DisplayName,
		},
	})
}

func (s *Server) userLogoutHandler(c *gin.Context) {
	if s.sessionManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录系统未初始化"})
		return
	}

	http.SetCookie(c.Writer, s.sessionManager.BuildExpiredCookie(isSecureRequest(c)))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) userMeHandler(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             user.LocalUserID,
		"newapi_user_id": user.NewAPIUserID,
		"username":       user.Username,
		"email":          user.Email,
		"display_name":   user.DisplayName,
	})
}

func (s *Server) listMyCredentialsHandler(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}
	if s.userCredentialStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户凭证服务未初始化"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	credentials, total, err := s.userCredentialStore.ListUserCredentials(user.LocalUserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取我的凭证失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   credentials,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) removeMyCredentialHandler(c *gin.Context) {
	user, ok := currentUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}
	if s.userCredentialStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户凭证服务未初始化"})
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的凭证ID"})
		return
	}

	removed, err := s.userCredentialStore.RemoveUserCredential(user.LocalUserID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除凭证失败"})
		return
	}
	if !removed {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到可移除的凭证"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "已从本站列表移除"})
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}
