package filter

import (
	"regexp"
	"testing"

	"gotest.tools/assert"
)

func TestMatchName(t *testing.T) {
	pats := []string{"^main$", "mai"}
	inputs := []string{"main", "maint"}
	wants := []bool{true, false, true, true}
	var index int
	for _, pat := range pats {
		reg, err := regexp.Compile(pat)
		assert.NilError(t, err, "compile failed", pat)
		for _, input := range inputs {
			assert.Equal(t, reg.MatchString(input), wants[index])
			index += 1
		}
	}
}
