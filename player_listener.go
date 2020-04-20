package holdem

type PlayerListener interface {
	PreFlop(*Game, [2]*Card) error
	BetStart(*Game) error
	Flop(*Game, []*Card) error
	Turn(*Game, *Card) error
	River(*Game, *Card) error
	Bet(*Game, *Player, int64) error
	Call(*Game, *Player) error
	Fold(*Game, *Player) error
	Check(*Game, *Player) error
	Raise(*Game, *Player, int64) error
	AllIn(*Game, *Player, int64) error
}
