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
	ErrCodeInvalidInsurance
	ErrCodeCannotEnablePayToPlay
	ErrCodeNotStandUp
	ErrCodeAlreadySeated
)

var (
	errLessChip              = errors.New("chip is less than 0")
	errNotInBetTime          = errors.New("it is not in bet time")
	errSeatTaken             = errors.New("the seat is token by other player")
	errNoChip                = errors.New("you have less chip")
	errInvalidBetAction      = errors.New("invalid bet action")
	errInvalidBetNum         = errors.New("invalid bet amount")
	errNotPlaying            = errors.New("you are not playing or bringin")
	errNoJoin                = errors.New("no game join")
	errNoSeat                = errors.New("not seated")
	errInvalidInsurance      = errors.New("invalid insurance amount")
	errCannotEnablePayToPlay = errors.New("you can't pay to play")
	errNotStandUp            = errors.New("you didnt leave table")
	errAlreadySeated         = errors.New("you are already seated")
)
