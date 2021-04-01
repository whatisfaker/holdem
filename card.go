package holdem

import (
	"errors"
	"fmt"
)

//HandValueType 手牌牌型类型
//https://zh.wikipedia.org/wiki/%E5%BE%B7%E5%B7%9E%E6%92%B2%E5%85%8B#%E7%89%8C%E5%9E%8B%E5%A4%A7%E5%B0%8F%E8%A7%84%E5%88%99
type HandValueType int8

const (
	HVHighCard HandValueType = iota + 1
	HVOnePair
	HVTwoPair
	HVThreeOfAKind
	HVStraight
	HVFlush
	HVFullHouse
	HVFourOfAKind
	HVStraightFlush
	HVRoyalFlush
)

func (c HandValueType) String() string {
	if c == 1 {
		return "高牌(high card)"
	}
	if c == 2 {
		return "一对(one pair)"
	}
	if c == 3 {
		return "两对(two pairs)"
	}
	if c == 4 {
		return "三条(three of a kind)"
	}
	if c == 5 {
		return "顺子(straight)"
	}
	if c == 6 {
		return "同花(flush)"
	}
	if c == 7 {
		return "葫芦(full house)"
	}
	if c == 8 {
		return "四条(four of a kind)"
	}
	if c == 9 {
		return "同花顺(straight flush)"
	}
	if c == 10 {
		return "皇家同花顺(royal flush)"
	}
	return "Unknonw Hand Value Type"
}

var (
	cardMap                 = map[int8]string{2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "10", 11: "J", 12: "Q", 13: "K", 14: "A"}
	suitMap                 = [4]string{"♠", "♥", "♣", "♦"}
	ErrInvalidCard          = errors.New("invalid card num(2-14)/suit(0-3)")
	ErrInvalidHandValueType = errors.New("invalid hand value type")
)

type Card struct {
	Num  int8 //2-14
	Suit int8 //0-3
}

func NewCard(num int8, suit int8) (*Card, error) {
	if num < 2 || num > 14 {
		return nil, ErrInvalidCard
	}
	if suit < 0 || suit > 3 {
		return nil, ErrInvalidCard
	}
	return &Card{
		Num:  num,
		Suit: suit,
	}, nil
}

func (c Card) SuitString() string {
	return suitMap[c.Suit]
}

func (c Card) NumString() string {
	return cardMap[c.Num]
}

//HandValue 手牌
type HandValue struct {
	cards            [5]*Card
	value            int64
	maxHandValueType HandValueType
}

//NewHandValue 创建手牌（已计算最高牌型）
func NewHandValue(nc []*Card) (*HandValue, error) {
	if len(nc) != 5 {
		return nil, errors.New("cards length is not 5")
	}
	var a [5]*Card
	copy(a[:], nc)
	t := &HandValue{
		cards: a,
	}
	t.evaluate()
	return t, nil
}

func (c *HandValue) Cards() [5]*Card {
	return c.cards
}

func (c *HandValue) MaxHandValueType() HandValueType {
	return c.maxHandValueType
}

func (c *HandValue) Value() int64 {
	return c.value
}

func (c *HandValue) HasCards(nc ...*Card) bool {
	ret := make(map[string]bool)
	for _, v := range nc {
		key := fmt.Sprintf("%d-%d", v.Suit, v.Num)
		ret[key] = false
	}
	for _, v := range c.cards {
		key := fmt.Sprintf("%d-%d", v.Suit, v.Num)
		if _, ok := ret[key]; ok {
			ret[key] = true
		}
	}
	for _, v := range ret {
		if !v {
			return false
		}
	}
	return true
}

func (c *HandValue) TaggingCards(nc []*Card) map[int]bool {
	ret := make(map[*Card]bool)
	for _, v := range nc {
		ret[v] = false
	}
	for _, v := range c.cards {
		ret[v] = true
	}
	ret2 := make(map[int]bool)
	for i, v := range nc {
		ret2[i] = ret[v]
	}
	return ret2
}

func (c *HandValue) DebugCards(nc []*Card) string {
	mp := c.TaggingCards(nc)
	var str string = "\n"
	for i := range nc {
		if i > 0 {
			str += "\t"
		}
		str += nc[i].NumString()
	}
	str += "\n"
	for i := range nc {
		if i > 0 {
			str += "\t"
		}
		str += nc[i].SuitString()
	}
	str += "\n"
	for i := range nc {
		if i > 0 {
			str += "\t"
		}
		if mp[i] {
			str += "●"
		} else {
			str += "○"
		}
	}
	str += "\n"
	str += fmt.Sprintf("%d\n", c.value)
	return str
}

func (c *HandValue) String() string {
	return fmt.Sprintf("%s(%s) - %s(%s) - %s(%s) - %s(%s) - %s(%s) : %s", c.cards[0].NumString(), c.cards[0].SuitString(), c.cards[1].NumString(), c.cards[1].SuitString(), c.cards[2].NumString(), c.cards[2].SuitString(), c.cards[3].NumString(), c.cards[3].SuitString(), c.cards[4].NumString(), c.cards[4].SuitString(), c.maxHandValueType.String())
}

