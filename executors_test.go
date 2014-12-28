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
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	NumMocks      = 3
	NumMocksLarge = 999
)

func NewMockGenerator(sleepTime time.Duration, n int) *internalGenerator {
	return &internalGenerator{"mock generator", func(args *runtimeArgs) string {
		time.Sleep(sleepTime)
		return strconv.FormatInt(int64(n), 10)
	}}
}

func CreateMocks(n int) []*internalGenerator {
	generators := make([]*internalGenerator, n, n)

	for i := 0; i < n; i++ {
		generators[i] = NewMockGenerator(
			// Sleep for time proportional to index to ensure that results come in
			// out-of-order.
			time.Duration(n-i)*time.Millisecond,
			i)
	}

	return generators
}

func MockRuntimeArgs(executor GeneratorExecutor) *runtimeArgs {
	return &runtimeArgs{
		Rng: rand.New(rand.NewSource(1)),
	}
}

func BenchmarkNoExecutorMultiGen(b *testing.B) {
	generators := CreateMocks(NumMocks)
	for i := 0; i < b.N; i++ {
		for j := 0; j < NumMocks; j++ {
			generators[j].Generate(nil)
		}
	}
}

func BenchmarkNoExecutor(b *testing.B) {
	generator := NewMockGenerator(6*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		generator.Generate(nil)
	}
}

func TestSerialExecutor(t *testing.T) {
	executor := NewSerialExecutor()
	generators := CreateMocks(NumMocks)
	results := executor.Execute(MockRuntimeArgs(executor), generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func BenchmarkSerialExecutorMultiGen(b *testing.B) {
	executor := NewSerialExecutor()
	generators := CreateMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(MockRuntimeArgs(executor), generators)
	}
}

func BenchmarkSerialExecutor(b *testing.B) {
	executor := NewSerialExecutor()
	generator := NewMockGenerator(2*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		executeGeneratorRepeatedly(executor, MockRuntimeArgs(executor), generator, NumMocks)
	}
}

func TestForkJoinExecutor(t *testing.T) {
	executor := NewForkJoinExecutor()
	generators := CreateMocks(NumMocks)
	results := executor.Execute(MockRuntimeArgs(executor), generators)
	AssertCorrectOrder(t, NumMocks, results)
}

func TestForkJoinExecutorLarge(t *testing.T) {
	executor := NewForkJoinExecutor()
	generator := NewMockGenerator(1*time.Millisecond, 0)
	results := executeGeneratorRepeatedly(executor, MockRuntimeArgs(executor), generator, NumMocksLarge)
	assert.Len(t, results, NumMocksLarge)
}

func BenchmarkForkJoinExecutorMultiGen(b *testing.B) {
	executor := NewForkJoinExecutor()
	generators := CreateMocks(NumMocks)

	for i := 0; i < b.N; i++ {
		executor.Execute(MockRuntimeArgs(executor), generators)
	}
}

func BenchmarkForkJoinExecutor(b *testing.B) {
	executor := NewForkJoinExecutor()
	generator := NewMockGenerator(2*time.Millisecond, 0)

	for i := 0; i < b.N; i++ {
		executeGeneratorRepeatedly(executor, MockRuntimeArgs(executor), generator, NumMocks)
	}
}

func AssertCorrectOrder(t *testing.T, n int, results string) {
	nums := make([]string, n, n)
	for i := 0; i < n; i++ {
		nums[i] = strconv.FormatInt(int64(i), 10)
	}

	assert.Equal(t, strings.Join(nums, ""), results)
}
