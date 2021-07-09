package holdem

import (
	"time"
)

type extOptions struct {
	autoStandUpMaxHand      uint
	insuranceOpen           bool
	insuranceOdds           map[int]float64
	insuranceWaitTimeout    time.Duration
	recorder                Recorder
	isPayToPlay             bool
	ante                    uint
	medadata                map[string]interface{}
	autoStart               bool //是否自动开始
	autoMinPlayers          int8 //最懂最少开始人数
	minPlayers              int8 //最小游戏人数
	delayStandUpTimeout     time.Duration
	waitForNotEnoughPlayers time.Duration //人数不够等待时间
	limitDelayTimes         uint          //延迟操作限制次数
	limitAutoCheckTimes     uint          //自动check限制次数
	limitAutoFoldTimes      uint          //自动flod限制次数
}

type HoldemOption interface {
	apply(*extOptions)
}

type funcOption struct {
	f func(*extOptions)
}

func newFuncOption(f func(*extOptions)) *funcOption {
	return &funcOption{
		f: f,
	}
}

func (fo *funcOption) apply(do *extOptions) {
	fo.f(do)
}

func OptionInsurance(insuranceOdds map[int]float64, insuranceWaitTimeout time.Duration) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.insuranceOdds = insuranceOdds
		o.insuranceWaitTimeout = insuranceWaitTimeout
	})
}

//OptionAutoStandUp 是否托管状态打完站起
func OptionAutoStandUp(num uint) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.autoStandUpMaxHand = num
	})
}

func OptionCustomRecorder(rc Recorder) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.recorder = rc
	})
}

//OptionPayToPlay 开启补盲
func OptionPayToPlay() HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.isPayToPlay = true
	})
}

func OptionMetadata(metadata map[string]interface{}) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.medadata = metadata
	})
}

func OptionAnte(ante uint) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.ante = ante
	})
}

func OptionAutoStart(autoMinPlayers int8) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.autoStart = true
		o.autoMinPlayers = autoMinPlayers
	})
}

func OptionMinPlayers(minPlayers int8) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.minPlayers = minPlayers
	})
}

func OptionWaitForRebuy(dur time.Duration) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.delayStandUpTimeout = dur
	})
}

func OptionWaitForNotEnoughPlayers(dur time.Duration) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.waitForNotEnoughPlayers = dur
	})
}

func OptionLimitDelayTimes(times uint) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.limitDelayTimes = times
	})
}

func OptionLimitAutoCheckTimes(times uint) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.limitAutoCheckTimes = times
	})
}

func OptionLimitAutoFoldTimes(times uint) HoldemOption {
	return newFuncOption(func(o *extOptions) {
		o.limitAutoFoldTimes = times
	})
}
