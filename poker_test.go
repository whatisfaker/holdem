package holdem

import "testing"

func TestNewPoker(t *testing.T) {
	a := NewPoker()
	i := 1
	for i > 0 {
		cs, err := a.GetCards(5)
		if err != nil {
			t.Error(err)
			return
		}
		hands := make(map[int8]*HandValue)
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
		t.Log("\n================================\n")
		maxHands := GetMaxHandValueFromTaggedHandValues(hands)
		for _, hand := range maxHands {
			t.Log(hand)
		}
		i--
		a.Reset()
	}
}

func TestCards(t *testing.T) {
	c1, _ := NewCard(2, 2)
	c2, _ := NewCard(14, 1)
	c3, _ := NewCard(14, 0)
	c4, _ := NewCard(4, 1)
	c5, _ := NewCard(9, 3)

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
