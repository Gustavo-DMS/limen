package magiclink

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thecodearcher/limen"
)

func TestRequestMagicLink_RequiresEmail(t *testing.T) {
	t.Parallel()

	_, plugin := newTestLimenAndPlugin(t)

	_, err := plugin.RequestMagicLink(context.Background(), "   ")
	assert.ErrorIs(t, err, ErrEmailRequired)
}

func TestVerifyMagicLink_CreatesUserLazilyWhenAutoCreateEnabled(t *testing.T) {
	t.Parallel()

	var token string
	_, plugin := newTestLimenAndPlugin(t, WithSendMagicLink(func(msg MagicLinkMessage) {
		token = msg.Token
	}))

	_, err := plugin.RequestMagicLink(context.Background(), "lazy@test.com")
	require.NoError(t, err)

	_, err = plugin.dbAction.FindUserByEmail(context.Background(), "lazy@test.com")
	require.ErrorIs(t, err, limen.ErrRecordNotFound)

	result, _, err := plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	require.NotNil(t, result.User)
	assert.Equal(t, "lazy@test.com", result.User.Email)

	user, err := plugin.dbAction.FindUserByEmail(context.Background(), "lazy@test.com")
	require.NoError(t, err)
	assert.Equal(t, result.User.ID, user.ID)
}

func TestVerifyMagicLink_PropagatesAdditionalDataToNewUser(t *testing.T) {
	t.Parallel()

	var token string
	_, plugin := newTestLimenAndPlugin(t, WithSendMagicLink(func(msg MagicLinkMessage) {
		token = msg.Token
	}))

	_, err := plugin.RequestMagicLink(context.Background(), "meta@test.com", &RequestMagicLinkOptions{
		AdditionalData: map[string]any{
			"first_name": "Ada",
			"role":       "founder",
		},
	})
	require.NoError(t, err)

	_, _, err = plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)

	user, err := plugin.dbAction.FindUserByEmail(context.Background(), "meta@test.com")
	require.NoError(t, err)
	assert.Equal(t, "Ada", user.Raw()["first_name"])
	assert.Equal(t, "founder", user.Raw()["role"])
}

func TestVerifyMagicLink_MarksEmailVerifiedAndConsumesOneUse(t *testing.T) {
	t.Parallel()

	var token string
	_, plugin := newTestLimenAndPlugin(t, WithSendMagicLink(func(msg MagicLinkMessage) {
		token = msg.Token
	}))

	_, err := plugin.RequestMagicLink(context.Background(), "verify@test.com")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	result, _, err := plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	require.NotNil(t, result.User.EmailVerifiedAt)

	_, _, err = plugin.VerifyMagicLink(context.Background(), token)
	assert.ErrorIs(t, err, ErrMagicLinkTokenInvalid)
}

func TestVerifyMagicLink_MarksExistingUnverifiedUser(t *testing.T) {
	t.Parallel()

	var token string
	l, plugin := newTestLimenAndPlugin(t, WithSendMagicLink(func(msg MagicLinkMessage) {
		token = msg.Token
	}))

	seeded := limen.SeedTestUser(t, l, "existing@test.com")
	require.Nil(t, seeded.EmailVerifiedAt, "precondition: seeded user must be unverified")

	_, err := plugin.RequestMagicLink(context.Background(), "existing@test.com")
	require.NoError(t, err)

	result, _, err := plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	require.NotNil(t, result.User.EmailVerifiedAt, "existing user should be marked verified after magic-link verify")
	assert.Equal(t, seeded.ID, result.User.ID, "should re-use the existing user, not create a new one")
}

func TestVerifyMagicLink_RespectsMarkEmailVerifiedDisabled(t *testing.T) {
	t.Parallel()

	var token string
	l, plugin := newTestLimenAndPlugin(t,
		WithMarkEmailVerified(false),
		WithSendMagicLink(func(msg MagicLinkMessage) {
			token = msg.Token
		}),
	)

	limen.SeedTestUser(t, l, "skip-verify@test.com")

	_, err := plugin.RequestMagicLink(context.Background(), "skip-verify@test.com")
	require.NoError(t, err)

	result, _, err := plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	assert.Nil(t, result.User.EmailVerifiedAt, "EmailVerifiedAt must remain nil when markEmailVerified is disabled")
}

func TestVerifyMagicLink_AllowsConfiguredMultipleUses(t *testing.T) {
	t.Parallel()

	var token string
	_, plugin := newTestLimenAndPlugin(t,
		WithMaxUses(2),
		WithSendMagicLink(func(msg MagicLinkMessage) {
			token = msg.Token
		}),
	)

	_, err := plugin.RequestMagicLink(context.Background(), "multi@test.com")
	require.NoError(t, err)

	_, _, err = plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	_, _, err = plugin.VerifyMagicLink(context.Background(), token)
	require.NoError(t, err)
	_, _, err = plugin.VerifyMagicLink(context.Background(), token)
	assert.ErrorIs(t, err, ErrMagicLinkTokenInvalid)
}

func TestVerifyMagicLink_ExpiredToken(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		var token string
		_, plugin := newTestLimenAndPlugin(t,
			WithTokenExpiration(15*time.Minute),
			WithSendMagicLink(func(msg MagicLinkMessage) {
				token = msg.Token
			}),
		)

		_, err := plugin.RequestMagicLink(context.Background(), "expired@test.com")
		require.NoError(t, err)
		require.NotEmpty(t, token)

		time.Sleep(16 * time.Minute)

		_, _, err = plugin.VerifyMagicLink(context.Background(), token)
		assert.ErrorIs(t, err, ErrMagicLinkTokenInvalid)
	})
}
