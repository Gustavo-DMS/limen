// Package magiclink provides passwordless magic-link authentication for Limen.
package magiclink

import (
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type magicLinkPlugin struct {
	core               *limen.LimenCore
	config             *config
	userSchema         *limen.UserSchema
	verificationSchema *limen.VerificationSchema
	dbAction           *limen.DatabaseActionHelper
}

// New creates a magic-link plugin with sensible passwordless-auth defaults.
func New(opts ...ConfigOption) *magicLinkPlugin {
	cfg := &config{
		tokenExpiration:   15 * time.Minute,
		autoCreateUser:    true,
		maxUses:           1,
		markEmailVerified: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &magicLinkPlugin{config: cfg}
}

func (p *magicLinkPlugin) Name() limen.PluginName {
	return limen.PluginMagicLink
}

func (p *magicLinkPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	p.userSchema = core.Schema.User
	p.verificationSchema = core.Schema.Verification
	p.dbAction = core.DBAction

	if p.config == nil {
		return fmt.Errorf("magic-link: config is required")
	}
	if p.config.tokenExpiration <= 0 {
		return fmt.Errorf("magic-link: token expiration must be positive")
	}
	if p.config.maxUses <= 0 {
		return fmt.Errorf("magic-link: max uses must be greater than zero")
	}

	return nil
}

func (p *magicLinkPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: "/magic-link",
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("/signin", 5, time.Minute),
			limen.NewRateLimitRule("/verify", 10, time.Minute),
		},
	}
}

func (p *magicLinkPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newMagicLinkHandlers(p, httpCore)
	routeBuilder.POST("/signin", "magic-link-request", handlers.RequestMagicLink)
	routeBuilder.GET("/verify", "magic-link-verify", handlers.VerifyMagicLink)
}

func (p *magicLinkPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	return []limen.SchemaIntrospector{}
}
