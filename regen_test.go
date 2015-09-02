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

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
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

func TestRegen(t *testing.T) {
	t.Parallel()

	Convey("Regen", t, func() {

		Convey("NewGenerator", func() {

			Convey("Handles nil GeneratorArgs", func() {
				generator, err := NewGenerator("", nil)
				So(generator, ShouldNotBeNil)
				So(err, ShouldBeNil)
			})

			Convey("Handles empty GeneratorArgs", func() {
				generator, err := NewGenerator("", &GeneratorArgs{})
				So(generator, ShouldNotBeNil)
				So(err, ShouldBeNil)
			})
		})

		Convey("Empty", func() {
			args := &GeneratorArgs{
				RngSource: rand.NewSource(0),
			}
			ConveyGeneratesStringMatching(args, "", "^$")
		})

		Convey("Literals", func() {
			ConveyGeneratesStringMatchingItself(nil,
				"a",
				"abc",
			)
		})

		Convey("DotNotNl", func() {
			ConveyGeneratesStringMatchingItself(nil, ".")

			Convey("No newlines are generated", func() {
				generator, _ := NewGenerator(".", nil)

				// Not a very strong assertion, but not sure how to do better. Exploring the entire
				// generation space (2^32) takes far too long for a unit test.
				for i := 0; i < SampleSize; i++ {
					So(generator.Generate(), ShouldNotContainSubstring, "\n")
				}
			})
		})

		Convey("String start/end", func() {
			args := &GeneratorArgs{
				RngSource: rand.NewSource(0),
				Flags:     0,
			}

			ConveyGeneratesStringMatching(args, `^abc$`, `^abc$`)
			ConveyGeneratesStringMatching(args, `$abc^`, `^abc$`)
			ConveyGeneratesStringMatching(args, `a^b$c`, `^abc$`)
		})

		Convey("QuestionMark", func() {
			ConveyGeneratesStringMatchingItself(nil,
				"a?",
				"(abc)?",
				"[ab]?",
				".?")
		})

		Convey("Plus", func() {
			ConveyGeneratesStringMatchingItself(nil, "a+")
		})

		Convey("Star", func() {
			ConveyGeneratesStringMatchingItself(nil, "a*")
		})

		Convey("CharClassNotNl", func() {
			ConveyGeneratesStringMatchingItself(nil,
				"[a]",
				"[abc]",
				"[a-d]",
				"[ac]",
				"[0-9]",
				"[a-z0-9]",
			)

			Convey("No newlines are generated", func() {
				// Try to narrow down the generation space. Still not a very strong assertion.
				generator, _ := NewGenerator("[^a-zA-Z0-9]", nil)
				for i := 0; i < SampleSize; i++ {
					assert.NotEqual(t, "\n", generator.Generate())
				}
			})
		})

		Convey("NegativeCharClass", func() {
			ConveyGeneratesStringMatchingItself(nil, "[^a-zA-Z0-9]")
		})

		Convey("Alternate", func() {
			ConveyGeneratesStringMatchingItself(nil,
				"a|b",
				"abc|def|ghi",
				"[ab]|[cd]",
				"foo|bar|baz", // rewrites to foo|ba[rz]
			)
		})

		Convey("Capture", func() {
			ConveyGeneratesStringMatching(nil, "(abc)", "^abc$")
			ConveyGeneratesStringMatching(nil, "()", "^$")
		})

		Convey("Concat", func() {
			ConveyGeneratesStringMatchingItself(nil, "[ab][cd]")
		})

		Convey("Repeat", func() {

			Convey("Unbounded", func() {
				ConveyGeneratesStringMatchingItself(nil, `a{1,}`)
			})

			Convey("HitsMin", func() {
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

				Println("counts:")
				for i, count := range counts {
					Printf("%d: %d\n", i, count)
				}

				So(counts[0], ShouldBeGreaterThan, 0)
			})

			Convey("HitsMax", func() {
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

				Println("counts:")
				for i, count := range counts {
					Printf("%d: %d\n", i, count)
				}

				So(counts[3], ShouldBeGreaterThan, 0)
			})
		})

		Convey("CharClasses", func() {

			Convey("Ascii", func() {
				ConveyGeneratesStringMatchingItself(nil,
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
			})

			Convey("Perl", func() {
				args := &GeneratorArgs{
					Flags: syntax.Perl,
				}

				ConveyGeneratesStringMatchingItself(args,
					`\d`,
					`\s`,
					`\w`,
					`\D`,
					`\S`,
					`\W`,
				)
			})

			Convey("Unicode groups not supported", func() {
				args := &GeneratorArgs{
					Flags: syntax.UnicodeGroups,
				}

				_, err := NewGenerator("", args)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func ConveyGeneratesStringMatchingItself(args *GeneratorArgs, patterns ...string) {
	for _, pattern := range patterns {
		Convey(fmt.Sprintf("String generated from /%s/ matches itself", pattern), func() {
			So(pattern, ShouldGenerateStringMatching, pattern, args)
		})
	}
}

func ConveyGeneratesStringMatching(args *GeneratorArgs, pattern string, expectedPattern string) {
	Convey(fmt.Sprintf("String generated from /%s/ matches /%s/", pattern, expectedPattern), func() {
		So(pattern, ShouldGenerateStringMatching, expectedPattern, args)
	})
}

func ShouldGenerateStringMatching(actual interface{}, expected ...interface{}) string {
	return ShouldGenerateStringMatchingTimes(actual, expected[0], expected[1], SampleSize)
}

func ShouldGenerateStringMatchingTimes(actual interface{}, expected ...interface{}) string {
	pattern := actual.(string)
	expectedPattern := expected[0].(string)
	args := expected[1].(*GeneratorArgs)
	times := expected[2].(int)

	generator, err := NewGenerator(pattern, args)
	if err != nil {
		panic(err)
	}

	for i := 0; i < times; i++ {
		result := generator.Generate()
		matched, err := regexp.MatchString(expectedPattern, result)
		if err != nil {
			panic(err)
		}
		if !matched {
			return fmt.Sprintf("string “%s” generated from /%s/ did not match /%s/.",
				result, pattern, expectedPattern)
		}
	}

	return ""
}
