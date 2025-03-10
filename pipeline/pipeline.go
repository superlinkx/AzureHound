// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package pipeline

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/bloodhoundad/azurehound/internal"
)

type Result[T any] struct {
	Error error
	Ok    T
}

// OrDone provides an explicit cancellation mechanism to ensure the encapsulated and downstream goroutines are cleaned
// up. This frees the caller from depending on the input channel to close in order to free the goroutine, thus
// preventing possible leaks.
func OrDone[D, T any](done <-chan D, in <-chan T) <-chan T {
	out := make(chan T)

	go func() {
		defer close(out)
		for {
			select {
			case <-done:
				return
			case val, ok := <-in:
				if !ok {
					return
				} else {
					select {
					case out <- val:
					case <-done:
					}
				}
			}
		}
	}()
	return out
}

// Mux joins multiple channels and returns a channel as single stream of data.
func Mux[D any](done <-chan D, channels ...<-chan any) <-chan any {
	var wg sync.WaitGroup
	out := make(chan interface{})

	muxer := func(channel <-chan any) {
		defer wg.Done()
		for item := range OrDone(done, channel) {
			out <- item
		}
	}

	wg.Add(len(channels))
	for _, channel := range channels {
		go muxer(channel)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Demux distributes the stream of data from a single channel across multiple channels to parallelize CPU use and I/O
func Demux[D, T any](done <-chan D, in <-chan T, size int) []<-chan T {
	outputs := make([]chan T, size)

	for i := range outputs {
		outputs[i] = make(chan T)
	}

	closeOutputs := func() {
		for i := range outputs {
			close(outputs[i])
		}
	}

	cases := internal.Map(outputs, func(out chan T) reflect.SelectCase {
		return reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(out),
		}
	})

	go func() {
		defer closeOutputs()
		for item := range OrDone(done, in) {
			// send item to exactly one channel
			for i := range cases {
				cases[i].Send = reflect.ValueOf(item)
			}
			reflect.Select(cases)
		}
	}()

	return internal.Map(outputs, func(out chan T) <-chan T { return out })
}

func Map[D, T, U any](done <-chan D, in <-chan T, fn func(T) U) <-chan U {
	out := make(chan U)
	go func() {
		defer close(out)
		for item := range OrDone(done, in) {
			out <- fn(item)
		}
	}()
	return out
}

func Filter[D, T any](done <-chan D, in <-chan T, fn func(T) bool) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for item := range OrDone(done, in) {
			if fn(item) {
				out <- item
			}
		}
	}()
	return out
}

// Tee copies the stream of data from a single channel to zero or more channels
func Tee[D, T any](done <-chan D, in <-chan T, outputs ...chan<- T) {
	go func() {
		// Need to close outputs when goroutine exits to ensure we avoid deadlock
		defer func() {
			for i := range outputs {
				close(outputs[i])
			}
		}()

		for item := range OrDone(done, in) {
			for _, out := range outputs {
				select {
				case <-done:
				case out <- item:
				}
			}
		}
	}()
}

func Batch[D, T any](done <-chan D, in <-chan T, maxItems int, maxTimeout time.Duration) <-chan []T {
	out := make(chan []T)

	go func() {
		defer close(out)

		timeout := time.After(maxTimeout)
		var batch []T
		for {
			select {
			case <-done:
				if len(batch) > 0 {
					out <- batch
					batch = nil
				}
				return
			case item, ok := <-in:
				if !ok {
					if len(batch) > 0 {
						out <- batch
						batch = nil
					}
					return
				} else {
					// Add to batch
					batch = append(batch, item)

					// Flush if limit is reached
					if len(batch) >= maxItems {
						out <- batch
						batch = nil
						timeout = time.After(maxTimeout)
					}
				}
			case <-timeout:
				if len(batch) > 0 {
					out <- batch
					batch = nil
				}
				timeout = time.After(maxTimeout)
			}
		}
	}()

	return out
}

func FormatJson[D, T any](done <-chan D, in <-chan T) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		for item := range OrDone(done, in) {
			if bytes, err := json.Marshal(item); err != nil {
				panic(err)
			} else {
				out <- string(bytes)
			}
		}
	}()

	return out
}
