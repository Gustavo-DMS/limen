package magiclink

import (
	"errors"
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrEmailRequired                 = limen.NewLimenError("email is required", http.StatusUnprocessableEntity, nil)
	ErrEmailNotFound                 = limen.NewLimenError("email not found", http.StatusNotFound, nil)
	ErrMagicLinkTokenInvalid         = limen.NewLimenError("invalid or expired magic link token", http.StatusBadRequest, nil)
	ErrMagicLinkTokenMaxUsesExceeded = limen.NewLimenError("magic link token has reached the maximum number of uses", http.StatusBadRequest, nil)
	ErrMaxUsesInvalid                = limen.NewLimenError("max uses must be greater than zero", http.StatusUnprocessableEntity, nil)
)

func isInvalidOrExpiredMagicLinkError(err error) bool {
	return errors.Is(err, ErrMagicLinkTokenInvalid) ||
		errors.Is(err, ErrMagicLinkTokenMaxUsesExceeded) ||
		errors.Is(err, ErrMaxUsesInvalid) ||
		errors.Is(err, ErrEmailNotFound)
}
