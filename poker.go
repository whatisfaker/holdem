package holdem

import (
	"math/rand"
	"sync"
	"time"

	"errors"
)

var pokerCards []*Card
var smOnce sync.Once
var ErrCardOutOfIndex = errors.New("left cards count is less than expect")
var ErrInvalidCardLength = errors.New("unsupported card length")

type Poker struct {
	cards        []*Card
	currentIndex int
	maxCards     int
}

func init() {
	pokerCards = make([]*Card, 0)
	smOnce.Do(func() {
		var i, j int8
		for j = 0; j < 4; j++ {
			for i = 2; i <= 14; i++ {
				pokerCards = append(pokerCards, &Card{
					Num:  i,
					Suit: j,
				})
			}
		}
	})
}

func NewPoker() *Poker {
	cards := make([]*Card, 0)
	cards = append(cards, pokerCards...)
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd.Shuffle(len(cards), func(i int, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
	return &Poker{
		cards:        cards,
		currentIndex: 0,
		maxCards:     len(cards),
	}
}

func newPokerWithExceptCardsAndNoShuffle(exceptCards []*Card) *Poker {
	exceptMap := make(map[int8]bool)
	for _, card := range exceptCards {
		exceptMap[card.Value()] = true
	}
	cards := make([]*Card, 0)
	for _, v := range pokerCards {
		if _, ok := exceptMap[v.Value()]; !ok {
			cards = append(cards, v)
		}
	}
	return &Poker{
		cards:        cards,
		currentIndex: 0,
		maxCards:     len(cards),
	}
}

func (c *Poker) Reset() {
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd.Shuffle(len(c.cards), func(i int, j int) {
		c.cards[i], c.cards[j] = c.cards[j], c.cards[i]
	})
	c.currentIndex = 0
}

//ResetAfterOffset 在某个位置以后的牌重置
func (c *Poker) ResetAfterOffset(offset int) {
	offset = offset % len(c.cards)
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd.Shuffle(len(c.cards)-offset-1, func(i int, j int) {
		c.cards[i+offset+1], c.cards[j+offset+1] = c.cards[j+offset+1], c.cards[i+offset+1]
	})
	c.currentIndex = offset + 1
}

func (c *Poker) GetCards(n int) ([]*Card, error) {
	t := c.currentIndex
	t += n
	if t > c.maxCards {
		return nil, ErrCardOutOfIndex
	}
	c.currentIndex = t
	return c.cards[c.currentIndex-n : c.currentIndex], nil
}

//State 当前排序，最大牌数
func (c *Poker) State() (int, int) {
	return c.currentIndex, len(c.cards)
}

type Outs struct {
	Len    int
	Detail map[int8]map[*Card]*HandValue //seat: card: handValue
}

type LeaderOuts struct {
	//SeatNumber 领先者座位号
	SeatNumber int8
	//Outs 对应多个池子的出路
	Outs map[*Pot]*Outs
}

//GetOuts 通过当前手牌，和预测手牌，计算每个分组中领先者对应的他人Outs
func GetOuts(allHands map[int8]*HandValue, allNextHands map[int8]map[*Card]*HandValue, pots []*Pot) map[int8]*LeaderOuts {
	leaderOuts := make(map[int8]*LeaderOuts)
	for _, p := range pots {
		pot := p
		groupSeats := pot.SeatNumber
		//存储所有参与当前池(pot)的当前手牌
		mp := make(map[int8]*HandValue)
		for s, v := range allHands {
			if _, ok := groupSeats[s]; ok {
				mp[s] = v
			} else {
				delete(allHands, s)
			}
		}
		//计算领先者
		max := GetMaxHandValueFromTaggedHandValues(mp)
		//去除领先者，留下其他人
		for s := range max {
			delete(groupSeats, s)
		}
		//开始比较
		for s := range max {
			//获得领先者的预测下手牌
			target := allNextHands[s]
			outs := &Outs{
				Len:    0,
				Detail: make(map[int8]map[*Card]*HandValue),
			}
			//根据每张可能牌去计算其他人的可能牌比较大小（计算outs)
			for cd := range target {
				mp2 := make(map[int8]*HandValue)
				for os := range groupSeats {
					mp2[os] = allNextHands[os][cd]
				}
				max2 := GetMaxHandValueFromTaggedHandValues(mp2)
				b := false
				for s, hands := range max2 {
					if hands.value >= target[cd].value {
						b = true
						_, ok := outs.Detail[s]
						if !ok {
							outs.Detail[s] = make(map[*Card]*HandValue)
						}
						outs.Detail[s][cd] = hands
					}
				}
				//相同卡 只算一个out
				if b {
					outs.Len++
				}
			}
			//归类到返回的领先者outs中（分池子）
			lo, ok := leaderOuts[s]
			if ok {
				lo.Outs[pot] = outs
			} else {
				lo = &LeaderOuts{
					SeatNumber: s,
					Outs: map[*Pot]*Outs{
						pot: outs,
					},
				}
			}
			leaderOuts[s] = lo
		}
	}
	return leaderOuts
}

//GetAllOuts 获取每一张补牌对应的最大手牌(当前最大手牌,和每发一张牌的最大手牌)
func GetAllOuts(publicCards []*Card, seatCards map[int8][]*Card) (map[int8]*HandValue, map[int8]map[*Card]*HandValue) {
	eCards := append(make([]*Card, 0), publicCards...)
	mp := make(map[int8]*HandValue)
	for s, v := range seatCards {
		eCards = append(eCards, v...)
		hv, _ := GetMaxHandValueFromCard(append(publicCards, v...))
		mp[s] = hv
	}
	poker := newPokerWithExceptCardsAndNoShuffle(eCards)
	pcs := make(map[int8]map[*Card]*HandValue)
	for {
		cards, err := poker.GetCards(1)
		if err != nil {
			break
		}
		card := cards[0]
		for seat, v := range seatCards {
			cds := append(publicCards, v...)
			cds = append(cds, card)
			ohv, _ := GetMaxHandValueFromCard(cds)
			cardsMap, ok := pcs[seat]
			if !ok {
				cardsMap = make(map[*Card]*HandValue)
			}
			cardsMap[card] = ohv
			pcs[seat] = cardsMap
		}
	}
	return mp, pcs
}

func GetHandValueFromCard(nc []*Card) ([]*HandValue, error) {
	switch len(nc) {
	case 5:
		nh, err := NewHandValue(nc)
		if err != nil {
			return nil, err
		}
		return []*HandValue{nh}, nil
	case 6:
		hands := make([]*HandValue, 0)
		err := comb(6, 5, func(out []int) error {
			nnc := make([]*Card, 0, 5)
			for i := 0; i < 5; i++ {
				nnc = append(nnc, nc[out[i]])
			}
			hand, err := NewHandValue(nnc)
			if err != nil {
				return err
			}
			hands = append(hands, hand)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return hands, nil
	case 7:
		hands := make([]*HandValue, 0)
		err := comb(7, 5, func(out []int) error {
			nnc := make([]*Card, 0, 5)
			for i := 0; i < 5; i++ {
				nnc = append(nnc, nc[out[i]])
			}
			hand, err := NewHandValue(nnc)
			if err != nil {
				return err
			}
			hands = append(hands, hand)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return hands, nil
	}
	return nil, ErrInvalidCardLength
}

func GetMaxHandValueFromCard(nc []*Card) (*HandValue, error) {
	hands, err := GetHandValueFromCard(nc)
	if err != nil {
		return nil, err
	}
	return GetMaxHandValue(hands...)[0], nil
}

//GetMaxHandValue 通过标记获得最大的标记牌型
func GetMaxHandValueFromTaggedHandValues(hvs map[int8]*HandValue) map[int8]*HandValue {
	var maxValue int64
	ret := make(map[int8]*HandValue)
	for _, v := range hvs {
		if v.value > maxValue {
			maxValue = v.value
		}
	}
	for num, v := range hvs {
		if v.value == maxValue {
			ret[num] = v
		}
	}
	return ret
}

//GetMaxHandValue 从一组牌型中找到最大的
func GetMaxHandValue(hvs ...*HandValue) []*HandValue {
	var maxValue int64
	ret := make([]*HandValue, 0)
	for _, v := range hvs {
		if v.value > maxValue {
			maxValue = v.value
		}
	}
	for _, v := range hvs {
		if v.value == maxValue {
			ret = append(ret, v)
		}
	}
	return ret
}

func comb(n, m int, emit func([]int) error) error {
	s := make([]int, m)
	last := m - 1
	var rc func(int, int) error
	rc = func(i, next int) error {
		for j := next; j < n; j++ {
			s[i] = j
			if i == last {
				err := emit(s)
				if err != nil {
					return err
				}
			} else {
				err := rc(i+1, j+1)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return rc(0, 0)
}
