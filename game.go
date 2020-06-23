package holdem

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/core/errors"
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

var ErrLessPlayer = errors.New("Players count is less then 2 ,can't start")

type Game interface {
}

type game struct {
	//一副牌
	poker *Poker
	//被抛弃的牌
	burnCards []*Card
	//公共牌
	publicCards []*Card
	//根据座位号的手牌
	handCards map[int8][2]*Card
	//座位总数
	seatCount int8
	//座位号的用户
	players map[int8]*player
	//坐下的用户状态
	seated map[int8]int8
	//步骤
	step int8
	//日志
	log *zap.Logger
	//锁
	numLock sync.Mutex
	//观众监听
	listener AudienceListener
	//游戏挂载点
	gameHook GameHook
	//按钮位
	button int8
	//小盲
	sb int64
	//前注
	ante int64
	pod  int64
	//玩家总数
	playerCount int8
	//投注通道
	betCh chan *Bet
	//当前投注
	currentBet int64
	//暂停时间秒
	pause int64
	//暂停通道
	pauseCh chan byte
}

func NewGame(count int8, gl AudienceListener, log *zap.Logger) *game {
	g := &game{
		poker:       NewPoker(),
		burnCards:   make([]*Card, 0),
		publicCards: make([]*Card, 0),
		handCards:   make(map[int8][2]*Card),
		seatCount:   count,
		players:     make(map[int8]*player),
		seated:      make(map[int8]int8),
		log:         log,
		listener:    gl,
		button:      -1,
		betCh:       make(chan *Bet),
		pauseCh:     make(chan byte),
	}
	var i int8
	for i = 0; i < count; i++ {
		g.seated[i] = SeatStatusEmpty
	}
	return g
}

func (c *game) BurnCard() error {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return err
	}
	c.burnCards = append(c.burnCards, cards...)
	return nil
}

func (c *game) handCard() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	return cards[0], nil
}

func (c *game) Flop() ([]*Card, error) {
	cards, err := c.poker.GetCards(3)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards, nil
}

func (c *game) Turn() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *game) River() (*Card, error) {
	cards, err := c.poker.GetCards(1)
	if err != nil {
		return nil, err
	}
	c.publicCards = append(c.publicCards, cards...)
	return cards[0], nil
}

func (c *game) Pause() {
	if c.pause == 0 {
		atomic.AddInt64(&c.pause, 1)
		c.pauseCh <- byte(1)
	}
	go func() {
		timer := time.NewTimer(1 * time.Minute)
		defer timer.Stop()
		<-timer.C
		if c.pause == 1 {
			atomic.AddInt64(&c.pause, -1)
			c.pauseCh <- byte(2)
		}
	}()
}

func (c *game) isPause() bool {
	return c.pause == 1
}

func (c *game) Resume() {
	if c.pause == 1 {
		atomic.AddInt64(&c.pause, -1)
	}
}

func (c *game) nextSeat(index int8, steps ...int8) int8 {
	var i int8
	var step int8 = 1
	if len(steps) > 0 {
		step = steps[0]
	}
	for i = 0; i < step; i++ {
		index++
		if index >= c.playerCount {
			index = 0
		}
	}
	return index
}

func (c *game) getSmallBlindNum() int8 {
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

//blinds 下盲注(+前注)
func (c *game) blinds() {
	seat := c.getSmallBlindNum()
	c.players[seat].blinds(c.sb + c.ante)
	c.players[c.nextSeat(seat)].blinds(c.sb*2 + c.ante)
	c.currentBet = c.sb * 2
}

func (c *game) Start() error {
	c.playerCount = 0
	c.numLock.Lock()
	for i := range c.players {
		if c.seated[i] == SeatStatusTaken {
			delete(c.players, i)
			c.seated[i] = SeatStatusEmpty
			continue
		}
		c.playerCount++
	}
	c.numLock.Unlock()
	if c.playerCount < 2 {
		return ErrLessPlayer
	}
	//记录开始的chip和用户初始化操作
	for _, v := range c.players {
		v.start()
		v.resetBet()
	}
	//1. 下盲注
	c.gameHook.BeforeBlinds(c)
	c.blinds()

	//2. 发底牌
	c.gameHook.BeforePreFlop(c)
	seat := c.nextSeat(c.button)
	var i int8
	for i = 0; i < c.playerCount*2; i++ {
		card, err := c.handCard()
		if err != nil {
			c.log.Error("game hand card error", zap.Error(err))
			return err
		}
		player := c.players[seat]
		err = player.handCard(card)
		if err != nil {
			c.log.Error("player hand card error", zap.Error(err))
			return err
		}
		seat = c.nextSeat(seat)
	}
	seat = c.getSmallBlindNum()
	seat = c.nextSeat(seat, 2)
	c.players[seat].betStart()
	isRunning, err := c.waitBet(seat, 30*time.Second)
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}
	return nil
}

func (c *game) LockNum(num int8, reNum bool) (int8, bool) {
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

func (c *game) UnlockNum(num int8) {
	c.numLock.Lock()
	defer c.numLock.Unlock()
	delete(c.players, num)
	c.seated[num] = SeatStatusEmpty
}

func (c *game) Seated(num int8, bringIn int64, pl PlayerListener) (Player, error) {
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
