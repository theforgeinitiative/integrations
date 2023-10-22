package api

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

const AuthMethodHeader = "X-Auth-Method"

type AuthMiddleware struct {
	Audience             string
	AllowedGCPPrincipals []string

	jwtValidator *validator.Validator
}

type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func NewAuthMiddleware(issuer, audience string, gcpPrincipals []string) (AuthMiddleware, error) {
	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return AuthMiddleware{}, fmt.Errorf("failed to parse the issuer url: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuer,
		[]string{audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to set up the jwt validator")
	}

	return AuthMiddleware{
		AllowedGCPPrincipals: gcpPrincipals,
		jwtValidator:         jwtValidator,
	}, nil
}

func (a *AuthMiddleware) Require(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		splitToken := strings.Split(c.Request().Header.Get("Authorization"), "Bearer ")
		if len(splitToken) < 2 {
			return echo.NewHTTPError(401)
		}
		reqToken := splitToken[1]

		// Use GCP auth if header is present
		if c.Request().Header.Get(AuthMethodHeader) == "gcp" {
			tok, err := idtoken.Validate(context.Background(), reqToken, a.Audience)
			if err != nil {
				return echo.NewHTTPError(401, "invalid gcp token provided")
			}
			// match against trusted principals
			email, ok := tok.Claims["email"]
			if !ok {
				return echo.NewHTTPError(401, "gcp token did not have email claim")
			}
			for _, p := range a.AllowedGCPPrincipals {
				if p == email {
					c.Logger().Printf("Validated request with GCP credentials")
					c.Set("authorized_user", email)
					return next(c)
				}
			}
			c.Logger().Infof("ID token principal was not authorized: %s", email)
			return echo.NewHTTPError(401)
		}

		// TODO: do some actual permissions checking. For now, just validate token
		claims, err := a.jwtValidator.ValidateToken(c.Request().Context(), reqToken)
		if err != nil {
			c.Logger().Infof("Failed to validate token: %s", err)
			return echo.NewHTTPError(401)
		}
		c.Set("authorized_user", claims.(*validator.ValidatedClaims).RegisteredClaims.Subject)
		return next(c)
	}
}
