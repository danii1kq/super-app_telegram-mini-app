package split

import "errors"

var (
	ErrNoParticipants       = errors.New("split: no participants")
	ErrInvalidAmount        = errors.New("split: amount must be positive")
	ErrDuplicateParticipant = errors.New("split: duplicate participant")
	ErrNotMember            = errors.New("split: viewer is not a group member")
)
