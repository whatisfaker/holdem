package holdem

type PlayType int8

const (
	PlayTypeNormal         PlayType = iota //正常
	PlayTypeNeedPayToPlay                  //需要补盲
	PlayTypeAgreePayToPlay                 //同意补盲
	PlayTypeDisable                        //不能打牌
)

func (c PlayType) String() string {
	switch c {
	default:
		return "normal"
	case PlayTypeNeedPayToPlay:
		return "need bb"
	case PlayTypeAgreePayToPlay:
		return "aggree bb"
	case PlayTypeDisable:
		return "disable bb"
	}
}

type UserInfo interface {
	ID() string
	Name() string
	Avatar() string
	Info() map[string]string
}

type GameInfo struct {
	seatNumber  int8
	status      ActionDef
	needStandUp bool //需要离开
	roundBet    int
	handBet     int
	bringIn     int
	te          PlayType
	chip        int
	cards       []*Card
	handValue   *HandValue
	cardResults []*CardResult
	insurance   map[int8]*BuyInsurance //CardValue: buy
}

func (c *GameInfo) CalcHandValue(pc []*Card) {
	if c.handValue != nil {
		return
	}
	tmp := make([]*Card, 0, 7)
	tmp = append(tmp, pc...)
	tmp = append(tmp, c.cards...)
	c.handValue, _ = GetMaxHandValueFromCard(tmp)
	mp := c.handValue.TaggingCards(tmp)
	c.cardResults = make([]*CardResult, 0, 7)
	for i, v := range pc {
		c.cardResults = append(c.cardResults, NewCardResult(v, mp[i]))
	}
	c.cardResults = append(c.cardResults, NewCardResult(c.cards[0], mp[5]))
	c.cardResults = append(c.cardResults, NewCardResult(c.cards[1], mp[6]))
}

func (c *GameInfo) ResetForNextHand() {
	c.status = ActionDefNone
	c.roundBet = 0
	c.handBet = 0
	c.cards = nil
	c.handValue = nil
	c.cardResults = nil
}
