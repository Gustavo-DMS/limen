package magiclink

import (
	"context"

	"github.com/thecodearcher/limen"
)

// API is the public interface for the magic-link plugin.
// Call Use to obtain a type-safe reference from a Limen instance.
type API interface {
	RequestMagicLink(ctx context.Context, email string, opts ...*RequestMagicLinkOptions) (*MagicLinkMessage, error)
	VerifyMagicLink(ctx context.Context, token string) (*limen.AuthenticationResult, *MagicLinkState, error)
}

// Use returns a type-safe API for the magic-link plugin.
// Panics if the plugin was not registered in Config.Plugins.
func Use(a *limen.Limen) API {
	return limen.Use[API](a, limen.PluginMagicLink)
}
