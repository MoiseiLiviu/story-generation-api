package channel_utils

import (
	"generate-script-lambda/application/ports/outbound"
	"sync"
)

func MergeChannels[T any](workerPool outbound.TaskDispatcher, channels ...<-chan T) (<-chan T, error) {
	var wg sync.WaitGroup
	merged := make(chan T)

	output := func(c <-chan T) {
		for val := range c {
			merged <- val
		}
		wg.Done()
	}

	wg.Add(len(channels))
	for _, c := range channels {
		ch := c
		err := workerPool.Submit(func() {
			output(ch)
		})
		if err != nil {
			return nil, err
		}
	}

	err := workerPool.Submit(func() {
		wg.Wait()
		close(merged)
	})
	if err != nil {
		return nil, err
	}

	return merged, nil
}
