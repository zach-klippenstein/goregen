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
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	NumMocks      = 3
	NumMocksLarge = 999
)

type MockGenerator struct {
	SleepTime time.Duration
	N         int
}

func (gen MockGenerator) Generate() string {
	time.Sleep(gen.SleepTime)
	return strconv.FormatInt(int64(gen.N), 10)
}

func CreateMocks(n int) []Generator {
	generators := make([]Generator, n, n)
	for i := 0; i < n; i++ {
		generators[i] = MockGenerator{
			N: i,
			// Sleep for time proportional to index to ensure that results come in
			// out-of-order.
			SleepTime: time.Duration(n-i) * time.Millisecond,
		}
	}
	return generators
}

func BenchmarkNoExecutorMultiGen(b *testing.B) {
	generators := CreateMocks(NumMocks)
	for i := 0; i < b.N; i++ {
		for j := 0; j < NumMocks; j++ {
			generators[j].Generate()
		}
	}
}

func BenchmarkNoExecutor(b *testing.B) {
	generator := MockGenerator{
		SleepTime: 6 * time.Millisecond,
	}

	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}

func TestSerialExecutor(t *testing.T) {
	executor := NewSerialExecutor()
	generators := CreateMocks(NumMocks)
	results := executor.Execute(generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func BenchmarkSerialExecutorMultiGen(b *testing.B) {
	executor := NewSerialExecutor()
	generators := CreateMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func BenchmarkSerialExecutor(b *testing.B) {
	executor := NewSerialExecutor()
	generator := MockGenerator{
		SleepTime: 2 * time.Millisecond,
	}

	for i := 0; i < b.N; i++ {
		Execute(executor, generator, NumMocks)
	}
}

func TestForkJoinExecutor(t *testing.T) {
	executor := NewForkJoinExecutor()
	generators := CreateMocks(NumMocks)
	results := executor.Execute(generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func TestForkJoinExecutorLarge(t *testing.T) {
	executor := NewForkJoinExecutor()
	generator := MockGenerator{SleepTime: 1 * time.Millisecond}
	results := Execute(executor, generator, NumMocksLarge)
	assert.Len(t, results, NumMocksLarge)
}

func BenchmarkForkJoinExecutorMultiGen(b *testing.B) {
	executor := NewForkJoinExecutor()
	generators := CreateMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(generators)
	}
}

func BenchmarkForkJoinExecutor(b *testing.B) {
	executor := NewForkJoinExecutor()
	generator := MockGenerator{
		SleepTime: 2 * time.Millisecond,
	}

	for i := 0; i < b.N; i++ {
		Execute(executor, generator, NumMocks)
	}
}

func AssertCorrectOrder(t *testing.T, n int, results string) {
	nums := make([]string, n, n)
	for i := 0; i < n; i++ {
		nums[i] = strconv.FormatInt(int64(i), 10)
	}

	assert.Equal(t, strings.Join(nums, ""), results)
}
