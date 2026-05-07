package magiclink

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/thecodearcher/limen"
)

func (p *magicLinkPlugin) RequestMagicLink(ctx context.Context, email string, opts ...*RequestMagicLinkOptions) (*MagicLinkMessage, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, ErrEmailRequired
	}

	token, err := p.generateToken(email)
	if err != nil {
		return nil, err
	}

	var requestOpts *RequestMagicLinkOptions
	if len(opts) > 0 {
		requestOpts = opts[0]
	}

	state := p.newMagicLinkState(email, requestOpts)
	stateValue, err := encodeMagicLinkState(state)
	if err != nil {
		return nil, err
	}

	tokenHash := p.generateTokenHash(token)
	_, err = p.dbAction.CreateVerification(ctx, MagicLinkAction, tokenHash, stateValue, p.config.tokenExpiration)
	if err != nil {
		return nil, err
	}

	message := MagicLinkMessage{
		Email:          email,
		Token:          token,
		URL:            p.buildMagicLinkURL(token),
		AdditionalData: state.AdditionalData,
	}

	if p.config.sendMagicLink != nil {
		p.config.sendMagicLink(message)
	}

	return &message, nil
}

// VerifyMagicLink consumes one allowed use of a magic-link token and returns an auth result.
func (p *magicLinkPlugin) VerifyMagicLink(ctx context.Context, token string) (*limen.AuthenticationResult, *magicLinkState, error) {
	tokenHash := p.generateTokenHash(token)
	verification, err := p.validateToken(ctx, tokenHash)
	if err != nil {
		return nil, nil, err
	}

	result, state, err := p.validateMagicLinkState(ctx, verification)
	if err != nil {
		return nil, state, p.handleVerifyMagicLinkError(ctx, verification, err)
	}

	return result, state, nil
}

func (p *magicLinkPlugin) handleVerifyMagicLinkError(ctx context.Context, verification *limen.Verification, err error) error {
	if !isInvalidOrExpiredMagicLinkError(err) {
		return err
	}
	return p.dbAction.DeleteVerification(ctx, verification.ID)
}

func (p *magicLinkPlugin) validateMagicLinkState(ctx context.Context, verification *limen.Verification) (*limen.AuthenticationResult, *magicLinkState, error) {
	state, err := decodeMagicLinkState(verification.Value)
	if err != nil {
		return nil, nil, err
	}

	if state.UsedCount >= p.config.maxUses {
		return nil, state, ErrMagicLinkTokenMaxUsesExceeded
	}

	existingUser, err := p.resolveUserForVerification(ctx, state.Email)
	if err != nil {
		return nil, state, err
	}
	state.IsNewUser = existingUser == nil

	if err := p.core.WithTransaction(ctx, func(ctx context.Context) error {
		if err := p.consumeMagicLink(ctx, verification, state); err != nil {
			return err
		}

		return p.upsertVerifiedUser(ctx, existingUser, state.Email, state.AdditionalData)
	}); err != nil {
		return nil, state, err
	}

	refreshedUser, err := p.dbAction.FindUserByEmail(ctx, state.Email)
	if err != nil {
		return nil, state, err
	}

	return &limen.AuthenticationResult{User: refreshedUser}, state, nil
}

func (p *magicLinkPlugin) upsertVerifiedUser(ctx context.Context, existingUser *limen.User, email string, additionalData map[string]any) error {
	if existingUser != nil {
		if err := p.markEmailVerified(ctx, existingUser); err != nil {
			return err
		}
		return nil
	}

	now := time.Now()
	user := &limen.User{
		Email:           email,
		EmailVerifiedAt: &now,
	}
	if err := p.dbAction.CreateUser(ctx, user, additionalData); err != nil {
		return err
	}
	return nil
}

func (p *magicLinkPlugin) markEmailVerified(ctx context.Context, user *limen.User) error {
	if !p.config.markEmailVerified || user.EmailVerifiedAt != nil {
		return nil
	}
	now := time.Now()
	return p.dbAction.UpdateUser(ctx, &limen.User{EmailVerifiedAt: &now}, []limen.Where{
		limen.Eq(p.userSchema.GetIDField(), user.ID),
	})
}

func (p *magicLinkPlugin) consumeMagicLink(ctx context.Context, verification *limen.Verification, state *magicLinkState) error {
	state.UsedCount++

	if state.UsedCount >= p.config.maxUses {
		return p.dbAction.DeleteVerification(ctx, verification.ID)
	}

	newValue, err := encodeMagicLinkState(state)
	if err != nil {
		return err
	}

	verification.Value = newValue
	return p.dbAction.UpdateVerification(ctx, verification, []limen.Where{
		limen.Eq(p.verificationSchema.GetIDField(), verification.ID),
	})
}

// resolveUserForVerification returns the existing user or nil if auto-create is enabled and the user is not found.
// if the user is not found and auto-create is disabled, it returns an error.
func (p *magicLinkPlugin) resolveUserForVerification(ctx context.Context, email string) (*limen.User, error) {
	user, err := p.dbAction.FindUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, limen.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, limen.ErrRecordNotFound) && !p.config.autoCreateUser {
		return nil, ErrEmailNotFound
	}

	return user, nil
}

func (p *magicLinkPlugin) buildMagicLinkURL(token string) string {
	callbackURL := p.core.GetBaseURLWithPluginPath(limen.PluginMagicLink, "/verify")
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return ""
	}
	query := parsed.Query()
	query.Set("token", token)

	parsed.RawQuery = query.Encode()
	return parsed.String()
}
