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
	"bytes"
	"runtime"
	"strings"
	"sync"
)

/*
GeneratorExecutor runs a list of Generators and returns their results concatenated in order.
*/
type GeneratorExecutor interface {
	// Executes a list of generators and returns their in-order, concatenated results.
	Execute(generators []*internalGenerator) string
}

type serialExecutor struct{}

type forkJoinExecutor struct {
}

var numCpu = runtime.NumCPU()

// Execute executes a single generator n times.
func executeGeneratorRepeatedly(executor GeneratorExecutor, generator *internalGenerator, n int) string {
	generators := make([]*internalGenerator, n, n)

	for i := 0; i < n; i++ {
		generators[i] = generator
	}

	return executor.Execute(generators)
}

// NewSerialExecutor returns an executor that runs generators one after the other,
// on the current goroutine.
func NewSerialExecutor() GeneratorExecutor {
	return serialExecutor{}
}

func (serialExecutor) Execute(generators []*internalGenerator) string {
	var buffer bytes.Buffer
	numGens := len(generators)

	for i := 0; i < numGens; i++ {
		buffer.WriteString(generators[i].Generate())
	}

	return buffer.String()
}

/*
NewForkJoinExecutor returns an executor that runs each generator
on its own goroutine.

Protects its source using a mutex. See the Multi-threading section in the package documentation
for more information.
*/
func NewForkJoinExecutor() GeneratorExecutor {
	return forkJoinExecutor{}
}

func (forkJoinExecutor) Execute(generators []*internalGenerator) string {
	numGens := len(generators)
	results := make([]string, numGens, numGens)
	var waiter sync.WaitGroup

	waiter.Add(numGens)
	for i := 0; i < numGens; i++ {
		go func(i int) {
			defer waiter.Done()
			results[i] = generators[i].Generate()
		}(i)
	}
	waiter.Wait()

	return strings.Join(results, "")
}
