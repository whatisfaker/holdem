package holdem

import "errors"

const (
	ErrCodeLessChip = 1001 + iota
	ErrCodeNotInBetTime
	ErrCodeSeatTaken
	ErrCodeNoChip
	ErrCodeInvalidBetAction
	ErrCodeInvalidBetNum
	ErrCodeNotPlaying
	ErrCodeNoJoin
	ErrCodeNoSeat
)

var (
	errLessChip         = errors.New("chip is less than 0")
	errNotInBetTime     = errors.New("it is not in bet time")
	errSeatTaken        = errors.New("the seat is token by other player")
	errNoChip           = errors.New("you have bring in first")
	errInvalidBetAction = errors.New("invalid bet action")
	errInvalidBetNum    = errors.New("invalid bet amount")
	errNotPlaying       = errors.New("you are not playing")
	errNoJoin           = errors.New("no game join")
	errNoSeat           = errors.New("not seated")
)
