package holdem

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrSeatTaken = errors.New("the seat is token by other player")
)

type Game interface{}

type Roomer interface {
	Name()
}

type holdem struct {
	poker       *Poker
	roundNum    int
	seatCount   int8
	playerCount int8
	players     map[int8]*Agent
	playerSeat  []int8
	roomers     map[*Agent]bool
	buttonSeat  int8
	button      *Agent
	seatLock    sync.Mutex
	sb          int
	ante        int
	pot         int
	roundBet    int
	minRaise    int
	publicCards []*Card
	log         *zap.Logger
}

func NewHoldem(sc int8, sb int, ante int, log *zap.Logger) *holdem {
	return &holdem{
		poker:       NewPoker(),
		players:     make(map[int8]*Agent),
		roomers:     make(map[*Agent]bool),
		playerSeat:  make([]int8, 0),
		publicCards: make([]*Card, 0, 5),
		seatCount:   sc,
		buttonSeat:  -1,
		sb:          sb,
		ante:        ante,
		log:         log,
	}
}

func (c *holdem) StandUp(i int8, r *Agent) error {
	if c.players[i] == nil {
		return errors.New("no player")
	}
	c.seatLock.Lock()
	r.gameInfo.seatNumber = -1
	c.players[i] = nil
	c.playerCount--
	c.seatLock.Unlock()
	//通知其他人
	for rr := range c.roomers {
		if rr != r {
			rr.recv.RoomerStandUp(i, r.user)
		}
	}
	return nil
}

func (c *holdem) Join(rs ...*Agent) {
	for _, r := range rs {
		c.roomers[r] = true
	}
}

//Seated 坐下
func (c *holdem) Seated(i int8, r *Agent) error {
	if c.players[i] != nil {
		return ErrSeatTaken
	}
	c.seatLock.Lock()
	r.gameInfo.seatNumber = i
	c.players[i] = r
	c.playerCount++
	c.seatLock.Unlock()
	//通知其他人
	for rr := range c.roomers {
		if rr != r {
			rr.recv.RoomerSeated(i, r.user)
		}
	}
	return nil
}

func (c *holdem) CheckAndStart() bool {
	if len(c.players) == int(c.seatCount) {
		go c.StartHand()
		return true
	}
	return false
}

func (c *holdem) buttonPosition() {
	var first, cur, last *Agent
	var i int8
	var buIdx int8 = -1
	if c.roundNum == 0 {
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		buIdx = int8(rd.Intn(int(c.seatCount)))
	} else {
		buIdx = c.buttonSeat + 1
		if buIdx >= c.seatCount {
			buIdx -= c.seatCount
		}
	}
	c.roundNum++
	for i = 0; i < c.seatCount; i++ {
		if p, ok := c.players[i]; ok {
			if cur == nil {
				cur = p
				first = p
				last = p
				continue
			}
			cur.nextAgent = p
			cur = p
			last = p
		}
	}
	last.nextAgent = first
	cur = first
	for {
		if cur.gameInfo.seatNumber >= buIdx {
			c.buttonSeat = cur.gameInfo.seatNumber
			c.button = cur
			break
		}
		if cur.gameInfo.seatNumber == last.gameInfo.seatNumber {
			c.buttonSeat = first.gameInfo.seatNumber
			c.button = first
			break
		}
		cur = cur.nextAgent
	}
}

