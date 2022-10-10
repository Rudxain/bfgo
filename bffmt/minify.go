package bffmt

import (
	"os"
	"regexp"

	"github.com/baris-inandi/bfgo/lang/readcode"
	"github.com/baris-inandi/bfgo/utils"
)

func MinifyFile(files ...string) {
	// canonical "-" unconditional cell-reseter
	const ODD_RESET = "[-]"
	// canonical "-" conditional (break-if-even) cell-reseter
	const EVEN_RESET = "[--]"

	// matches 256 consecutive + or - (exclusive, so mixes don't match)
	var x256PlusMinus = regexp.MustCompile(`\+{256}|-{256}`)

	// matches pairs of BF opcodes that cancel each other (any order)
	var mutualCancel = regexp.MustCompile(`\+-|-\+|><|<>`)

	// matches any odd (unconditional) cell-reseters, except ODD_RESET
	var isOddReset = regexp.MustCompile(`\[(?:(?:\+\+){1,128}\+|(?:--){1,128}-)\]`)

	// matches any even (conditional break) cell-reseters, except EVEN_RESET
	var isEvenReset = regexp.MustCompile(`\[(?:(?:\+\+){2,128}|(?:--){2,128})\]`)

	// matches ODD_RESET, preceded by 1 or more "+" or "-" (mixed)
	var isPrefixedReset = regexp.MustCompile(`[+-]+\[-\]`)

	// finds index of 1st byte that isn't in the charset "[],.", or -1 if not found.
	//
	// `start` ignores all runes before that index.
	// If `start` is negative, it becomes relative to the end.
	var indexNoIOBrace = func(s string, start int) int {
		size := len(s)
		start = utils.RelativeIndex(start, size)

		// "I couldn't find a way to write it in functional-paradigm" @Rudxain
		for start < size {
			c := s[start]
			if c != '[' && c != ']' && c != ',' && c != '.' {
				return start
			}
			start += 1
		}
		return -1
	}

	// # Memory Simulator
	//
	// Interprets memory ops ("+-<>") in [basic blocks]
	// and replaces each block by a minimal sequence.
	//
	// It assumes `IOBrace` is a black-box with potential-side effects,
	// so each sequence is only locally-optimal.
	//
	// current implementation is identity fn (TO-DO)
	//
	// [basic blocks]: https://en.wikipedia.org/wiki/Basic_block
	var memSim = func(s string) string {
		// simulated BF memory/tape
		var mem = map[int]uint8{}
		// relative memory pointer
		var ptr int = 0

		/* # pseudo-code
		0. split s by IOBrace (consecutives are treated as 1).
		1. sim each substr in the resulting array,
		such that each sub has its own isolated mem.
		2. replace each substr by its "canonical form"
		derived from mem.
		3. re-insert delimiters at corresponding positions.
		*/
		// we need an outer loop to cleanup mem.
		// and inner loop should break whenever it finds IOBrace
		for i := indexNoIOBrace(s, 0); i < len(s) && i > -1; i = indexNoIOBrace(s, i+1) {
			switch s[i] {
			case '+':
				{
					mem[ptr] += 1
					continue
				}
			case '-':
				{
					mem[ptr] -= 1
					continue
				}
			case '>':
				{
					ptr += 1
					continue
				}
			case '<':
				{
					ptr -= 1
					continue
				}
			}
		}
		return s
	}

	// # Advanced BF minifier
	//
	// Explained in [#2]. It assumes s only has valid opcodes.
	//
	// [#2]: https://github.com/baris-inandi/brainfuck-go/issues/2
	var minify = func(s string) string {
		s = x256PlusMinus.ReplaceAllLiteralString(s, "")

		var size int
		for do := true; do; do = (size != len(s)) {
			size = len(s)
			s = mutualCancel.ReplaceAllLiteralString(s, "")
		}
		// order matters, (from this point onwards)

		// TO-DO: mem-sim must supersede both regexps above
		s = memSim(s)
		// these 3 are "amplified" by mem-sim
		s = isEvenReset.ReplaceAllLiteralString(s, EVEN_RESET)
		s = isOddReset.ReplaceAllLiteralString(s, ODD_RESET)
		s = isPrefixedReset.ReplaceAllLiteralString(s, ODD_RESET)
		return s
	}

	for _, f := range files {
		minified := minify(readcode.ReadBFCode(f))
		err := os.WriteFile(f, []byte(minified), 0o644)
		if err != nil {
			panic(err)
		}
	}
}
