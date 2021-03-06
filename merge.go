package urx

import "reflect"

func Merge(obs ...Observable) Observable {
	return Create(func(subscriber Subscriber) {
		subscriptions := make(map[Observable]Subscription)
		for i := range obs {
			subscriptions[obs[i]] = obs[i].Subscribe()
		}

		defer subscriber.Notify(Complete())
		subscriber.Add(func() {
			for _, sub := range subscriptions {
				if sub.IsSubscribed() {
					sub.Unsubscribe()
				}
			}
		})

		subscriber.Notify(Start())
		for {
			var selects []reflect.SelectCase
			var selectIdx []Observable
			for obs, sub := range subscriptions {
				selects = append(selects, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.Events())})
				selectIdx = append(selectIdx, obs)
			}

			from, val, ok := reflect.Select(selects)
			var notification Notification
			if !ok {
				notification = Complete()
			} else {
				notification, ok = val.Interface().(Notification)
				if !ok {
					panic("could not convert something known to be a notification to a notification")
				}
			}
			if notification.Type == OnComplete {
				delete(subscriptions, selectIdx[from])
				if len(subscriptions) > 0 {
					continue
				}
				return
			}
			if notification.Type == OnStart {
				continue
			}
			subscriber.Notify(notification)
		}
	})
}
