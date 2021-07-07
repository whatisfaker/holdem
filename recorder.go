package holdem

type Recorder interface {
	GameStart(*HoldemBase)
	HandBegin(*HoldemState)
	Ante(base *HoldemBase, seat int8, id string, chip uint, num uint)
	Action(base *HoldemBase, round Round, seat int8, id string, chip uint, action ActionDef, num uint)
	InsureResult(base *HoldemBase, round Round, seat int8, id string, bet uint, win float64)
	HandEnd(state *HoldemState, r []*Result)
	GameEnd(base *HoldemBase)
}

type NopRecorder struct {
}

func newNopRecorder() Recorder {
	return &NopRecorder{}
}

var _ Recorder = (*NopRecorder)(nil)

func (c *NopRecorder) GameStart(*HoldemBase) {}

func (c *NopRecorder) GameEnd(*HoldemBase) {}

func (c *NopRecorder) HandBegin(*HoldemState) {}

func (c *NopRecorder) Ante(meta *HoldemBase, seat int8, id string, chip uint, num uint) {
}

func (c *NopRecorder) Action(meta *HoldemBase, round Round, seat int8, id string, chip uint, action ActionDef, num uint) {
}

func (c *NopRecorder) InsureResult(meta *HoldemBase, round Round, seat int8, id string, bet uint, win float64) {
}

func (c *NopRecorder) HandEnd(state *HoldemState, r []*Result) {}
