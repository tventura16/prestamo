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

// Verifier valida tokens JWT emitidos por Keycloak.
//
// El servicio se conecta a Keycloak por URL interna (auth-service:8080) pero
// los tokens son emitidos con iss = URL pública (localhost:8080). Usamos
// oidc.InsecureIssuerURLContext para hacer la verificación con el iss público.
type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewVerifier(ctx context.Context, internalURL, publicURL, realm, clientID string) (*Verifier, error) {
	internalIssuer := fmt.Sprintf("%s/realms/%s", strings.TrimSuffix(internalURL, "/"), realm)
	publicIssuer := fmt.Sprintf("%s/realms/%s", strings.TrimSuffix(publicURL, "/"), realm)

	// Saltamos la auto-discovery de oidc.NewProvider (que obtendría la URL
	// de JWKs apuntando al public URL, inalcanzable desde el container).
	// Configuramos manualmente:
	//   - JWKs URL → internalIssuer (auth-service:8080)
	//   - issuer esperado en el token → publicIssuer (localhost:8080)
	jwksURL := internalIssuer + "/protocol/openid-connect/certs"
	keySet := oidc.NewRemoteKeySet(ctx, jwksURL)

	return &Verifier{
		verifier: oidc.NewVerifier(publicIssuer, keySet, &oidc.Config{
			ClientID:          clientID,
			SkipClientIDCheck: true, // Keycloak pone client_id en azp, aud puede ser "account"
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
			UserID:   userID,
			Username: raw.PreferredUsername,
			Email:    raw.Email,
			Roles:    raw.RealmAccess.Roles,
		})
		c.Next()
	}
}

// FromContext extrae claims del request actual.
func FromContext(c *gin.Context) (Claims, bool) {
	v, ok := c.Get(ctxKey)
	if !ok {
		return Claims{}, false
	}
	cl, ok := v.(Claims)
	return cl, ok
}

// RequireRole devuelve un middleware que rechaza si el usuario no tiene
// alguno de los roles requeridos.
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

// GuardRole devuelve un middleware de control de rol nil-safe. Cuando el
// verifier es nil (AUTH_ENABLED=false en desarrollo) actúa como passthrough,
// de modo que declarar guards por endpoint no rompe el modo sin autenticación.
func (v *Verifier) GuardRole(roles ...string) gin.HandlerFunc {
	if v == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return RequireRole(roles...)
}
