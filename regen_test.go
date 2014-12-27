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
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zach-klippenstein/goregen/util"
	"math/rand"
	"regexp"
	"testing"
)

// Each expression is generated and validated this many times.
const SampleSize = 999

func ExampleGenerate() {
	pattern := "[ab]{5}"
	str, _ := Generate(pattern)

	if matched, _ := regexp.MatchString(pattern, str); matched {
		fmt.Println("Matches!")
	}
	// Output:
	// Matches!
}

func ExampleNewGenerator() {
	pattern := "[ab]{5}"
	args := &GeneratorArgs{
		Rng: rand.New(rand.NewSource(0)),
	}
	generator, _ := NewGenerator(pattern, args)
	str := generator.Generate()

	if matched, _ := regexp.MatchString(pattern, str); matched {
		fmt.Println("Matches!")
	}
	// Output:
	// Matches!
}

func TestEmpty(t *testing.T) {
	args := &GeneratorArgs{
		Rng: rand.New(rand.NewSource(0)),
	}
	AssertGenerates(t, args, "^$", "")
}

func TestLiteralSingleChar(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"a",
		"abc",
	)
}

func TestDot(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, ".")
}

func TestQuestionMark(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"a?",
		"(abc)?",
		"[ab]?",
		".?",
	)
}

func TestPlus(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, "a+")
}

func TestStar(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, "a*")
}

func TestCharClass(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"[a]",
		"[abc]",
		"[a-d]",
		"[ac]",
		"[0-9]",
		"[a-z0-9]",
	)
}

func TestNegativeCharClass(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"[^a-zA-Z0-9]",
	)
}

func TestOr(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"a|b",
		"abc|def|ghi",
		"foo|bar|baz", // rewrites to foo|ba[rz]
	)
}

func TestCapture(t *testing.T) {
	t.Parallel()
	args := &GeneratorArgs{
		Rng: rand.New(rand.NewSource(0)),
	}
	AssertGenerates(t, args, "abc", "(abc)")
	AssertGenerates(t, args, "", "()")
}

func TestConcat(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t,
		"[ab][cd]",
	)
}

func TestRepeatHitsMin(t *testing.T) {
	t.Parallel()
	regexp := "a{0,3}"
	var counts [4]int
	args := &GeneratorArgs{util.NewRand(0)}
	generator, _ := NewGenerator(regexp, args)

	for i := 0; i < SampleSize; i++ {
		str := generator.Generate()
		counts[len(str)]++
	}

	t.Log("counts:")
	for i, count := range counts {
		t.Logf("%d: %d", i, count)
	}

	assert.True(t, counts[0] > 0)
}

func TestRepeatHitsMax(t *testing.T) {
	t.Parallel()
	regexp := "a{0,3}"
	var counts [4]int
	args := &GeneratorArgs{util.NewRand(0)}
	generator, _ := NewGenerator(regexp, args)

	for i := 0; i < SampleSize; i++ {
		str := generator.Generate()
		counts[len(str)]++
	}

	t.Log("counts:")
	for i, count := range counts {
		t.Logf("%d: %d", i, count)
	}

	assert.True(t, counts[3] > 0)
}

func AssertGeneratesAndMatches(t *testing.T, patterns ...string) {
	args := &GeneratorArgs{
		Rng: util.NewRand(0),
	}

	for _, pattern := range patterns {
		AssertGenerates(t, args, pattern, pattern)
	}
}

func AssertGenerates(t *testing.T, args *GeneratorArgs, expectedPattern string, pattern string) {
	AssertGeneratesTimes(t, args, expectedPattern, pattern, SampleSize)
}

func AssertGeneratesTimes(t *testing.T, args *GeneratorArgs, expectedPattern string, pattern string, times int) {
	generator, _ := NewGenerator(pattern, args)

	for i := 0; i < times; i++ {
		result := generator.Generate()
		matched, err := regexp.MatchString(expectedPattern, result)
		if err != nil {
			panic(err)
		}
		assert.True(t, matched, "string generated from pattern /%s/ did not match /%s/", pattern, expectedPattern)
	}
}