func (c *HandValue) caculateValue() {
	switch c.maxHandValueType {
	case HVHighCard:
		c.value = int64(c.cards[0].Num)<<16 + int64(c.cards[1].Num)<<12 + int64(c.cards[2].Num)<<8 + int64(c.cards[3].Num)<<4 + int64(c.cards[4].Num)
	case HVOnePair:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<12 + int64(c.cards[2].Num)<<8 + int64(c.cards[3].Num)<<4 + int64(c.cards[4].Num)
	case HVTwoPair:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<8 + int64(c.cards[3].Num)<<4 + int64(c.cards[4].Num)
	case HVThreeOfAKind:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<8 + int64(c.cards[3].Num)<<4 + int64(c.cards[4].Num)
	case HVStraight:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)
	case HVFlush:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<16 + int64(c.cards[1].Num)<<12 + int64(c.cards[2].Num)<<8 + int64(c.cards[3].Num)<<4 + int64(c.cards[4].Num)
	case HVFullHouse:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<4 + int64(c.cards[3].Num)
	case HVFourOfAKind:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)<<4 + int64(c.cards[4].Num)
	case HVStraightFlush:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)
	case HVRoyalFlush:
		c.value = int64(c.maxHandValueType)<<20 + int64(c.cards[0].Num)
	default:
		panic(ErrInvalidHandValueType)
	}
}

func (c *HandValue) evaluate() {
	c.generalCheck()
	if c.maxHandValueType == HVFlush || c.maxHandValueType == HVHighCard {
		isStraight := c.isStraight()
		//Flush more logical
		if c.maxHandValueType == HVFlush && isStraight {
			if c.cards[0].Num == 14 {
				c.maxHandValueType = HVRoyalFlush
			} else {
				c.maxHandValueType = HVStraightFlush
			}
			c.caculateValue()
			return
		}
		if isStraight {
			c.maxHandValueType = HVStraight
			c.caculateValue()
		}
	}
}

func (c *HandValue) isStraight() bool {
	var i int8
	if c.cards[0].Num == 14 && c.cards[1].Num == 5 && c.cards[2].Num == 4 && c.cards[3].Num == 3 && c.cards[4].Num == 2 {
		c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[1], c.cards[2], c.cards[3], c.cards[4], c.cards[0]
		return true
	}
	for i = 1; i < 5; i++ {
		if (c.cards[0].Num - c.cards[i].Num) != i {
			return false
		}
	}
	return true
}

func (c *HandValue) generalCheck() {
	ppairs := make(map[*Card]int8)
	sameValues := make(map[int8]bool)
	isFlush := true
	for i := 0; i < 5; i++ {
		for j := i + 1; j < 5; j++ {
			if isFlush && c.cards[i].Suit != c.cards[j].Suit {
				isFlush = false
			}
			if c.cards[i].Num < c.cards[j].Num {
				c.cards[i], c.cards[j] = c.cards[j], c.cards[i]
				continue
			}
			if c.cards[i].Num == c.cards[j].Num {
				ppairs[c.cards[i]] = c.cards[i].Num
				ppairs[c.cards[j]] = c.cards[j].Num
				sameValues[c.cards[i].Num] = true
				if c.cards[i].Suit > c.cards[j].Suit {
					c.cards[i], c.cards[j] = c.cards[j], c.cards[i]
				}
			}
		}
	}
	if isFlush {
		c.maxHandValueType = HVFlush
		c.caculateValue()
		return
	}
	l := len(ppairs)
	if l == 0 {
		c.maxHandValueType = HVHighCard
		c.caculateValue()
		return
	}
	sl := len(sameValues)
	var oneSameValue int8
	for k := range sameValues {
		oneSameValue = k
		break
	}
	//OnePair
	if l == 2 {
		if c.cards[1].Num == oneSameValue && c.cards[2].Num == oneSameValue {
			c.cards[0], c.cards[1], c.cards[2] = c.cards[1], c.cards[2], c.cards[0]
		} else if c.cards[2].Num == oneSameValue && c.cards[3].Num == oneSameValue {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3] = c.cards[2], c.cards[3], c.cards[0], c.cards[1]
		} else if c.cards[3].Num == oneSameValue && c.cards[4].Num == oneSameValue {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[3], c.cards[4], c.cards[0], c.cards[1], c.cards[2]
		}
		c.maxHandValueType = HVOnePair
		c.caculateValue()
		return
	}
	//Two Pair
	if l == 4 && sl == 2 {
		if c.cards[0].Num != c.cards[1].Num {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[1], c.cards[2], c.cards[3], c.cards[4], c.cards[0]
		} else if c.cards[2].Num != c.cards[3].Num {
			c.cards[2], c.cards[3], c.cards[4] = c.cards[3], c.cards[4], c.cards[2]
		}
		c.maxHandValueType = HVTwoPair
		c.caculateValue()
		return
	}
	//Three of A kind
	if l == 3 {
		if c.cards[0].Num != oneSameValue && c.cards[1].Num != oneSameValue {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[2], c.cards[3], c.cards[4], c.cards[0], c.cards[1]
		} else if c.cards[0].Num != oneSameValue && c.cards[1].Num == oneSameValue {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3] = c.cards[1], c.cards[2], c.cards[3], c.cards[0]
		}
		c.maxHandValueType = HVThreeOfAKind
		c.caculateValue()
		return
	}
	//Four of a kind
	if l == 4 && sl == 1 {
		if c.cards[0].Num != oneSameValue {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[1], c.cards[2], c.cards[3], c.cards[4], c.cards[0]
		}
		c.maxHandValueType = HVFourOfAKind
		c.caculateValue()
		return
	}
	//Full House
	if l == 5 {
		if c.cards[0].Num > c.cards[2].Num {
			c.cards[0], c.cards[1], c.cards[2], c.cards[3], c.cards[4] = c.cards[2], c.cards[3], c.cards[4], c.cards[0], c.cards[1]
		}
		c.maxHandValueType = HVFullHouse
		c.caculateValue()
		return
	}
}
