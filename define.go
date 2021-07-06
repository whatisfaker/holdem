package holdem

import "time"

type Round int8

const (
	RoundPreFlop Round = iota + 1
	RoundFlop
	RoundTurn
	RoundRiver
)

const (
	GameStatusNotStart int8 = iota
	GameStatusStarted
	GameStatusCancel
	GameStatusComplete
)

const (
	StandUpNone int8 = iota
	StandUpNoChip
	StandUpAction
	StandUpGameEnd
	StandUpGameForce
	StandUpGameExchange
	StandUpAutoExceedMaxTimes
)

func (c Round) String() string {
	switch c {
	case RoundPreFlop:
		return "preflop"
	case RoundFlop:
		return "flop"
	case RoundTurn:
		return "turn"
	case RoundRiver:
		return "river"
	}
	return "unknonw"
}

type ActionDef int8

const (
	ActionDefNone ActionDef = iota
	ActionDefAnte
	ActionDefSB
	ActionDefBB
	ActionDefBet
	ActionDefCall
	ActionDefFold
	ActionDefCheck
	ActionDefRaise
	ActionDefAllIn
)

func (c ActionDef) String() string {
	switch c {
	case ActionDefAnte:
		return "ante"
	case ActionDefSB:
		return "small blind"
	case ActionDefBB:
		return "big blind"
	case ActionDefBet:
		return "bet"
	case ActionDefCall:
		return "call"
	case ActionDefFold:
		return "fold"
	case ActionDefCheck:
		return "check"
	case ActionDefRaise:
		return "raise"
	case ActionDefAllIn:
		return "all in"
	default:
		return "ready"
	}
}

const (
	delaySend = 200 * time.Millisecond
)
