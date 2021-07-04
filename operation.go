package holdem

type Operator struct {
	//ID
	ID string
	//SeatNumber 座位号
	SeatNumber int8
	//Chip 手上筹码
	Chip uint
	//BringIn 带入过多少
	BringIn uint
	//HandBet 本手已下注了多少
	HandBet uint
	//RoundBet 本轮下注了多少
	RoundBet uint
	//MinRaise 最小下注额度
	MinRaise uint
	//CurrentTableBet 当前轮桌上下注额
	CurrentTableBet uint
}

func newOperator(r *Agent, bet uint, minRaise uint) *Operator {
	if r == nil {
		return nil
	}
	return &Operator{
		ID:              r.recv.ID(),
		SeatNumber:      r.gameInfo.seatNumber,
		Chip:            r.gameInfo.chip,
		BringIn:         r.gameInfo.bringIn,
		HandBet:         r.gameInfo.handBet,
		RoundBet:        r.gameInfo.roundBet,
		MinRaise:        minRaise,
		CurrentTableBet: bet,
	}
}
