package holdem

type GameHook interface {
	BeforeBlinds(Game)
	BeforePreFlop(Game)
}
