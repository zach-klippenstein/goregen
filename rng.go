/*
Copyright 2014 Zachary Klippenstein

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package regen

import "math/rand"

// The default Source implementation is very slow to seed. Replaced with a
// 64-bit xor-shift source from http://vigna.di.unimi.it/ftp/papers/xorshift.pdf.
// This source seeds very quickly, and only uses a single variable, so concurrent
// modification by multiple goroutines is possible.
type xorShift64Source struct {
	state uint64
}

func newXorShift64Source(seed int64) rand.Source {
	// a zero seed will only generate zeros.
	if seed == 0 {
		seed = 1
	}

	return &xorShift64Source{uint64(seed)}
}

func (src *xorShift64Source) Seed(seed int64) {
	src.state = uint64(seed)
}

func (src *xorShift64Source) Int63() int64 {
	x := src.state
	x ^= x >> 12 // a
	x ^= x << 25 // b
	x ^= x >> 27 // c
	src.state = x

	return int64((x * 2685821657736338717) >> 1)
}
