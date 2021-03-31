package holdem

import "errors"

const (
	ErrCodeLessChip = 1001 + iota
	ErrCodeNotInBetTime
	ErrCodeSeatTaken
	ErrCodeNoChip
)

var (
	errLessChip     = errors.New("chip is less than 0")
	errNotInBetTime = errors.New("it is not in bet time")
	errSeatTaken    = errors.New("the seat is token by other player")
	errNoChip       = errors.New("you have bring in first")
)
