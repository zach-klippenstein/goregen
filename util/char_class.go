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

package util

import (
	"fmt"
)

// CharClass represents a regular expression character class as a list of ranges.
// The runes contained in the class can be accessed by index.
type CharClass struct {
	Ranges    []CharClassRange
	TotalSize int32
}

// ParseCharClass parses a character class as represented by syntax.Parse into a slice of CharClassRange structs.
// e.g. "[a0-9]" -> "aa09" -> a, 0-9
// e.g. "[^a-z]" -> "…" -> 0-(a-1), (z+1)-(max rune)
func ParseCharClass(runes []rune) *CharClass {
	var totalSize int32
	numRanges := len(runes) / 2
	ranges := make([]CharClassRange, numRanges, numRanges)

	for i := 0; i < numRanges; i++ {
		start := runes[i*2]
		end := runes[i*2+1]

		r := CharClassRange{
			Start: start,
			Size:  end - start + 1,
		}

		ranges[i] = r
		totalSize += r.Size
	}

	return &CharClass{ranges, totalSize}
}

// GetRuneAt gets a rune from CharClass as a contiguous array of runes.
func (class *CharClass) GetRuneAt(i int32) rune {
	for _, r := range class.Ranges {
		if i < r.Size {
			return r.Start + rune(i)
		}
		i -= r.Size
	}
	panic("index out of bounds")
}

// CharClassRange represents a single range of characters in a character class.
type CharClassRange struct {
	Start rune
	Size  int32
}

func (r CharClassRange) String() string {
	return fmt.Sprintf("%s-%s", RunesToString(r.Start), RunesToString(r.Start+rune(r.Size)))
}
