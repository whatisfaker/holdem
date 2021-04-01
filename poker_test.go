package holdem

import (
	"testing"

	"gopkg.in/stretchr/testify.v1/assert"
)

func TestNewPoker(t *testing.T) {
	a := NewPoker()
	i := 2
	for i > 0 {
		cs, err := a.GetCards(5)
		if err != nil {
			t.Error(err)
			return
		}
		hands := make(map[int8]*HandValue)
		t.Log("\n============ hand =============\n")
		for j := 0; j < 9; j++ {
			cs2, _ := a.GetCards(2)
			allcs := append(cs, cs2...)
			hand, err := GetMaxHandValueFromCard(allcs)
			if err != nil {
				t.Error(err)
				return
			}
			str := hand.String()
			str += hand.DebugCards(allcs)
			t.Log(str)
			hands[int8(j)] = hand
		}
		t.Log("\n============ max hand =============\n")
		maxHands := GetMaxHandValueFromTaggedHandValues(hands)
		for _, hand := range maxHands {
			t.Log(hand, hand.Value())
		}
		i--
		a.Reset()
	}
}

func TestCards(t *testing.T) {
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)

	a1, _ := NewCard(13, 0)
	a2, _ := NewCard(14, 2)
	b1, _ := NewCard(2, 1)
	b2, _ := NewCard(5, 0)

	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5, a1, a2})
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5, b1, b2})
	t.Log(r1.String())
	t.Log(r2.String())

	max := GetMaxHandValue(r1, r2)

	t.Log(max[0].String())

}

func TestHVHighCard(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(7, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVHighCard)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(8, 1)
	c3, _ = NewCard(4, 0)
	c4, _ = NewCard(5, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVHighCard)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVOnePair(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVOnePair)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(4, 1)
	c3, _ = NewCard(4, 0)
	c4, _ = NewCard(5, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVOnePair)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVTwoPair(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(12, 0)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVTwoPair)

	c1, _ = NewCard(11, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(13, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVTwoPair)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVThreeOfAKind(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(2, 1)
	c3, _ := NewCard(2, 0)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVThreeOfAKind)

	c1, _ = NewCard(11, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(11, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVThreeOfAKind)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVStraight(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 0)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 3)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraight)

	c1, _ = NewCard(10, 2)
	c2, _ = NewCard(11, 1)
	c3, _ = NewCard(12, 0)
	c4, _ = NewCard(13, 1)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVStraight)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 1)
	c2, _ := NewCard(7, 1)
	c3, _ := NewCard(4, 1)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFlush)

	c1, _ = NewCard(3, 2)
	c2, _ = NewCard(7, 2)
	c3, _ = NewCard(4, 2)
	c4, _ = NewCard(5, 2)
	c5, _ = NewCard(14, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVFullHouse(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(3, 1)
	c2, _ := NewCard(3, 2)
	c3, _ := NewCard(3, 0)
	c4, _ := NewCard(10, 1)
	c5, _ := NewCard(10, 2)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFullHouse)

	c1, _ = NewCard(3, 1)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(3, 0)
	c4, _ = NewCard(14, 0)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFullHouse)
	assert.Equal(r2.Value() > r1.Value(), true)
}
func TestHVFourOfAKind(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(3, 1)
	c2, _ := NewCard(3, 2)
	c3, _ := NewCard(3, 0)
	c4, _ := NewCard(3, 3)
	c5, _ := NewCard(13, 2)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVFourOfAKind)

	c1, _ = NewCard(3, 1)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(3, 0)
	c4, _ = NewCard(3, 3)
	c5, _ = NewCard(14, 3)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVFourOfAKind)
	assert.Equal(r2.Value() > r1.Value(), true)
}
func TestHVStraightFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(2, 1)
	c2, _ := NewCard(3, 1)
	c3, _ := NewCard(4, 1)
	c4, _ := NewCard(5, 1)
	c5, _ := NewCard(14, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraightFlush)

	c1, _ = NewCard(2, 2)
	c2, _ = NewCard(3, 2)
	c3, _ = NewCard(4, 2)
	c4, _ = NewCard(5, 2)
	c5, _ = NewCard(6, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVStraightFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}

func TestHVRoyalFlush(t *testing.T) {
	assert := assert.New(t)
	c1, _ := NewCard(9, 1)
	c2, _ := NewCard(10, 1)
	c3, _ := NewCard(11, 1)
	c4, _ := NewCard(12, 1)
	c5, _ := NewCard(13, 1)
	r1, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r1.MaxHandValueType(), HVStraightFlush)

	c1, _ = NewCard(10, 2)
	c2, _ = NewCard(11, 2)
	c3, _ = NewCard(12, 2)
	c4, _ = NewCard(13, 2)
	c5, _ = NewCard(14, 2)
	r2, _ := GetMaxHandValueFromCard([]*Card{c1, c2, c3, c4, c5})
	assert.Equal(r2.MaxHandValueType(), HVRoyalFlush)
	assert.Equal(r2.Value() > r1.Value(), true)
}
