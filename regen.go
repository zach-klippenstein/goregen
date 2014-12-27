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
A library for generating random strings from regular expressions.
The generated strings will match the expressions they were generated from.

E.g.
	regen.Generate("[a-z0-9]{1,64}")
will return a lowercase alphanumeric string
between 1 and 64 characters long.
*/
package regen

import (
	"bytes"
	"fmt"
	"github.com/zach-klippenstein/goregen/util"
	"math/rand"
	"regexp/syntax"
)

/*
MaxUpperBound is the number of instances to generate for unbounded repeat expressions.

E.g. .* will generate no more than MaxUpperBound characters.
*/
const MaxUpperBound = 4096

var defaultGeneratorArgs = GeneratorArgs{
	Rng: util.NewRand(rand.Int63()),
}

type GeneratorArgs struct {
	Rng *rand.Rand
}

// Generator generates random strings.
type Generator interface {
	Generate() string
}

type aGenerator func() string

func (gen aGenerator) Generate() string {
	return gen()
}

// generatorFactory is a function that creates a random string generator from a regular expression AST.
type generatorFactory func(r *syntax.Regexp, args *GeneratorArgs) (Generator, error)

// Must be initialized in init() to avoid "initialization loop" compile error.
var generatorFactories map[syntax.Op]generatorFactory

func init() {
	generatorFactories = map[syntax.Op]generatorFactory{
		syntax.OpEmptyMatch:     opEmptyMatch,
		syntax.OpLiteral:        opLiteral,
		syntax.OpAnyCharNotNL:   opAnyCharNotNl,
		syntax.OpAnyChar:        opAnyChar,
		syntax.OpQuest:          opQuest,
		syntax.OpStar:           opStar,
		syntax.OpPlus:           opPlus,
		syntax.OpRepeat:         opRepeat,
		syntax.OpCharClass:      opCharClass,
		syntax.OpConcat:         opConcat,
		syntax.OpAlternate:      opAlternate,
		syntax.OpCapture:        opCapture,
		syntax.OpBeginLine:      noop,
		syntax.OpEndLine:        noop,
		syntax.OpBeginText:      noop,
		syntax.OpEndText:        noop,
		syntax.OpWordBoundary:   noop,
		syntax.OpNoWordBoundary: noop,
	}
}

// Generate a random string that matches the regular expression r.
// If args is nil, default values are used.
func Generate(r string) (string, error) {
	generator, err := NewGenerator(r, nil)
	if err != nil {
		return "", err
	}
	return generator.Generate(), nil
}

// NewGenerator creates a generator that returns random strings that match the regular expression in r.
// If args is nil, default values are used.
func NewGenerator(r string, args *GeneratorArgs) (generator Generator, err error) {
	if nil == args {
		defaultCopy := defaultGeneratorArgs
		args = &defaultCopy
	}

	var regexp *syntax.Regexp
	regexp, err = syntax.Parse(r, 0)
	if err != nil {
		return
	}

	return newGenerator(regexp, args)
}

// Create a new generator for each expression in rs.
func newGenerators(rs []*syntax.Regexp, args *GeneratorArgs) ([]Generator, error) {
	generators := make([]Generator, len(rs), len(rs))
	var err error

	// create a generator for each alternate pattern
	for i, subR := range rs {
		generators[i], err = newGenerator(subR, args)
		if err != nil {
			return nil, err
		}
	}

	return generators, nil
}

// Create a new generator for r.
func newGenerator(r *syntax.Regexp, args *GeneratorArgs) (generator Generator, err error) {
	simplified := r.Simplify()

	factory, ok := generatorFactories[simplified.Op]
	if ok {
		return factory(simplified, args)
	}

	return nil, fmt.Errorf("invalid generator pattern: /%s/ as /%s/\n%s",
		r, simplified, util.InspectToStr(simplified))
}

// Generator that does nothing.
func noop(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	return aGenerator(func() string {
		return ""
	}), nil
}

