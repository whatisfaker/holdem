package holdem

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	SeatStatusEmpty int8 = iota + 1
	SeatStatusTaken
	SeatStatusSeated
)

const (
	StepNotStarted int8 = iota - 1
	StepPreFlopRound
	StepFlopRound
	StepTurnRound
	StepRiverRound
)

type Game struct {
	poker       *Poker
	burnCards   []*Card
	publicCards []*Card
	handCards   map[int8][2]*Card
	seatCount   int8
	players     map[int8]*Player
	seated      map[int8]int8
	step        int8
	log         *zap.Logger
	numLock     sync.Mutex
	listener    GameListener
	button      int8
	sb          int64
	pod         int64
	playerCount int8
	betCh       chan *Bet
}

func NewGame(count int8, gl GameListener, log *zap.Logger) *Game {
	g := &Game{
		poker:       NewPoker(),
		burnCards:   make([]*Card, 0),
		publicCards: make([]*Card, 0),
		handCards:   make(map[int8][2]*Card),
		seatCount:   count,
		players:     make(map[int8]*Player),
		seated:      make(map[int8]int8),
		log:         log,
		listener:    gl,
		button:      -1,
		betCh:       make(chan *Bet, 0),
	}
	var i int8
	for i = 0; i < count; i++ {
		g.seated[i] = SeatStatusEmpty
	}
	return g
}

func (c *Game) BurnCard() error {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return err
	}
	c.burnCards = append(c.burnCards, cards...)
	return nil
}

func (c *Game) handCard() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	return cards[0], nil
}

func (c *Game) Flop() ([]*Card, error) {
	cards, err := c.poker.GetCards(3)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards, nil
}

func (c *Game) Turn() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *Game) River() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *Game) nextSeat(index int8) int8 {
	index++
	if index >= c.playerCount {
		index = 0
	}
	return index
}

func (c *Game) getSmallBlindNum() int8 {
	l := len(c.players)
	if c.button == -1 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		c.button = int8(r.Intn(l))
	}
	if l == 2 {
		return c.button
	}
	return c.nextSeat(c.button)
}

func (c *Game) blinds() {
	seat := c.getSmallBlindNum()
	c.players[seat].Blinds(c.sb)
	c.players[c.nextSeat(seat)].Blinds(c.sb * 2)
}

func (c *Game) Start() error {

	c.listener.BeforeBlinds(c)
	c.blinds()
	c.listener.BeforePreFlop(c)
	var i int8
	seat := c.nextSeat(c.button)
	for i = 0; i < c.playerCount*2; i++ {
		card, err := c.handCard()
		if err != nil {
			c.log.Error("game hand card error", zap.Error(err))
			return err
		}
		player := c.players[seat]
		err = player.HandCard(card)
		if err != nil {
			c.log.Error("player hand card error", zap.Error(err))
			return err
		}
		seat = c.nextSeat(seat)
	}
	seat = c.getSmallBlindNum()
	seat = c.nextSeat(seat)
	seat = c.nextSeat(seat)
	c.players[seat].listener.BetStart(c)
	isRunning, err := c.waitBet(seat, 30*time.Second)
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}
	return nil
}

func (c *Game) LockNum(num int8, reNum bool) (int8, bool) {
	//invalid number
	if num < 0 || num >= c.seatCount {
		return 0, false
	}
	// has taken
	var realNum int8 = -1
	if _, ok := c.players[num]; ok {
		if !reNum {
			return 0, false
		}
		for k, v := range c.seated {
			if v == SeatStatusEmpty {
				realNum = k
				break
			}
		}
		if realNum < 0 {
			return 0, false
		}
	} else {
		realNum = num
	}
	c.numLock.Lock()
	defer c.numLock.Unlock()
	player := NewPlayer(c, num, c.log.With(zap.Int8("player", num)))
	c.players[realNum] = player
	c.seated[realNum] = SeatStatusTaken
	c.log.Debug("lock seat", zap.Int8("no", num), zap.Bool("re_num", reNum))
	return realNum, true
}

func (c *Game) UnlockNum(num int8) {
	c.numLock.Lock()
	defer c.numLock.Unlock()
	delete(c.players, num)
	c.seated[num] = SeatStatusEmpty
}

func (c *Game) Seated(num int8, bringIn int64, pl PlayerListener) (*Player, error) {
	if player, ok := c.players[num]; !ok {
		return nil, fmt.Errorf("no player lock num before seat")
	} else {
		player.bringIn = bringIn
		player.listener = pl
		c.numLock.Lock()
		defer c.numLock.Unlock()
		c.seated[num] = SeatStatusSeated
		return player, nil
	}
}
