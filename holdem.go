package holdem

import (
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Round int8

const (
	RoundPreFlop Round = iota + 1
	RoundFlop
	RoundTurn
	RoundRiver
)

func (c Round) String() string {
	switch c {
	case RoundPreFlop:
		return "preflop"
	case RoundFlop:
		return "flop"
	case RoundTurn:
		return "turn"
	case RoundRiver:
		return "river"
	}
	return "unknonw"
}

type StartNewHandInfo struct {
	AnteAllIns []int8
	SB         *Bet
	SBSeat     int8
	BB         *Bet
	BBSeat     int8
	PayToPlay  []int8
}

type Holdem struct {
	poker                *Poker                              //扑克
	handNum              uint                                //手数
	seatCount            int8                                //座位数量
	playerCount          int8                                //座位上用户数量
	players              map[int8]*Agent                     //座位上用户字典
	playingPlayerCount   int8                                //当前游戏玩家数量（座位上游戏的）
	roomers              map[*Agent]bool                     //参与游戏的玩家（包括旁观）
	buttonSeat           int8                                //庄家座位号
	sbSeat               int8                                //小盲座位号
	bbSeat               int8                                //大盲座位号
	isPayToPlay          bool                                //是否需要补盲
	payToPlayMap         map[int8]PlayType                   //补牌的规则
	button               *Agent                              //庄家玩家
	waitBetTimeout       time.Duration                       //等待下注的超时时间
	seatLock             sync.Mutex                          //玩家锁
	isStarted            bool                                //是否开始
	isPlaying            bool                                //是否在游戏中
	handStartInfo        *StartNewHandInfo                   //当前一手开局信息
	sb                   int                                 //小盲
	ante                 int                                 //前注
	pot                  int                                 //彩池
	roundBet             int                                 //当前轮下注额
	minRaise             int                                 //最小加注量
	publicCards          []*Card                             //公共牌
	log                  *zap.Logger                         //日志
	autoStart            bool                                //是否自动开始
	minPlayers           int8                                //最小自动开始玩家
	nextGame             func(uint) (bool, time.Duration)    //是否继续下一轮的回调函数和等待下一手时间(当前手数) - 内部可以用各种条件来判断是否继续
	insurance            bool                                //是否有保险
	insuranceOdds        map[int]float64                     //保险赔率
	insuranceWaitTimeout time.Duration                       //保险等待时间
	insuranceResult      map[int8]map[Round]*InsuranceResult //保险结果
	insuranceUsers       []*Agent                            //参与保险的玩家
}

func NewHoldem(
	sc int8, //最大座位数
	sb int, //小盲
	ante int, //前注
	autoStart bool, //是否自动开始
	minPlayers int8, //最小游戏人数
	waitBetTimeout time.Duration, //等待下注超时时间
	nextGame func(uint) (bool, time.Duration), //是否继续下一手判断/等待时间
	isPayToPlay bool, //是否需要补盲
	insurance bool, //保险
	log *zap.Logger, //日志
) *Holdem {
	if nextGame == nil {
		nextGame = func(uint) (bool, time.Duration) {
			return false, 0
		}
	}
	payMap := make(map[int8]PlayType)
	var i int8
	for i = 1; i <= sc; i++ {
		payMap[i] = PlayTypeNormal
	}
	return &Holdem{
		poker:          NewPoker(),
		players:        make(map[int8]*Agent),
		roomers:        make(map[*Agent]bool),
		publicCards:    make([]*Card, 0, 5),
		waitBetTimeout: waitBetTimeout,
		seatCount:      sc,
		autoStart:      autoStart,
		minPlayers:     minPlayers,
		sb:             sb,
		ante:           ante,
		log:            log,
		insurance:      insurance,
		nextGame:       nextGame,
		isPayToPlay:    isPayToPlay,
		payToPlayMap:   payMap,
	}
}

//Join 加入游戏,并没有坐下
func (c *Holdem) join(rs ...*Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	for _, r := range rs {
		c.roomers[r] = true
		r.recv.RoomerGameInformation(c)
	}
}

//Seated 坐下
func (c *Holdem) seated(i int8, r *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	if c.players[i] != nil {
		r.ErrorOccur(ErrCodeSeatTaken, errSeatTaken)
		return
	}
	if r.gameInfo == nil || r.gameInfo.chip < c.ante+c.sb*2 {
		r.ErrorOccur(ErrCodeNoChip, errNoChip)
		return
	}
	r.gameInfo.seatNumber = i
	r.gameInfo.te = PlayTypeNormal
	c.players[i] = r
	c.playerCount++
	//开启补盲
	if c.isPayToPlay {
		r.gameInfo.te = c.payToPlayMap[i]
	}
	//通知自己坐下了
	r.recv.PlayerSeatedSuccess(i, r.gameInfo.te)
	//通知其他人
	for rr := range c.roomers {
		if rr != r {
			rr.recv.RoomerSeated(i, r.user, r.gameInfo.te)
		}
	}
	if !c.isStarted && c.autoStart && c.playerCount >= c.minPlayers {
		if ok, _ := c.nextGame(c.handNum); ok {
			c.Start()
		}
	}
}

//directStandUp 不用等待本手结束直接站起来
func (c *Holdem) directStandUp(i int8, r *Agent) {
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	c.standUp(i, r)
}

//standUp 站起来
func (c *Holdem) standUp(i int8, r *Agent) {
	r.gameInfo = nil
	delete(c.players, i)
	c.playerCount--
	//通知自己站起来了
	r.recv.PlayerStandUp(i)
	//通知其他人
	for rr := range c.roomers {
		if rr != r {
			rr.recv.RoomerStandUp(i, r.user)
		}
	}
}

//Information 游戏信息
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

//buttonPosition 决定按钮位置
func (c *Holdem) buttonPosition() bool {
	c.log.Debug("button position begin", zap.Int8("seat count", c.playerCount))
	var cur *Agent
	var i, buIdx int8
	if c.handNum == 0 {
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		buIdx = int8(rd.Intn(int(c.seatCount))) + 1
	} else {
		//庄位移动
		buIdx = c.sbSeat
	}
	c.handNum++
	c.seatLock.Lock()
	defer c.seatLock.Unlock()
	c.playingPlayerCount = 0
	payMap := make(map[int8]PlayType)
	for i = 0; i < c.seatCount; i++ {
		seat := ((i + buIdx - 1) % c.seatCount) + 1
		payMap[seat] = PlayTypeNeedPayToPlay
		if p, ok := c.players[seat]; ok {
			//不需要补盲
			if !c.isPayToPlay {
				p.gameInfo.te = PlayTypeNormal
			}
			if p.gameInfo.te == PlayTypeNormal || p.gameInfo.te == PlayTypeAgreePayToPlay {
				c.playingPlayerCount++
			}
			if cur == nil {
				c.button = p
				c.buttonSeat = seat
				cur = p
				if p.gameInfo.te == PlayTypeDisable {
					p.gameInfo.te = PlayTypeNeedPayToPlay
					//通知玩家可以补盲了
					p.recv.PlayerCanPayToPlay(seat)
				}
				continue
			}
			cur.nextAgent = p
			cur = p
		}
	}
	cur.nextAgent = c.button
	c.sbSeat = c.button.nextAgent.gameInfo.seatNumber
	c.bbSeat = c.button.nextAgent.nextAgent.gameInfo.seatNumber
	if c.isPayToPlay {
		payMap[c.sbSeat] = PlayTypeDisable
		payMap[c.bbSeat] = PlayTypeDisable
		if c.button.nextAgent.nextAgent.gameInfo.te == PlayTypeNeedPayToPlay {
			c.button.nextAgent.nextAgent.gameInfo.te = PlayTypeNormal
			c.playingPlayerCount++
		}
	} else {
		for k := range payMap {
			payMap[k] = PlayTypeNormal
		}
	}
	if c.playingPlayerCount <= 1 {
		c.log.Debug("button position end(false)", zap.Int8("seat count", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
		return false
	}
	c.payToPlayMap = payMap
	c.handStartInfo = &StartNewHandInfo{
		AnteAllIns: []int8{},
		PayToPlay:  []int8{},
		SBSeat:     c.sbSeat,
		BBSeat:     c.bbSeat,
	}
	c.log.Debug("button position end(true)", zap.Int8("buseat", c.buttonSeat), zap.String("buuser", c.button.user.ID()), zap.Int8("players", c.playerCount), zap.Int8("pc", c.playingPlayerCount))
	return true
}

//doAnte 前注
func (c *Holdem) doAnte() {
	u := c.button
	for {
		if u.gameInfo.te == PlayTypeDisable || u.gameInfo.te == PlayTypeNeedPayToPlay {
			u = u.nextAgent
			if u == c.button {
				break
			}
			continue
		}
		if u.gameInfo.chip >= c.ante {
			c.pot += c.ante
			u.gameInfo.chip -= c.ante
			u.gameInfo.status = ActionDefNone
			c.log.Debug("ante", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", c.ante))
			continue
		}
		c.handStartInfo.AnteAllIns = append(c.handStartInfo.AnteAllIns, u.gameInfo.seatNumber)
		c.pot += u.gameInfo.chip
		u.gameInfo.chip = 0
		u.gameInfo.status = ActionDefAllIn
		c.log.Debug("ante", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.chip))
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
}

//smallBlind 小盲
func (c *Holdem) smallBlind() {
	u := c.button.nextAgent
	if u.gameInfo.te == PlayTypeDisable || u.gameInfo.te == PlayTypeNeedPayToPlay {
		c.log.Debug("small blind(empty)", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int8("play type", int8(u.gameInfo.te)))
		return
	}
	if u.gameInfo.status == ActionDefAllIn {
		c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("allin", 0))
		return
	}
	if u.gameInfo.chip >= c.sb {
		c.pot += c.sb
		u.gameInfo.roundBet = c.sb
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefSB
		c.handStartInfo.SB = &Bet{
			Action: ActionDefSB,
			Num:    c.sb,
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
	c.handStartInfo.SB = &Bet{
		Action: ActionDefAllIn,
		Num:    u.gameInfo.roundBet,
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

//bigBlind 大盲
func (c *Holdem) bigBlind() {
	u := c.button.nextAgent.nextAgent
	if u.gameInfo.status == ActionDefAllIn {
		c.log.Debug("big blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("allin", 0))
		return
	}
	if u.gameInfo.chip >= 2*c.sb {
		c.pot += c.sb * 2
		u.gameInfo.roundBet = c.sb * 2
		u.gameInfo.handBet += u.gameInfo.roundBet
		u.gameInfo.chip -= u.gameInfo.roundBet
		u.gameInfo.status = ActionDefBB
		c.handStartInfo.BB = &Bet{
			Action: ActionDefBB,
			Num:    c.sb * 2,
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
	c.handStartInfo.BB = &Bet{
		Action: ActionDefAllIn,
		Num:    u.gameInfo.roundBet,
	}
	c.log.Debug("small blind", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", u.gameInfo.roundBet))
}

func (c *Holdem) payToPlay() {
	u := c.button
	for {
		if u.gameInfo.te == PlayTypeAgreePayToPlay {
			c.pot += c.sb * 2
			u.gameInfo.roundBet = c.sb * 2
			u.gameInfo.handBet += u.gameInfo.roundBet
			u.gameInfo.chip -= u.gameInfo.roundBet
			u.gameInfo.status = ActionDefBB
			u.gameInfo.te = PlayTypeNormal
			c.handStartInfo.PayToPlay = append(c.handStartInfo.PayToPlay, u.gameInfo.seatNumber)
			c.log.Debug("pay to play", zap.Int8("seat", u.gameInfo.seatNumber), zap.Int("amount", c.sb*2))
		}
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
}

//preflop 翻牌前叫注
func (c *Holdem) preflop(op *Agent) ([]*Agent, bool) {
	c.roundBet = c.sb * 2
	c.minRaise = c.sb * 2
	u := op
	var roundComplete, showcard bool
	var unfoldUsers []*Agent
	c.log.Debug(RoundPreFlop.String()+" bet begin", zap.Int8("pc", c.playingPlayerCount), zap.Int8("sseat", u.gameInfo.seatNumber), zap.String("suser", u.user.ID()))
	for u != nil {
		c.log.Debug("wait bet", zap.Int8("seat", u.gameInfo.seatNumber), zap.String("status", u.gameInfo.status.String()), zap.String("round", RoundPreFlop.String()))
		bet := u.waitBet(c.roundBet, c.minRaise, RoundPreFlop, c.waitBetTimeout)
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

		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		var next *Agent
		var op *Operator
		if !roundComplete {
			next = c.getNextOperator(u)
			op = NewOperator(next, c.roundBet, c.minRaise)
		}
		thisAgent := u

		//稍微延迟告诉客户端可以下注
		time.AfterFunc(200*time.Millisecond, func() {
			thisAgent.recv.PlayerActionSuccess(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op)
			for r := range c.roomers {
				if r != thisAgent {
					r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op, r == next)
				}
			}
		})
		u = next
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
	c.log.Debug(RoundPreFlop.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//flopTurnRiver 叫注（轮描述）
func (c *Holdem) flopTurnRiver(u *Agent, round Round) ([]*Agent, bool) {
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
	c.log.Debug(round.String()+" bet begin", zap.Int8("pc", c.playingPlayerCount), zap.Int8("sseat", u.gameInfo.seatNumber), zap.String("suser", u.user.ID()))
	for u != nil {
		bet := u.waitBet(c.roundBet, c.minRaise, round, c.waitBetTimeout)
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
		roundComplete, unfoldUsers, showcard = c.checkRoundComplete()
		var next *Agent
		var op *Operator
		if !roundComplete {
			next = c.getNextOperator(u)
			op = NewOperator(next, c.roundBet, c.minRaise)
		}
		//稍微延迟告诉客户端可以下注
		thisAgent := u
		time.AfterFunc(200*time.Millisecond, func() {
			thisAgent.recv.PlayerActionSuccess(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op)
			for r := range c.roomers {
				if r != thisAgent {
					r.recv.RoomerGetAction(c.button.gameInfo.seatNumber, thisAgent.gameInfo.seatNumber, bet.Action, bet.Num, op, r == next)
				}
			}
		})
		u = next
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
	c.log.Debug(round.String()+" bet end", zap.Int("left", len(unfoldUsers)), zap.Bool("showcard", showcard))
	return unfoldUsers, showcard
}

//checkRoundComplete 判断是否叫注轮结束
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

func (c *Holdem) getNextOperator(u *Agent) *Agent {
	u = u.nextAgent
	for {
		if u.gameInfo.te == PlayTypeDisable || u.gameInfo.te == PlayTypeNeedPayToPlay {
			u = u.nextAgent
		}
		if u.gameInfo.status == ActionDefFold || u.gameInfo.status == ActionDefAllIn {
			u = u.nextAgent
		} else {
			return u
		}
	}
}

//deal 发牌
func (c *Holdem) deal() *Agent {
	cnt := 2
	c.log.Debug("deal begin", zap.Int("cards_count", 2))
	first := c.button.nextAgent
	cards := make([][]*Card, c.playingPlayerCount)
	max := cnt
	for ; max > 0; max-- {
		for i := 0; i < int(c.playingPlayerCount); i++ {
			cds, _ := c.poker.GetCards(1)
			if len(cards[i]) == 0 {
				cards[i] = make([]*Card, 0)
			}
			cards[i] = append(cards[i], cds...)
		}
	}
	cur := first
	seats := make([]int8, 0)
	for {
		seats = append(seats, cur.gameInfo.seatNumber)
		cur = cur.nextAgent
		if cur == first {
			break
		}
	}
	firstAg := c.getNextOperator(c.button.nextAgent.nextAgent)
	op := NewOperator(firstAg, 2*c.sb, 2*c.sb)
	//稍微延迟告诉客户端可以下注
	time.AfterFunc(200*time.Millisecond, func() {
		i := 0
		for {
			cur.gameInfo.cards = cards[i]
			cur.recv.PlayerGetCard(cur.gameInfo.seatNumber, cards[i], seats, int8(cnt), c.handStartInfo, op, cur == firstAg)
			i++
			cur = cur.nextAgent
			if cur == first {
				break
			}
		}
		for r := range c.roomers {
			if r.gameInfo == nil {
				r.recv.RoomerGetCard(seats, int8(cnt), c.handStartInfo, op)
			}
		}
	})
	c.log.Debug("deal end")
	return firstAg
}

//dealPublicCards 发公共牌
func (c *Holdem) dealPublicCards(n int, round Round) ([]*Card, *Agent) {
	c.log.Debug("deal public cards", zap.Int("cards_count", n))
	//洗牌
	_, _ = c.poker.GetCards(1)
	cards, _ := c.poker.GetCards(n)
	c.publicCards = append(c.publicCards, cards...)
	firstAg := c.getNextOperator(c.button)
	firstOp := NewOperator(firstAg, 0, 2*c.sb)
	time.AfterFunc(200*time.Millisecond, func() {
		for r := range c.roomers {
			r.recv.RoomerGetPublicCard(cards, firstOp, firstAg == r)
		}
	})
	return cards, firstAg
}

//complexWin 斗牌结算
func (c *Holdem) complexWin(users []*Agent) {
	pots, _ := c.calcPot(users)
	results, _, _ := c.calcWin(users, pots)
	c.pot = 0
	ret := make([]*Result, 0)
	u := c.button
	//所有玩家的最终状况
	for {
		r := &Result{
			SeatNumber: u.gameInfo.seatNumber,
		}
		if u.gameInfo.cardResults != nil {
			r.Cards = u.gameInfo.cardResults
			r.HandValueType = u.gameInfo.handValue.MaxHandValueType()
		}
		if rv, ok := results[u.gameInfo.seatNumber]; ok {
			u.gameInfo.chip += rv.Num
			r.Num = rv.Num
		}
		//保险
		if iv, ok := c.insuranceResult[u.gameInfo.seatNumber]; ok {
			r.InsuranceResult = iv
		}
		r.Chip = u.gameInfo.chip
		ret = append(ret, r)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	for r := range c.roomers {
		r.recv.RoomerGetResult(ret)
	}
	c.log.Debug("cwin", zap.Any("result", ret))
}

//simpleWin 单人获胜（只有一人未盖牌)
func (c *Holdem) simpleWin(agent *Agent) {
	ret := make([]*Result, 0)
	u := c.button
	for {
		r := &Result{
			SeatNumber: u.gameInfo.seatNumber,
		}
		if u.gameInfo.seatNumber == agent.gameInfo.seatNumber {
			u.gameInfo.chip += c.pot
			r.Num = c.pot
			c.pot = 0
		}
		//保险
		if iv, ok := c.insuranceResult[u.gameInfo.seatNumber]; ok {
			r.InsuranceResult = iv
		}
		r.Chip = u.gameInfo.chip
		ret = append(ret, r)
		u = u.nextAgent
		if u == c.button {
			break
		}
	}
	for r := range c.roomers {
		r.recv.RoomerGetResult(ret)
	}
	c.log.Debug("swin", zap.Int8("seat", agent.gameInfo.seatNumber), zap.String("user", agent.user.ID()), zap.Any("result", ret))
}

//StartHand 开始新的一手
func (c *Holdem) startHand() {
	c.pot = 0
	if c.ante > 0 {
		//前注
		c.doAnte()
	}
	c.publicCards = c.publicCards[:0]
	//洗牌
	c.poker.Reset()
	//下盲注
	c.smallBlind()
	c.bigBlind()
	//补盲
	c.payToPlay()
	//发牌（返回第一个行动的人）
	firstAg := c.deal()
	//翻牌前下注
	users, showcard := c.preflop(firstAg)
	//如果只有一个人翻牌游戏结束
	if len(users) == 1 {
		c.simpleWin(users[0])
		return
	}
	//洗牌,并发送3张公共牌
	_, firstAg = c.dealPublicCards(3, RoundFlop)
	//未亮牌要下注
	if !showcard {
		//翻牌轮下注
		users, showcard = c.flopTurnRiver(firstAg, RoundFlop)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//已亮牌并且有保险开始保险逻辑
	if showcard && c.insurance {
		//等待买保险
		c.insuranceStart(users, RoundFlop)
	}
	//洗牌,并发送1张公共牌
	var cards []*Card
	cards, firstAg = c.dealPublicCards(1, RoundTurn)
	//已亮牌并且有保险开始保险计算
	if showcard && c.insurance {
		//保险计算结果
		c.insuranceEnd(cards[0], RoundFlop)
	}
	//未亮牌要下注
	if !showcard {
		//转牌轮下注
		users, showcard = c.flopTurnRiver(firstAg, RoundTurn)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//已亮牌并且有保险开始保险逻辑
	if showcard && c.insurance {
		//等待买保险
		c.insuranceStart(users, RoundTurn)
	}
	//洗牌,并发送1张公共牌
	cards, firstAg = c.dealPublicCards(1, RoundRiver)
	//已亮牌并且有保险开始保险计算
	if showcard && c.insurance {
		//保险计算结果
		c.insuranceEnd(cards[0], RoundTurn)
	}
	//未亮牌要下注
	if !showcard {
		//河牌轮下注
		users, _ = c.flopTurnRiver(firstAg, RoundRiver)
		//如果只有一个人未盖牌游戏结束
		if len(users) == 1 {
			c.simpleWin(users[0])
			return
		}
	}
	//比牌计算结果
	c.complexWin(users)
}

//Start 开始游戏
func (c *Holdem) Start() {
	c.log.Debug("game start")
	c.isStarted = true
}

//Wait 等待开始
func (c *Holdem) Wait() {
	for {
		if !c.isStarted {
			continue
		}
		ok := c.buttonPosition()
		if !ok {
			c.log.Debug("players are not enough, wait")
		} else {
			c.isPlaying = true
			c.startHand()
			//清理座位用户
			c.seatLock.Lock()
			for i, r := range c.players {
				r.gameInfo.ResetForNextHand()
				if r.gameInfo.chip == 0 {
					c.log.Debug("less chip auto stand up", zap.Int8("seat", i), zap.String("user", r.user.ID()))
					c.standUp(i, r)
					continue
				}
				if r.gameInfo.needStandUp {
					c.log.Debug("user stand up", zap.Int8("seat", i), zap.String("user", r.user.ID()))
					c.standUp(i, r)
				}
			}
			c.isPlaying = false
			c.seatLock.Unlock()
		}
		next, wait := c.nextGame(c.handNum)
		if !next {
			c.isStarted = false
			c.log.Debug("game end")
			break
		}
		time.Sleep(wait)
	}
}
