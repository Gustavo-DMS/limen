package magiclink

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/thecodearcher/limen"
)

// MagicLinkState is the per-request state persisted alongside a magic-link
// verification token. It captures the redirect targets supplied at request
// time and information resolved during verification (such as whether the
// sign-in was for a newly created user).
type MagicLinkState struct {
	Email              string         `json:"e"`
	UsedCount          int            `json:"c"`
	RedirectURI        string         `json:"r,omitempty"`
	NewUserRedirectURI string         `json:"nr,omitempty"`
	ErrorRedirectURI   string         `json:"er,omitempty"`
	AdditionalData     map[string]any `json:"a,omitempty"`
	Nonce              string         `json:"n,omitempty"`
	IsNewUser          bool           `json:"-"`
}

func (p *magicLinkPlugin) newMagicLinkState(email string, opts *RequestMagicLinkOptions) *MagicLinkState {
	if opts == nil {
		opts = &RequestMagicLinkOptions{}
	}
	state := &MagicLinkState{
		Email:              email,
		UsedCount:          0,
		RedirectURI:        opts.RedirectURI,
		NewUserRedirectURI: opts.NewUserRedirectURI,
		ErrorRedirectURI:   opts.ErrorRedirectURI,
		AdditionalData:     opts.AdditionalData,
		Nonce:              p.generateStateNonce(),
	}

	return state
}

func encodeMagicLinkState(state *MagicLinkState) (string, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeMagicLinkState(value string) (*MagicLinkState, error) {
	var state MagicLinkState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (p *magicLinkPlugin) generateToken(email string) (string, error) {
	if p.config.generateToken != nil {
		return p.config.generateToken(email)
	}
	tokenBytes := make([]byte, 32)
	_, _ = rand.Read(tokenBytes)
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func (p *magicLinkPlugin) generateStateNonce() string {
	nonceBytes := make([]byte, 16)
	_, _ = rand.Read(nonceBytes)
	return base64.RawURLEncoding.EncodeToString(nonceBytes)
}

func (p *magicLinkPlugin) generateTokenHash(token string) string {
	mac := hmac.New(sha256.New, p.core.Secret())
	mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// validateToken validates a token hash and returns the verification record
// or an error if the token is invalid or expired (this does not check if the token has been used more than the max uses).
func (p *magicLinkPlugin) validateToken(ctx context.Context, tokenHash string) (*limen.Verification, error) {
	verification, err := p.dbAction.FindVerificationByAction(ctx, MagicLinkAction, tokenHash)
	if err != nil {
		if errors.Is(err, limen.ErrRecordNotFound) {
			return nil, ErrMagicLinkTokenInvalid
		}
		return nil, err
	}
	if verification.ExpiresAt.Before(time.Now()) {
		if err := p.dbAction.DeleteVerification(ctx, verification.ID); err != nil {
			return nil, err
		}
		return nil, ErrMagicLinkTokenInvalid
	}
	return verification, nil
}
