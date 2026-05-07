package magiclink

import (
	"time"
)

type config struct {
	tokenExpiration   time.Duration
	generateToken     func(email string) (string, error)
	sendMagicLink     func(MagicLinkMessage)
	autoCreateUser    bool
	maxUses           int
	markEmailVerified bool
}

// ConfigOption configures the magic-link plugin.
type ConfigOption func(*config)

// WithTokenExpiration sets how long generated magic links remain valid.
func WithTokenExpiration(expiration time.Duration) ConfigOption {
	return func(c *config) {
		c.tokenExpiration = expiration
	}
}

// WithGenerateToken sets the token generator used for new magic links.
func WithGenerateToken(generateToken func(email string) (string, error)) ConfigOption {
	return func(c *config) {
		c.generateToken = generateToken
	}
}

// WithSendMagicLink sets the callback used to deliver a magic link.
func WithSendMagicLink(sendMagicLink func(MagicLinkMessage)) ConfigOption {
	return func(c *config) {
		c.sendMagicLink = sendMagicLink
	}
}

// WithAutoCreateUser controls whether unknown emails create a user by default.
func WithAutoCreateUser(autoCreateUser bool) ConfigOption {
	return func(c *config) {
		c.autoCreateUser = autoCreateUser
	}
}

// WithMaxUses sets the default number of times a generated magic link can be used.
func WithMaxUses(maxUses int) ConfigOption {
	return func(c *config) {
		c.maxUses = maxUses
	}
}

// WithMarkEmailVerified controls whether successful magic-link login verifies the email.
func WithMarkEmailVerified(markEmailVerified bool) ConfigOption {
	return func(c *config) {
		c.markEmailVerified = markEmailVerified
	}
}

type RequestMagicLinkOptions struct {
	RedirectURI        string
	NewUserRedirectURI string
	ErrorRedirectURI   string
	AdditionalData     map[string]any
}

type MagicLinkMessage struct {
	Email          string
	Token          string
	URL            string
	AdditionalData map[string]any
}
