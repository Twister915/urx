package urx

import "sync"

// The simple observable is simply a function which takes a subscriber and provides it with data
type simpleObservable struct {
	onSub *func(Subscriber)
}

// this creates a subscription (by calling the simpleObservable function immediately)
func (obs simpleObservable) privSubscribe() privSubscription {
	//first, create a subscriber/observer combo
	outSub := initSimpleSubscriber()
	outSub.mangleError = true
	f := *obs.onSub
	go func() {
		outSub.Notify(Start())
		f(outSub)
	}()
	return outSub
}

// applies an operator to the observable such that subscriptions to the resulting observable flow through the operator
func (obs simpleObservable) Lift(op Operator) (newObs privObservable) {
	newObs = &liftedObservable{obs, op}
	return
}

type simpleSubscriber struct {
	//notifications from source are written here
	out   chan Notification
	unsub chan interface{}
	//used to write up to a parent when an unsubscription
	hooks
	lock         sync.RWMutex
	mangleError  bool
	unsubscribed bool
	unsubClosed  bool
	extraLockers []sync.Locker
}

func initSimpleSubscriber() (out *simpleSubscriber) {
	out = new(simpleSubscriber)
	out.out = make(chan Notification)
	out.unsub = make(chan interface{})
	return
}

func (sub *simpleSubscriber) Events() <-chan Notification {
	return sub.out
}

func (sub *simpleSubscriber) IsSubscribed() bool {
	sub.RLock()
	defer sub.RUnlock()
	return !sub.unsubscribed
}

func (sub *simpleSubscriber) Unsubscribe() {
	sub.RLock()
	if sub.unsubscribed {
		sub.RUnlock()
		return
	}
	close(sub.unsub)
	sub.unsubClosed = true
	select {
	case sub.out <- Complete():
	default:
	}
	sub.RUnlock()
	sub.handleComplete()
}

func (sub *simpleSubscriber) Lock() {
	for i := range sub.extraLockers {
		sub.extraLockers[i].Lock()
	}
	sub.lock.Lock()
}

func (sub *simpleSubscriber) Unlock() {
	for i := range sub.extraLockers {
		sub.extraLockers[i].Unlock()
	}
	sub.lock.Unlock()
}

func (sub *simpleSubscriber) RLock() {
	sub.lock.RLock()
}

func (sub *simpleSubscriber) RUnlock() {
	sub.lock.RUnlock()
}

func (sub *simpleSubscriber) Notify(n Notification) {
	sub.RLock()
	if sub.unsubscribed || !sub.rawSend(n) {
		sub.RUnlock()
		return
	}
	if n.Type == OnError && sub.mangleError {
		n = Complete()
		if !sub.rawSend(n) {
			sub.RUnlock()
			return
		}
	}
	sub.RUnlock()
	if n.Type == OnComplete {
		sub.handleComplete()
	}
}

func (sub *simpleSubscriber) rawSend(n Notification) bool {
	select {
	case sub.out <- n:
		return true
	case <-sub.unsub:
		return false
	}
}

func (sub *simpleSubscriber) handleComplete() {
	sub.Lock()
	defer sub.Unlock()
	if sub.unsubscribed {
		return
	}
	sub.unsubscribed = true
	close(sub.out)
	if !sub.unsubClosed {
		close(sub.unsub)
		sub.unsubClosed = true
	}
	sub.callHooks()
}
