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
		hands := make([]*HandValue, 0)
		for j := 0; j < 9; j++ {
			cs2, _ := a.GetCards(2)
			allcs := append(cs, cs2...)
			hand, err := GetMaxHandValueFromCard(allcs)
			if err != nil {
				t.Error(err)
				return
			}
			str := hand.Sprint()
			str += hand.SprintTaggingCards(allcs)
			t.Log(str)
			hands = append(hands, hand)
		}
		t.Log("\n================================\n")
		maxHands := GetMaxHandValue(hands...)
		for _, hand := range maxHands {
			str := hand.Sprint()
			t.Log(str)
		}
		i--
		a.Reset()
	}

}
