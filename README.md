# holdem

模拟抽象了德州游戏

## Holdem 

游戏本身

- Wait() 等待主动开始，或者自动开局

- Start() 主动开始

```golang
h = holdem.NewHoldem(9, 100, 10, true, 2, 20*time.Second, nextGame, true, true, mp, 10*time.Second, 5*time.Second, holdem.NewNopRecorder(), log.With(zap.String("te", "server")))
go h.Wait()
```

启动协程来等待开始

## Agent

代理每一位参与游戏的人（上桌的和旁观的），这是一个虚拟的对象，实际实现的时候根据需要转换为实际需要的真实用户


```golang
NewAgent(recv Reciever, user UserInfo, log *zap.Logger) *Agent
```

创建一个代理，需要一个接收者，和一个用户信息的接口

----

### Reciever 接收者（接口）

接收游戏过程中所有玩家应该收到的信息，通过不同的接收者实现，来实现不同的处理。这个接收者无关通讯协议，只是把游戏信息及时传送，怎么传递给真实客户端，就通过不同的接收者实现来处理，test/example就是通过最简单的channel来实现通讯协议，真实CS架构则一般使用TCP来实现。

具体直接查看 [reciever.go] 接口定义来做自己的协议实现

----
### UserInfo 用户信息（接口）

它是一个抽象用户基础信息的接口,当你有真实的用户时候用它来携带用户信息，同时和游戏逻辑无关

```golang
type UserInfo interface {
	ID() string
	Name() string
	Avatar() string
	Info() map[string]string
}
```

----
### 主动行为（直接调用）

- ErrorOccur: 主动报告错误

- Join 加入游戏

- Info 获取游戏信息

- Leave 离开游戏

- BringIn 带入

- Seated 坐下

- StandUp 站起

- Bet 行动

- PayToPlay 补盲

### 总结

代理人通过主动行为来参与游戏，接收者接收游戏过程中的各种信息。

主动行为一定会有一个主动行为的结果反馈通过接收者反馈给代理人。

(本身设计的时候把【发送/接收】，异步分离了，所以可能在主动行为获取反馈方面会有点混乱，主要是为了做到尽可能的解耦)

## 记录器（接口）

Recorder是对游戏进程的记录，它也是一个接口，可以用不同的存储或者数据库来实时记录。你可以理解为它只是一个游戏记录的一个钩子(hook)而已

```golang
type Recorder interface {
    //开始
	Begin(*HoldemState)
    //前注
	Ante(seat int8, chip int, num int)
    //行动
	Action(round Round, seat int8, chip int, action ActionDef, num int)
    //保险
	InsureResult(round Round, seat int8, bet int, win float64)
    //结束
	End([]*Result)
}
```


