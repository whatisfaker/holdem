package holdem

type Player struct {
	game         *Game
	num          int8
	handCards    [2]*Card
	maxHandValue *HandValue
}

type PlayerListener interface {
	PreFlop(*Game, [2]*Card) error
	Flop(*Game, []*Card) error
	Turn(*Game, *Card) error
	River(*Game, *Card) error
	Bet(*Game, *Player, int64) error
	Call(*Game, *Player) error
	Fold(*Game, *Player) error
	Raise(*Game, *Player, int64) error
	AllIn(*Game, *Player, int64) error
}

func NewPlayer(game *Game, num int8) *Player {
	return &Player{
		game:      game,
		num:       num,
		handCards: [2]*Card{},
	}
}

func (c *Player) CaculateMaxHandValue(pc []*Card) error {
	cards := append(pc, c.handCards[0], c.handCards[1])
	handValue, err := GetMaxHandValueFromCard(cards)
	if err != nil {
		return err
	}
	c.maxHandValue = handValue
	return nil
}

func (c *Player) HandCards([]*Card) {

}