func (c *holdem) smallBlind() {
	u := c.button.nextAgent
	if u.gameInfo.chip >= c.sb {
		c.pot += c.sb
		u.gameInfo.roundBet = c.sb
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefSB
		for r := range c.roomers {
			r.recv.RoomerGetAction(u.gameInfo.seatNumber, ActionDefSB, c.sb)
		}
		c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", c.sb))
		return
	}
	//不够时 全下
	c.pot += u.gameInfo.chip
	u.gameInfo.roundBet = u.gameInfo.chip
	u.gameInfo.handBet += u.gameInfo.roundBet
	u.gameInfo.chip -= u.gameInfo.roundBet
	u.gameInfo.status = ActionDefAllIn
	for r := range c.roomers {
		r.recv.RoomerGetAction(u.gameInfo.seatNumber, ActionDefAllIn, u.gameInfo.chip)
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

func (c *holdem) bigBlind() {
	u := c.button.nextAgent.nextAgent
	if u.gameInfo.chip >= 2*c.sb {
		c.pot += c.sb * 2
		u.gameInfo.roundBet = c.sb * 2
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefBB
		for r := range c.roomers {
			r.recv.RoomerGetAction(u.gameInfo.seatNumber, ActionDefBB, c.sb*2)
		}
		c.log.Debug("big blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", c.sb*2))
		return
	}
	//不够时 全下
	c.pot += u.gameInfo.chip
	u.gameInfo.roundBet = u.gameInfo.chip
	u.gameInfo.handBet += u.gameInfo.roundBet
	u.gameInfo.chip -= u.gameInfo.roundBet
	u.gameInfo.status = ActionDefAllIn
	for r := range c.roomers {
		r.recv.RoomerGetAction(u.gameInfo.seatNumber, ActionDefAllIn, u.gameInfo.chip)
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

const waitBetTimeout = 2 * time.Second

func (c *holdem) preflop() ([]*Agent, bool) {
	c.roundBet = c.sb * 2
	c.minRaise = c.sb * 2
	u := c.button.nextAgent.nextAgent.nextAgent
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	for {
		if u.gameInfo.status == ActionDefFold || u.gameInfo.status == ActionDefAllIn {
			u = u.nextAgent
		}
		bet := u.waitBet(c.roundBet, c.minRaise, 1, waitBetTimeout)
		switch bet.Action {
		case ActionDefFold:
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = bet.RoundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = bet.RoundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := bet.RoundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if bet.RoundBet > c.roundBet {
				c.roundBet = bet.RoundBet
			}
		default:
			c.log.Error("incorrect action", zap.Int8("action", int8(bet.Action)))
			panic("incorrect action")
		}
		for r := range c.roomers {
			if r != u {
				r.recv.RoomerGetAction(u.gameInfo.seatNumber, bet.Action, bet.Num)
			}
		}
		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		if !roundComplete {
			u = u.nextAgent
		} else {
			break
		}
	}
	if showcard {
		scs := make([]*ShowCard, 0)
		for _, v := range unfoldUsers {
			scs = append(scs, &ShowCard{
				SeatNumber: v.gameInfo.seatNumber,
				Cards:      v.gameInfo.cards,
			})
		}
		for r := range c.roomers {
			r.recv.RoomerGetShowCards(scs)
		}
	}
	return unfoldUsers, showcard
}

func (c *holdem) flopTurnRiver(isRiver ...bool) ([]*Agent, bool) {
	c.roundBet = 0
	c.minRaise = c.sb * 2
	u := c.button.nextAgent
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	for {
		//跳过all in和盖牌的玩家
		if u.gameInfo.status != ActionDefFold && u.gameInfo.status != ActionDefAllIn {
			u = u.nextAgent
			continue
		}
		bet := u.waitBet(c.roundBet, c.minRaise, 1, waitBetTimeout)
		switch bet.Action {
		case ActionDefFold:
		case ActionDefCheck:
		case ActionDefBet:
			c.pot += bet.Num
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = bet.RoundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = bet.RoundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := bet.RoundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if bet.RoundBet > c.roundBet {
				c.roundBet = bet.RoundBet
			}
		default:
			c.log.Error("incorrect action", zap.Int8("action", int8(bet.Action)))
			panic("incorrect action")
		}
		for r := range c.roomers {
			if r != u {
				r.recv.RoomerGetAction(u.gameInfo.seatNumber, bet.Action, bet.Num)
			}
		}
		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		if !roundComplete {
			u = u.nextAgent
		} else {
			break
		}
	}
	//非河牌直接亮牌
	if len(isRiver) == 0 && showcard {
		scs := make([]*ShowCard, 0)
		for _, v := range unfoldUsers {
			scs = append(scs, &ShowCard{
				SeatNumber: v.gameInfo.seatNumber,
				Cards:      v.gameInfo.cards,
			})
		}
		for r := range c.roomers {
			r.recv.RoomerGetShowCards(scs)
		}
	}
	return unfoldUsers, showcard
}

func (c *holdem) checkRoundComplete() (bool, []*Agent, bool) {
	u := c.button
	users := make([]*Agent, 0)
	allInCount := 0
	for {
		//已盖牌跳过
		if u.gameInfo.status == ActionDefFold {
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		//All in 作为未盖牌跟进
		if u.gameInfo.status == ActionDefAllIn {
			users = append(users, u)
			allInCount++
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		//本轮下注不等于本轮的注,直接返回未结束
		if u.gameInfo.roundBet != c.roundBet {
			return false, nil, false
		}
		users = append(users, u)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	return true, users, allInCount >= len(users)-1
}

func (c *holdem) deal(cnt int) {
	first := c.button.nextAgent
	cards := make([][]*Card, c.playerCount)
	for ; cnt > 0; cnt-- {
		for i := 0; i < int(c.playerCount); i++ {
			cds, _ := c.poker.GetCards(1)
			if len(cards[i]) == 0 {
				cards[i] = make([]*Card, 0)
			}
			cards[i] = append(cards[i], cds...)
		}
	}
	cur := first
	seats := make([]int8, 0)
	i := 0
	for {
		seats = append(seats, cur.gameInfo.seatNumber)
		cur.gameInfo.cards = cards[i]
		cur.recv.PlayerGetCard(cards[i])
		i++
		cur = cur.nextAgent
		if cur == first {
			break
		}
	}
	for r := range c.roomers {
		if r.gameInfo == nil {
			r.recv.RoomerGetCard(seats, int8(cnt))
		}
	}
}

//StartHand 开始新的一手
func (c *holdem) StartHand() {
	c.pot = 0
	if c.ante > 0 {
		c.pot += int(c.playerCount) * c.ante
	}
	c.publicCards = c.publicCards[:0]
	//洗牌
	c.poker.Reset()
	//确定庄家位,处理所有在座玩家
	c.buttonPosition()
	//下盲注
	c.smallBlind()
	c.bigBlind()
	//发牌
	c.deal(2)
	//翻牌前下注
	users, showcard := c.preflop()
	//如果只有一个人翻牌游戏结束
	if len(users) == 1 {
		c.simpleWin(users[0])
		return
	}
	//洗牌,并发送3张公共牌
	c.dealPublicCards(3)
	//未亮牌要下注
	if !showcard {
		//翻牌轮下注
		users, showcard = c.flopTurnRiver()
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//洗牌,并发送1张公共牌
	c.dealPublicCards(1)
	//未亮牌要下注
	if !showcard {
		//转牌轮下注
		users, showcard = c.flopTurnRiver()
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//洗牌,并发送1张公共牌
	c.dealPublicCards(1)
	//未亮牌要下注
	if !showcard {
		//河牌轮下注
		users, _ = c.flopTurnRiver()
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//比牌计算结果
	c.complexWin(users)
}

func (c *holdem) dealPublicCards(n int) {
	//洗牌
	_, _ = c.poker.GetCards(1)
	cards, _ := c.poker.GetCards(n)
	c.publicCards = append(c.publicCards, cards...)
	for r := range c.roomers {
		r.recv.RoomerGetPublicCard(cards)
	}
}

func (c *holdem) complexWin(users []*Agent) {
	pots := c.calcPot(users)
	results, _, _ := c.calcWin(users, pots)
	c.pot = 0
	ret := make([]*Result, 0)
	for _, v := range users {
		r := &Result{
			SeatNumber:    v.gameInfo.seatNumber,
			Cards:         v.gameInfo.cardResults,
			HandValueType: v.gameInfo.handValue.MaxHandValueType(),
		}
		if rv, ok := results[v.gameInfo.seatNumber]; ok {
			r.Num = rv.Num
		}
		ret = append(ret, r)
	}
	for r := range c.roomers {
		r.recv.RoomerGetResult(ret)
	}
}

func (c *holdem) simpleWin(agent *Agent) {
	agent.gameInfo.chip += int(c.pot)
	results := []*Result{{
		SeatNumber: agent.gameInfo.seatNumber,
		Num:        c.pot,
	}}
	c.pot = 0
	for r := range c.roomers {
		r.recv.RoomerGetResult(results)
	}
}
