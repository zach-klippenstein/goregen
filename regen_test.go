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
	"math/rand"
	"regexp"
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	generator, _ := NewGenerator(pattern, &GeneratorArgs{
		RngSource: rand.NewSource(0),
	})

	str := generator.Generate()

	if matched, _ := regexp.MatchString(pattern, str); matched {
		fmt.Println("Matches!")
	}
	// Output:
	// Matches!
}

func ExampleNewGenerator_perl() {
	pattern := `\d{5}`

	generator, _ := NewGenerator(pattern, &GeneratorArgs{
		Flags: syntax.Perl,
	})

	str := generator.Generate()

	if matched, _ := regexp.MatchString("[[:digit:]]{5}", str); matched {
		fmt.Println("Matches!")
	}
	// Output:
	// Matches!
}

func TestNilArgs(t *testing.T) {
	generator, err := NewGenerator("", nil)
	assert.NotNil(t, generator)
	assert.Nil(t, err)
}

func TestNilArgValues(t *testing.T) {
	generator, err := NewGenerator("", &GeneratorArgs{})
	assert.NotNil(t, generator)
	assert.Nil(t, err)
}

func TestEmpty(t *testing.T) {
	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
	}
	AssertGenerates(t, args, "^$", "")
}

func TestLiteralSingleChar(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"a",
		"abc",
	)
}

func TestDotNotNl(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil, ".")

	generator, _ := NewGenerator(".", nil)

	// Not a very strong assertion, but not sure how to do better. Exploring the entire
	// generation space (2^32) takes far too long for a unit test.
	for i := 0; i < SampleSize; i++ {
		assert.NotEqual(t, "\n", generator.Generate())
	}
}

func TestQuestionMark(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"a?",
		"(abc)?",
		"[ab]?",
		".?",
	)
}

func TestPlus(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil, "a+")
}

func TestStar(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil, "a*")
}

func TestCharClassNotNl(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"[a]",
		"[abc]",
		"[a-d]",
		"[ac]",
		"[0-9]",
		"[a-z0-9]",
	)

	// Try to narrow down the generation space. Still not a very strong assertion.
	generator, _ := NewGenerator("[^a-zA-Z0-9]", nil)
	for i := 0; i < SampleSize; i++ {
		assert.NotEqual(t, "\n", generator.Generate())
	}
}

func TestNegativeCharClass(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"[^a-zA-Z0-9]",
	)
}

func TestAlternate(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"a|b",
		"abc|def|ghi",
		"[ab]|[cd]",
		"foo|bar|baz", // rewrites to foo|ba[rz]
	)
}

func TestCapture(t *testing.T) {
	t.Parallel()
	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
	}
	AssertGenerates(t, args, "abc", "(abc)")
	AssertGenerates(t, args, "", "()")
}

func TestConcat(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"[ab][cd]",
	)
}

func TestUnboundedRepeat(t *testing.T) {
	if testing.Short() {
		t.Skip("unbounded repeat can take ~15 seconds, skipping in short mode")
	}

	t.Parallel()
	args := &GeneratorArgs{RngSource: rand.NewSource(0)}
	AssertGeneratesAndMatches(t, args, `a{1,}`)
}

func BenchmarkLargeRepeatCreateSerial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewGenerator(`a{999}`, &GeneratorArgs{
			RngSource: rand.NewSource(0),
			Executor:  NewSerialExecutor(),
		})
	}
}

func BenchmarkLargeRepeatGenerateSerial(b *testing.B) {
	generator, err := NewGenerator(`a{999}`, &GeneratorArgs{
		RngSource: rand.NewSource(0),
		Executor:  NewSerialExecutor(),
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}

func BenchmarkLargeRepeatCreateForkJoin(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewGenerator(`a{999}`, &GeneratorArgs{
			RngSource: rand.NewSource(0),
			Executor:  NewForkJoinExecutor(),
		})
	}
}

func BenchmarkLargeRepeatGenerateForkJoin(b *testing.B) {
	generator, err := NewGenerator(`a{999}`, &GeneratorArgs{
		RngSource: rand.NewSource(0),
		Executor:  NewForkJoinExecutor(),
	})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		generator.Generate()
	}
}

func TestRepeatHitsMin(t *testing.T) {
	t.Parallel()
	regexp := "a{0,3}"
	var counts [4]int
	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
	}
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
	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
	}
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

func TestAsciiCharClasses(t *testing.T) {
	t.Parallel()
	AssertGeneratesAndMatches(t, nil,
		"[[:alnum:]]",
		"[[:alpha:]]",
		"[[:ascii:]]",
		"[[:blank:]]",
		"[[:cntrl:]]",
		"[[:digit:]]",
		"[[:graph:]]",
		"[[:lower:]]",
		"[[:print:]]",
		"[[:punct:]]",
		"[[:space:]]",
		"[[:upper:]]",
		"[[:word:]]",
		"[[:xdigit:]]",
		"[[:^alnum:]]",
		"[[:^alpha:]]",
		"[[:^ascii:]]",
		"[[:^blank:]]",
		"[[:^cntrl:]]",
		"[[:^digit:]]",
		"[[:^graph:]]",
		"[[:^lower:]]",
		"[[:^print:]]",
		"[[:^punct:]]",
		"[[:^space:]]",
		"[[:^upper:]]",
		"[[:^word:]]",
		"[[:^xdigit:]]",
	)
}

func TestPerlCharClasses(t *testing.T) {
	t.Parallel()
	args := &GeneratorArgs{
		Flags: syntax.Perl,
	}

	AssertGeneratesAndMatches(t, args,
		`\d`,
		`\s`,
		`\w`,
		`\D`,
		`\S`,
		`\W`,
	)
}

func TestUnicodeGroupsNotSupported(t *testing.T) {
	t.Parallel()
	args := &GeneratorArgs{
		Flags: syntax.UnicodeGroups,
	}

	_, err := NewGenerator("", args)
	assert.Error(t, err)
}

func TestBeginEndLine(t *testing.T) {
	t.Parallel()
	args := &GeneratorArgs{
		RngSource: rand.NewSource(0),
		Flags:     0,
	}

	AssertGenerates(t, args, `abc`, `^abc$`)
	AssertGenerates(t, args, `abc`, `$abc^`)
	AssertGenerates(t, args, `abc`, `a^b$c`)
}

func AssertGeneratesAndMatches(t *testing.T, args *GeneratorArgs, patterns ...string) {
	for _, pattern := range patterns {
		t.Logf("testing pattern /%s/", pattern)
		AssertGenerates(t, args, pattern, pattern)
	}
}

func AssertGenerates(t *testing.T, args *GeneratorArgs, expectedPattern string, pattern string) {
	AssertGeneratesTimes(t, args, expectedPattern, pattern, SampleSize)
}

func AssertGeneratesTimes(t *testing.T, args *GeneratorArgs, expectedPattern string, pattern string, times int) {
	generator, err := NewGenerator(pattern, args)
	assert.NoError(t, err)

	for i := 0; i < times; i++ {
		result := generator.Generate()
		matched, err := regexp.MatchString(expectedPattern, result)
		if err != nil {
			panic(err)
		}
		require.True(t, matched, "string generated from pattern /%s/ did not match /%s/: `%s`", pattern, expectedPattern, result)
	}
}
