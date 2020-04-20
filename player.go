package holdem

import "go.uber.org/zap"

type Bet struct {
	num    int8
	bet    int64
	action int8
}

type Player struct {
	game         *Game
	number       int8
	handCards    [2]*Card
	maxHandValue *HandValue
	log          *zap.Logger
	bringIn      int64
	listener     PlayerListener
}

func NewPlayer(game *Game, num int8, log *zap.Logger) *Player {
	return &Player{
		game:      game,
		number:    num,
		handCards: [2]*Card{},
		log:       log,
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

func (c *Player) HandCard(card *Card) error {
	if c.handCards[0] == nil {
		c.handCards[0] = card
		return nil
	}
	c.handCards[1] = card
	return c.listener.PreFlop(c.game, c.handCards)
}

func (c *Player) Blinds(chip int64) {
	c.bringIn -= chip
	c.game.pod += chip
}

func (c *Player) Bet(chip int64) {
	c.bringIn -= chip
	c.game.pod += chip
	c.game.betCh <- &Bet{
		num:    c.number,
		action: ActionBet,
		bet:    chip,
	}
}
