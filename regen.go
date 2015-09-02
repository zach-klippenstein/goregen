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

/*
Package regen is a library for generating random strings from regular expressions.
The generated strings will match the expressions they were generated from. Similar
to Ruby's randexp library.

E.g.
	regen.Generate("[a-z0-9]{1,64}")
will return a lowercase alphanumeric string
between 1 and 64 characters long.

Expressions are parsed using the Go standard library's parser: http://golang.org/pkg/regexp/syntax/.

Constraints

"." will generate any character, not necessarily a printable one.

"x{0,}", "x*", and "x+" will generate a random number of x's up to an arbitrary limit.
If you care about the maximum number, specify it explicitly in the expression,
e.g. "x{0,256}".

Flags

Flags can be passed to the parser by setting them in the GeneratorArgs struct.
Newline flags are respected, and newlines won't be generated unless the appropriate flags for
matching them are set.

E.g.
Generate(".|[^a]") will never generate newlines. To generate newlines, create a generator and pass
the flag syntax.MatchNL.

The Perl character class flag is supported, and required if the pattern contains them.

Unicode groups are not supported at this time. Support may be added in the future.

Concurrent Use

A generator can safely be used from multiple goroutines without locking.

A large bottleneck with running generators concurrently is actually the entropy source. Sources returned from
rand.NewSource() are slow to seed, and not safe for concurrent use. Instead, the source passed in GeneratorArgs
is used to seed an XorShift64 source (algorithm from the paper at http://vigna.di.unimi.it/ftp/papers/xorshift.pdf).
This source only uses a single variable internally, and is much faster to seed than the default source. One
source is created per call to NewGenerator. If no source is passed in, the default source is used to seed.

The source is not locked and does not use atomic operations, so there is a chance that multiple goroutines using
the same source may get the same output. While obviously not cryptographically secure, I think the simplicity and performance
benefit outweighs the risk of collisions. If you really care about preventing this, the solution is simple: don't
call a single Generator from multiple goroutines.

Benchmarks

Benchmarks are included for creating and running generators for limited-length,
complex regexes, and simple, highly-repetitive regexes.

	go test -bench .

The complex benchmarks generate fake HTTP messages with the following regex:
	POST (/[-a-zA-Z0-9_.]{3,12}){3,6}
	Content-Length: [0-9]{2,3}
	X-Auth-Token: [a-zA-Z0-9+/]{64}

	([A-Za-z0-9+/]{64}
	){3,15}[A-Za-z0-9+/]{60}([A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)

The repetitive benchmarks use the regex
	a{999}

See regen_benchmarks_test.go for more information.

On my mid-2014 MacBook Pro (2.6GHz Intel Core i5, 8GB 1600MHz DDR3),
the results of running the benchmarks with minimal load are:
	BenchmarkComplexCreation-4                       200	   8322160 ns/op
	BenchmarkComplexGeneration-4                   10000	    153625 ns/op
	BenchmarkLargeRepeatCreateSerial-4  	        3000	    411772 ns/op
	BenchmarkLargeRepeatGenerateSerial-4	        5000	    291416 ns/op
*/
package regen

import (
	"math/rand"
	"regexp/syntax"
)

// GeneratorArgs are arguments passed to NewGenerator that control how generators
// are created.
type GeneratorArgs struct {
	// Used to seed a custom RNG that is a lot faster than the default implementation.
	// See http://vigna.di.unimi.it/ftp/papers/xorshift.pdf.
	RngSource rand.Source

	// Default is 0 (syntax.POSIX).
	Flags syntax.Flags

	// Used by generators.
	rng *rand.Rand
}

// Generator generates random strings.
type Generator interface {
	Generate() string
}

/*
Generate a random string that matches the regular expression pattern.
If args is nil, default values are used.

This function does not seed the default RNG, so you must call rand.Seed() if you want
non-deterministic strings.
*/
func Generate(pattern string) (string, error) {
	generator, err := NewGenerator(pattern, nil)
	if err != nil {
		return "", err
	}
	return generator.Generate(), nil
}

// NewGenerator creates a generator that returns random strings that match the regular expression in pattern.
// If args is nil, default values are used.
func NewGenerator(pattern string, args *GeneratorArgs) (generator Generator, err error) {
	if nil == args {
		args = &GeneratorArgs{}
	}

	var seed int64
	if nil == args.RngSource {
		seed = rand.Int63()
	} else {
		seed = args.RngSource.Int63()
	}
	rngSource := xorShift64Source(seed)
	args.rng = rand.New(&rngSource)

	// unicode groups only allowed with Perl
	if (args.Flags&syntax.UnicodeGroups) == syntax.UnicodeGroups && (args.Flags&syntax.Perl) != syntax.Perl {
		return nil, generatorError(nil, "UnicodeGroups not supported")
	}

	var regexp *syntax.Regexp
	regexp, err = syntax.Parse(pattern, args.Flags)
	if err != nil {
		return
	}

	var gen *internalGenerator
	gen, err = newGenerator(regexp, args)
	if err != nil {
		return
	}

	return gen, nil
}
