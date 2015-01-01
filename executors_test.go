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

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	NumMocks      = 3
	NumMocksLarge = 999
)

func newMockGenerator(sleepTime time.Duration, n int) *internalGenerator {
	return &internalGenerator{"mock generator", func() string {
		// Can't use time.Sleep():
		// 999 sleeping goroutines can execute concurrently.
		// 999 busy goroutines can only execute one-per-CPU.
		// This affects the benchmarks in a significant way.
		busySleep(sleepTime)
		return strconv.FormatInt(int64(n), 10)
	}}
}

func newMockGenerators(sleepTime time.Duration, n int) []*internalGenerator {
	generators := make([]*internalGenerator, n, n)

	for i := 0; i < n; i++ {
		generators[i] = newMockGenerator(sleepTime, i)
	}

	return generators
}

func createMocks(n int) []*internalGenerator {
	generators := make([]*internalGenerator, n, n)

	for i := 0; i < n; i++ {
		generators[i] = newMockGenerator(
			// Sleep for time proportional to index to ensure that results come in
			// out-of-order.
			time.Duration(n-i)*time.Millisecond,
			i)
	}

	return generators
}

func createNoopGenerators(n int) []*internalGenerator {
	generators := make([]*internalGenerator, n, n)

	for i := 0; i < n; i++ {
		generators[i] = &internalGenerator{"noop", func() string {
			return ""
		}}
	}

	return generators
}

func BenchmarkNoExecutorMultiGen(b *testing.B) {
	generators := createMocks(NumMocks)
	for i := 0; i < b.N; i++ {
		for j := 0; j < NumMocks; j++ {
			generators[j].Generate()
		}
	}
}

func BenchmarkNoExecutor(b *testing.B) {
	generator := newMockGenerator(6*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}

func TestSerialExecutor(t *testing.T) {
	executor := NewSerialExecutor()
	generators := createMocks(NumMocks)
	results := executor.Execute(generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func BenchmarkSerialExecutorMultiGen(b *testing.B) {
	executor := NewSerialExecutor()
	generators := createMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func BenchmarkSerialExecutor(b *testing.B) {
	executor := NewSerialExecutor()
	generator := newMockGenerator(2*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		executeGeneratorRepeatedly(executor, generator, NumMocks)
	}
}

func BenchmarkSerialNoop(b *testing.B) {
	executor := NewSerialExecutor()
	generators := createNoopGenerators(NumMocksLarge)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func TestForkJoinExecutor(t *testing.T) {
	executor := NewForkJoinExecutor()
	generators := createMocks(NumMocks)
	results := executor.Execute(generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func TestForkJoinExecutorLarge(t *testing.T) {
	executor := NewForkJoinExecutor()
	generator := newMockGenerator(1*time.Millisecond, 0)
	results := executeGeneratorRepeatedly(executor, generator, NumMocksLarge)
	assert.Len(t, results, NumMocksLarge)
}

func BenchmarkForkJoinExecutorMultiGen(b *testing.B) {
	executor := NewForkJoinExecutor()
	generators := createMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func BenchmarkForkJoinExecutor(b *testing.B) {
	executor := NewForkJoinExecutor()
	generator := newMockGenerator(2*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		executeGeneratorRepeatedly(executor, generator, NumMocks)
	}
}

func BenchmarkForkJoinNoop(b *testing.B) {
	executor := NewForkJoinExecutor()
	generators := createNoopGenerators(NumMocksLarge)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func AssertCorrectOrder(t *testing.T, n int, results string) {
	nums := make([]string, n, n)
	for i := 0; i < n; i++ {
		nums[i] = strconv.FormatInt(int64(i), 10)
	}

	assert.Equal(t, strings.Join(nums, ""), results)
}

// Spins the CPU for a duration.
func busySleep(dur time.Duration) {
	start := time.Now()
	for time.Now().Sub(start) < dur {
	}
}
