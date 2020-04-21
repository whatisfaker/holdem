package holdem

type AudienceListener interface {
	PlayerPause(Game, Player) error
	PlayerResume(Game, Player) error
	PlayerBetStart(Game, Player) error
	PlayerBet(Game, Player, int64) error
	PlayerCall(Game, Player, int64) error
	PlayerRaise(Game, Player, int64) error
	PlayerCheck(Game, Player) error
	PlayerAllIn(Game, Player, int64) error
	PlayerFold(Game, Player) error
	PlayerWin(Game, []Player) error

	Flop(Game, []*Card) error
	Turn(Game, *Card) error
	River(Game, *Card) error
}
