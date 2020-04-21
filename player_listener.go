package holdem

type PlayerListener interface {
	AudienceListener
	HandCards(Game, [2]*Card) error
	BetStart(Game) error
	Win(Game, int64) error
}
