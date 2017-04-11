package urx

import (
	"testing"
	"time"
	"sync"
)

const count = 100

func TestMerge(t *testing.T) {
	one := createChanObs(10, time.Millisecond * 50).Map(func (in interface{}) interface{} {
		return in.(int) * -1
	})

	two := createChanObs(20, time.Millisecond * 25)

	var wg sync.WaitGroup
	values := make([][]int, count, count)
	wg.Add(count)
	for i := 0; i < count; i++ {
		s := &values[i]
		*s = make([]int, 0, 0)
		go func() {
			defer wg.Done()
			for i := range Merge(one, two).Subscribe().Values() {
				*s = append(*s, i.(int))
			}
		}()
	}
	wg.Wait()

	for i := range values {
		for i0 := range values {
			if i0 == i {
				continue
			}
			if len(values[i0]) != len(values[i]) {
				panic("non-equal slices")
			}
		}
	}
}
