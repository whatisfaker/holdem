package holdem

import (
	"go.uber.org/zap"
)

const (
	PlayerStatusNormal = iota
	PlayerStatusAllIn
	PlayerStatusFold
)

type Bet struct {
	num    int8
	bet    int64
	action int8
}

type Player interface {
	Pause()
	Resume()
	Bet(int64)
	Raise(int64)
	Fold()
	AllIn()
	Check()
	Call()
}

type player struct {
	game          *game
	number        int8
	handCards     [2]*Card
	maxHandValue  *HandValue
	log           *zap.Logger
	bringIn       int64
	gameStartLeft int64
	left          int64
	currentBet    int64
	listener      PlayerListener
	status        int8
	earn          int64
}

var _ Player = new(player)

func NewPlayer(game *game, num int8, log *zap.Logger) *player {
	return &player{
		game:      game,
		number:    num,
		handCards: [2]*Card{},
		log:       log,
	}
}

func (c *player) caculateMaxHandValue(pc []*Card) (*HandValue, error) {
	cards := append(pc, c.handCards[0], c.handCards[1])
	handValue, err := GetMaxHandValueFromCard(cards)
	if err != nil {
		return nil, err
	}
	c.maxHandValue = handValue
	return c.maxHandValue, nil
}

func (c *player) Bet(chip int64) {
	c.left -= chip
	c.currentBet += chip
	c.game.pod += chip
	c.game.betCh <- &Bet{
		num:    c.number,
		action: ActionBet,
		bet:    chip,
	}
}

func (c *player) Call() {
	more := c.game.currentBet - c.currentBet
	c.left -= more
	c.game.pod += more
	c.game.betCh <- &Bet{
		num:    c.number,
		action: ActionCall,
		bet:    more,
	}
}

func (c *player) AllIn() {
	c.game.pod += c.left
	c.game.betCh <- &Bet{
		num:    c.number,
		action: ActionAllIn,
		bet:    c.left,
	}
	c.left = 0
	c.status = PlayerStatusAllIn
}

func (c *player) Check() {

}

func (c *player) Raise(chip int64) {
	c.left -= chip
	c.currentBet += chip
	c.game.currentBet += chip
	c.game.pod += chip
	c.game.betCh <- &Bet{
		num:    c.number,
		action: ActionRaise,
		bet:    chip,
	}
}

func (c *player) Fold() {
	c.status = PlayerStatusFold
}

func (c *player) Pause() {
	c.game.Pause()
}

func (c *player) Resume() {
	c.game.Resume()
}

func (c *player) win(chip int64) error {
	c.left += chip
	c.earn = chip
	return c.listener.Win(c, chip)
}

func (c *player) bringInChip(b int64) {
	c.bringIn += b
	c.left += b
}

func (c *player) start() {
	c.gameStartLeft = c.left
}

func (c *player) resetBet() {
	c.currentBet = 0
}

func (c *player) handCard(card *Card) error {
	if c.handCards[0] == nil {
		c.handCards[0] = card
		return nil
	}
	c.handCards[1] = card
	return c.listener.HandCards(c.game, c.handCards)
}

func (c *player) blinds(chip int64) {
	c.left -= chip
	c.game.pod += chip
	c.currentBet = chip
}

func (c *player) betStart() {
	_ = c.listener.BetStart(c.game)
}
