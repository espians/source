// Public Domain (-) 2018-present, The Espian Source Authors.
// See the Espian Source UNLICENSE file for details.

package textwrap

import (
	"testing"
)

func TestDedent(t *testing.T) {
	input := `

		first line

			second

	third

		`
	expected := `

	first line

		second

third

	`
	output := Dedent(input)
	if output != expected {
		t.Errorf("Dedent did not match expected output.\nExpected: %q\n     Got: %q\n", expected, output)
	}
}
