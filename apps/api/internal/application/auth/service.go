package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/novudesk/novudesk/config"
	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/domain/role"
	"github.com/novudesk/novudesk/internal/domain/team"
	"github.com/novudesk/novudesk/internal/domain/user"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

// applyOverrides merges role permission keys with per-member overrides.
func applyOverrides(roleKeys []string, overrides []user.PermissionOverride) []string {
	set := make(map[string]struct{}, len(roleKeys))
	for _, k := range roleKeys {
		set[k] = struct{}{}
	}
	for _, o := range overrides {
		if o.IsGranted {
			set[o.PermissionKey] = struct{}{}
		} else {
			delete(set, o.PermissionKey)
		}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

// Claims are embedded in the JWT access token.
type Claims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"uid"`
	OrgID       string   `json:"oid"`
	RoleName    string   `json:"role"`
	Permissions []string `json:"perms"`
	TeamIDs     []string `json:"team_ids"`
}

type Service struct {
	users    user.Repository
	roles    role.Repository
	orgs     organization.Repository
	teams    team.Repository
	cfg      config.JWTConfig
	privKey  *rsa.PrivateKey
	pubKey   *rsa.PublicKey
}

func NewService(users user.Repository, roles role.Repository, cfg config.JWTConfig) (*Service, error) {
	privPEM, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(privPEM)
	var privKey *rsa.PrivateKey
	if key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
		var ok bool
		privKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parse private key: not an RSA key")
		}
	} else if key, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
		privKey = key
	} else {
		return nil, fmt.Errorf("parse private key: %w", err2)
	}

	return &Service{
		users:   users,
		roles:   roles,
		cfg:     cfg,
		privKey: privKey,
		pubKey:  &privKey.PublicKey,
	}, nil
}

// WithOrgRepo attaches an org repository used to resolve slugs during login.
func (s *Service) WithOrgRepo(orgs organization.Repository) *Service {
	s.orgs = orgs
	return s
}

// WithTeamRepo attaches a team repository used to load team memberships into JWT.
func (s *Service) WithTeamRepo(teams team.Repository) *Service {
	s.teams = teams
	return s
}

// Login authenticates a user and returns an access token + refresh token.
// orgSlug is the organization's slug (URL-friendly identifier).
func (s *Service) Login(ctx context.Context, email, password, orgSlug string) (accessToken, refreshToken string, err error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return "", "", apperrors.Internal(err)
	}
	if u == nil || u.PasswordHash == nil {
		return "", "", apperrors.Unauthorized("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return "", "", apperrors.Unauthorized("invalid credentials")
	}

	// Resolve slug → org ID.
	orgID := orgSlug
	if s.orgs != nil {
		org, err := s.orgs.FindBySlug(ctx, orgSlug)
		if err != nil || org == nil {
			return "", "", apperrors.Unauthorized("organization not found")
		}
		orgID = org.ID
	}

	member, err := s.users.GetMember(ctx, u.ID, orgID)
	if err != nil || member == nil || !member.IsActive {
		return "", "", apperrors.Forbidden("user is not a member of this organization")
	}

	perms, err := s.roles.GetPermissionKeys(ctx, member.RoleID)
	if err != nil {
		return "", "", apperrors.Internal(err)
	}

	// Apply per-member permission overrides on top of role permissions.
	memberID, err := s.users.GetMemberID(ctx, u.ID, orgID)
	if err == nil && memberID != "" {
		overrides, oErr := s.users.GetMemberPermissionOverrides(ctx, memberID)
		if oErr == nil && len(overrides) > 0 {
			perms = applyOverrides(perms, overrides)
		}
	}

	var teamIDs []string
	if s.teams != nil {
		teamIDs, _ = s.teams.ListTeamIDsByUser(ctx, u.ID, orgID)
	}
	if teamIDs == nil {
		teamIDs = []string{}
	}

	accessToken, err = s.issueAccessToken(u.ID, orgID, member.RoleName, perms, teamIDs)
	if err != nil {
		return "", "", apperrors.Internal(err)
	}

	refreshToken, err = s.issueRefreshToken()
	if err != nil {
		return "", "", apperrors.Internal(err)
	}

	return accessToken, refreshToken, nil
}

// ValidateAccessToken parses and validates a JWT, returning its claims.
func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.pubKey, nil
	})
	if err != nil {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid or expired token", 401)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apperrors.New(apperrors.CodeTokenInvalid, "invalid token claims", 401)
	}

	return claims, nil
}

// HashPassword returns a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

// HashToken returns a SHA-256 hex hash — used for refresh tokens and invite tokens.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateSecureToken creates a cryptographically random hex token.
func GenerateSecureToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) issueAccessToken(userID, orgID, roleName string, perms, teamIDs []string) (string, error) {
	now := time.Now()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
			ID:        uuid.NewString(),
		},
		UserID:      userID,
		OrgID:       orgID,
		RoleName:    roleName,
		Permissions: perms,
		TeamIDs:     teamIDs,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privKey)
}

func (s *Service) issueRefreshToken() (string, error) {
	return GenerateSecureToken(32)
}
