package holdem

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const maxCardNum = 52

var pokerCards []*Card
var smOnce sync.Once
var CardOutOfIndex = fmt.Errorf("left cards count is less than expect")

type Poker struct {
	cards        []*Card
	currentIndex int
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
	if t > maxCardNum-1 {
		return nil, CardOutOfIndex
	}
	c.currentIndex = t
	return c.cards[c.currentIndex-n : c.currentIndex], nil
}

//State 当前排序，最大牌数
func (c *Poker) State() (int, int) {
	return c.currentIndex, len(c.cards)
}

func GetMaxHandValueFromCard(nc []*Card) (*HandValue, error) {
	switch len(nc) {
	case 5:
		return NewHandValue(nc)
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
		return getMaxHandValue(hands...)[0], nil
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
		return getMaxHandValue(hands...)[0], nil
	}
	return nil, fmt.Errorf("Unsupported card length")
}

//GetMaxHandValue 通过标记获得最大的标记牌型
func GetMaxHandValue(hvs map[int8]*HandValue) map[int8]*HandValue {
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

//getMaxHandValue 从一组牌型中找到最大的
func getMaxHandValue(hvs ...*HandValue) []*HandValue {
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