func opEmptyMatch(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpEmptyMatch)
	return aGenerator(func() string {
		return ""
	}), nil
}

func opLiteral(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpLiteral)
	return aGenerator(func() string {
		return util.RunesToString(r.Rune...)
	}), nil
}

func opAnyChar(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpAnyChar)
	return aGenerator(func() string {
		return util.RunesToString(rune(args.Rng.Int31()))
	}), nil
}

func opAnyCharNotNl(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpAnyCharNotNL)
	return aGenerator(func() string {
		return util.RunesToString(rune(args.Rng.Int31()))
	}), nil
}

func opQuest(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpQuest)
	return createRepeatingGenerator(r, args, 0, 1)
}

func opStar(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpStar)
	return createRepeatingGenerator(r, args, 0, -1)
}

func opPlus(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpPlus)
	return createRepeatingGenerator(r, args, 1, -1)
}

func opRepeat(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpRepeat)
	return createRepeatingGenerator(r, args, r.Min, r.Max)
}

func opCharClass(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpCharClass)
	// case classes are encoded as pairs of runes representing ranges.
	// e.g. [0-9] = 09, [a0] = aa00 (2 1-len ranges)

	class := util.ParseCharClass(r.Rune)

	return aGenerator(func() string {
		i := util.Abs(args.Rng.Int31n(class.TotalSize))
		r := class.GetRuneAt(i)
		return util.RunesToString(r)
	}), nil
}

func opConcat(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpConcat)

	generators, err := newGenerators(r.Sub, args)
	if err != nil {
		return nil, generatorError(err, "error creating generators for concat pattern /%s/", r.String())
	}

	return aGenerator(func() string {
		var buffer bytes.Buffer
		for _, generator := range generators {
			buffer.WriteString(generator.Generate())
		}
		return buffer.String()
	}), nil
}

func opAlternate(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpAlternate)

	generators, err := newGenerators(r.Sub, args)
	if err != nil {
		return nil, generatorError(err, "error creating generators for alternate pattern /%s/", r.String())
	}

	var numGens int = len(generators)

	return aGenerator(func() string {
		i := args.Rng.Intn(numGens)
		generator := generators[i]
		return generator.Generate()
	}), nil
}

func opCapture(r *syntax.Regexp, args *GeneratorArgs) (Generator, error) {
	enforceOp(r, syntax.OpCapture)

	if err := enforceSingleSub(r); err != nil {
		return nil, err
	}

	return newGenerator(r.Sub[0], args)
}

// Panic if r.Op != op.
func enforceOp(r *syntax.Regexp, op syntax.Op) {
	if r.Op != op {
		panic(fmt.Sprintf("invalid Op: expected %s, was %s", util.OpToString(op), util.OpToString(r.Op)))
	}
}

// Return an error if r has 0 or more than 1 sub-expression.
func enforceSingleSub(r *syntax.Regexp) error {
	if len(r.Sub) != 1 {
		return generatorError(nil,
			"%s expected 1 sub-expression, but got %d: %s", util.OpToString(r.Op), len(r.Sub), r)
	}
	return nil
}

// Returns a generator that will run the generator for r's sub-expression [min, max] times.
func createRepeatingGenerator(r *syntax.Regexp, args *GeneratorArgs, min int, max int) (Generator, error) {
	if err := enforceSingleSub(r); err != nil {
		return nil, err
	}

	generator, err := newGenerator(r.Sub[0], args)
	if err != nil {
		return nil, generatorError(err, "Failed to create generator for subexpression: /%s/", r)
	}

	if max < 0 {
		max = MaxUpperBound
	}

	return aGenerator(func() string {
		var buffer bytes.Buffer
		n := min + args.Rng.Intn(max-min+1)

		for ; n > 0; n-- {
			buffer.WriteString(generator.Generate())
		}

		return buffer.String()
	}), nil
}
