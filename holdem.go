package holdem

import (
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	RoundPreFlop int8 = iota + 1
	RoundFlop
	RoundTurn
	RoundRiver
)

type Holdem struct {
	poker       *Poker
	handNum     uint
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
	nextGame    func(uint) bool
}

func NewHoldem(sc int8, sb int, ante int, nextGame func(uint) bool, log *zap.Logger) *Holdem {
	if nextGame == nil {
		nextGame = func(uint) bool {
			return false
		}
	}
	return &Holdem{
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
		nextGame:    nextGame,
	}
}

func (c *Holdem) StandUp(i int8, r *Agent) {
	if c.players[i] == nil {
		//r.recv.ErrorOccur(errors.New("no player"))
		return
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
}

func (c *Holdem) Join(rs ...*Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	for _, r := range rs {
		c.roomers[r] = true
		r.recv.RoomerGameInformation(c)
	}
}

func (c *Holdem) Seats() []int8 {
	var i int8
	ret := make([]int8, 0)
	for ; i < c.seatCount; i++ {
		if _, ok := c.players[i]; !ok {
			ret = append(ret, i)
		}
	}
	return ret
}

//Seated 坐下
func (c *Holdem) Seated(i int8, r *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	if c.players[i] != nil {
		r.ErrorOccur(ErrCodeSeatTaken, errSeatTaken)
		return
	}
	if r.gameInfo == nil || r.gameInfo.chip == 0 {
		r.ErrorOccur(ErrCodeNoChip, errNoChip)
		return
	}
	r.gameInfo.seatNumber = i
	c.players[i] = r
	c.playerCount++
	//通知其他人
	for rr := range c.roomers {
		if rr != r {
			rr.recv.RoomerSeated(i, r.user)
		}
	}
	c.checkAndStart()
}

func (c *Holdem) Information() (ante int, sb int, pot int, publicCards []*Card, seatCount int8, players []*ShowUser, onlines int) {
	ante = c.ante
	sb = c.sb
	pot = c.pot
	publicCards = c.publicCards
	seatCount = c.seatCount
	onlines = len(c.roomers)
	players = make([]*ShowUser, 0)
	for _, v := range c.players {
		players = append(players, v.ShowUser())
	}
	return
}

func (c *Holdem) checkAndStart() bool {
	if len(c.players) == int(c.seatCount) {
		go c.Start()
		return true
	}
	return false
}

func (c *Holdem) buttonPosition() {
	var first, cur, last *Agent
	var i int8
	var buIdx int8 = -1
	if c.handNum == 0 {
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		buIdx = int8(rd.Intn(int(c.seatCount))) + 1
	} else {
		buIdx = c.buttonSeat + 1
		if buIdx > c.seatCount {
			buIdx -= c.seatCount
		}
	}
	c.handNum++
	for i = 1; i <= c.seatCount; i++ {
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

func (c *Holdem) smallBlind() {
	u := c.button.nextAgent
	if u.gameInfo.chip >= c.sb {
		c.pot += c.sb
		u.gameInfo.roundBet = c.sb
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefSB
		for r := range c.roomers {
			r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, ActionDefSB, c.sb)
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
		r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, ActionDefAllIn, u.gameInfo.chip)
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

func (c *Holdem) bigBlind() {
	u := c.button.nextAgent.nextAgent
	if u.gameInfo.chip >= 2*c.sb {
		c.pot += c.sb * 2
		u.gameInfo.roundBet = c.sb * 2
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefBB
		for r := range c.roomers {
			r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, ActionDefBB, c.sb*2)
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
		r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, ActionDefAllIn, u.gameInfo.chip)
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

const waitBetTimeout = 20 * time.Second

func (c *Holdem) preflop() ([]*Agent, bool) {
	c.roundBet = c.sb * 2
	c.minRaise = c.sb * 2
	u := c.button.nextAgent.nextAgent.nextAgent
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	for {
		if u.gameInfo.status == ActionDefFold || u.gameInfo.status == ActionDefAllIn {
			u = u.nextAgent
		}
		bet := u.waitBet(c.roundBet, c.minRaise, RoundPreFlop, waitBetTimeout)
		switch bet.Action {
		case ActionDefFold:
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = u.gameInfo.roundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = u.gameInfo.roundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := u.gameInfo.roundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if u.gameInfo.roundBet > c.roundBet {
				c.roundBet = u.gameInfo.roundBet
			}
		default:
			c.log.Error("incorrect action", zap.String("action", bet.Action.String()))
			panic("incorrect action")
		}
		for r := range c.roomers {
			if r != u {
				r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, bet.Action, bet.Num)
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

func (c *Holdem) flopTurnRiver(round int8) ([]*Agent, bool) {
	c.roundBet = 0
	c.minRaise = c.sb * 2
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	//清理此轮
	uu := c.button
	for {
		uu.gameInfo.roundBet = 0
		uu = uu.nextAgent
		if uu == c.button {
			break
		}
	}
	u := c.button.nextAgent
	for {
		//跳过all in和盖牌的玩家
		if u.gameInfo.status == ActionDefFold || u.gameInfo.status == ActionDefAllIn {
			u = u.nextAgent
			continue
		}
		bet := u.waitBet(c.roundBet, c.minRaise, round, waitBetTimeout)
		switch bet.Action {
		case ActionDefFold:
		case ActionDefCheck:
		case ActionDefBet:
			c.pot += bet.Num
			c.roundBet = bet.Num
		case ActionDefCall:
			c.pot += bet.Num
		case ActionDefRaise:
			c.pot += bet.Num
			c.minRaise = u.gameInfo.roundBet - c.roundBet //当轮下注额度 - 目前这轮最高下注额
			c.roundBet = u.gameInfo.roundBet              //更新最高下注额
		case ActionDefAllIn:
			c.pot += bet.Num
			raise := u.gameInfo.roundBet - c.roundBet
			//如果加注大于最小加注 视为raise,否则视为call
			if raise >= c.minRaise {
				c.minRaise = raise
			}
			//大于本轮最大下注时候才更新本轮最大
			if u.gameInfo.roundBet > c.roundBet {
				c.roundBet = u.gameInfo.roundBet
			}
		default:
			c.log.Error("incorrect action", zap.Int8("action", int8(bet.Action)))
			panic("incorrect action")
		}
		for r := range c.roomers {
			if r != u {
				r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, u.gameInfo.seatNumber, bet.Action, bet.Num)
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
	if round != RoundRiver && showcard {
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

func (c *Holdem) checkRoundComplete() (bool, []*Agent, bool) {
	u := c.button
	users := make([]*Agent, 0)
	allInCount := 0
	hasCheck := false
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
		if c.roundBet > 0 && u.gameInfo.roundBet != c.roundBet {
			return false, nil, false
		}
		users = append(users, u)
		//是否包含一个check 用户
		if u.gameInfo.status == ActionDefCheck {
			hasCheck = true
		}
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	//如果大于一个人check,则未结束
	if hasCheck && len(users) > 1 {
		return false, nil, false
	}
	return true, users, allInCount > 0 && len(users) > 1 && allInCount >= len(users)-1
}

func (c *Holdem) deal(cnt int) {
	first := c.button.nextAgent
	cards := make([][]*Card, c.playerCount)
	max := cnt
	for ; max > 0; max-- {
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
		cur = cur.nextAgent
		if cur == first {
			break
		}
	}
	for {
		cur.gameInfo.cards = cards[i]
		cur.recv.PlayerGetCard(cur.gameInfo.seatNumber, cards[i], seats, int8(cnt))
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

func (c *Holdem) Start() {
	for {
		c.startHand()
		if !c.nextGame(c.handNum) {
			break
		}
	}
}

//StartHand 开始新的一手
func (c *Holdem) startHand() {
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
		users, showcard = c.flopTurnRiver(RoundFlop)
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
		users, showcard = c.flopTurnRiver(RoundTurn)
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
		users, _ = c.flopTurnRiver(RoundRiver)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//比牌计算结果
	c.complexWin(users)
}

func (c *Holdem) dealPublicCards(n int) {
	//洗牌
	_, _ = c.poker.GetCards(1)
	cards, _ := c.poker.GetCards(n)
	c.publicCards = append(c.publicCards, cards...)
	for r := range c.roomers {
		r.recv.RoomerGetPublicCard(cards)
	}
}

func (c *Holdem) complexWin(users []*Agent) {
	pots := c.calcPot(users)
	results, _, _ := c.calcWin(users, pots)
	c.pot = 0
	ret := make([]*Result, 0)
	//所有玩家的最终状况
	for _, v := range c.players {
		r := &Result{
			SeatNumber: v.gameInfo.seatNumber,
		}
		if v.gameInfo.cardResults != nil {
			r.Cards = v.gameInfo.cardResults
			r.HandValueType = v.gameInfo.handValue.MaxHandValueType()
		}
		if rv, ok := results[v.gameInfo.seatNumber]; ok {
			v.gameInfo.chip += rv.Num
			r.Num = rv.Num
		}
		r.Chip = v.gameInfo.chip
		ret = append(ret, r)
	}
	for r := range c.roomers {
		r.recv.RoomerGetResult(ret)
	}
}

func (c *Holdem) simpleWin(agent *Agent) {
	ret := make([]*Result, 0)
	for _, v := range c.players {
		r := &Result{
			SeatNumber: v.gameInfo.seatNumber,
		}
		if v.gameInfo.seatNumber == agent.gameInfo.seatNumber {
			v.gameInfo.chip += c.pot
			r.Num = c.pot
			c.pot = 0
		}
		r.Chip = v.gameInfo.chip
		ret = append(ret, r)
	}
	for r := range c.roomers {
		r.recv.RoomerGetResult(ret)
	}
}
