package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewVerifier(ctx context.Context, internalURL, publicURL, realm, clientID string) (*Verifier, error) {
	internalIssuer := fmt.Sprintf("%s/realms/%s", strings.TrimSuffix(internalURL, "/"), realm)
	publicIssuer := fmt.Sprintf("%s/realms/%s", strings.TrimSuffix(publicURL, "/"), realm)

	jwksURL := internalIssuer + "/protocol/openid-connect/certs"
	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)

	return &Verifier{
		verifier: oidc.NewVerifier(publicIssuer, keySet, &oidc.Config{
			ClientID:          clientID,
			SkipClientIDCheck: true,
		}),
	}, nil
}

type Claims struct {
	UserID   uuid.UUID
	Username string
	Email    string
	Roles    []string
}

const ctxKey = "_auth_claims"

func (v *Verifier) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		idToken, err := v.verifier.Verify(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token: " + err.Error()})
			return
		}
		var raw struct {
			Sub               string `json:"sub"`
			PreferredUsername string `json:"preferred_username"`
			Email             string `json:"email"`
			RealmAccess       struct {
				Roles []string `json:"roles"`
			} `json:"realm_access"`
		}
		if err := idToken.Claims(&raw); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		userID, err := uuid.Parse(raw.Sub)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid sub uuid"})
			return
		}
		c.Set(ctxKey, Claims{
			UserID: userID, Username: raw.PreferredUsername, Email: raw.Email,
			Roles: raw.RealmAccess.Roles,
		})
		c.Next()
	}
}

func FromContext(c *gin.Context) (Claims, bool) {
	v, ok := c.Get(ctxKey)
	if !ok {
		return Claims{}, false
	}
	cl, ok := v.(Claims)
	return cl, ok
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := FromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no auth context"})
			return
		}
		for _, need := range roles {
			for _, have := range cl.Roles {
				if have == need {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
	}
}
