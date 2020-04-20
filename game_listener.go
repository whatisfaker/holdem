package holdem

type GameListener interface {
	BeforeBlinds(*Game)
	BeforePreFlop(*Game)
}
