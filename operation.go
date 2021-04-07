package holdem

type Operator struct {
	//SeatNumber 座位号
	SeatNumber int8
	//Chip 手上筹码
	Chip int
	//BringIn 带入过多少
	BringIn int
	//HandBet 本手已下注了多少
	HandBet int
	//RoundBet 本轮下注了多少
	RoundBet int
	//MinRaise 最小下注额度
	MinRaise int
	//CurrentTableBet 当前轮桌上下注额
	CurrentTableBet int
}

func NewOperator(r *Agent, bet int, minRaise int) *Operator {
	return &Operator{
		SeatNumber:      r.gameInfo.seatNumber,
		Chip:            r.gameInfo.chip,
		BringIn:         r.gameInfo.bringIn,
		HandBet:         r.gameInfo.handBet,
		RoundBet:        r.gameInfo.roundBet,
		MinRaise:        minRaise,
		CurrentTableBet: bet,
	}
}
