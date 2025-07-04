package irc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/a-random-lemurian/go-irc"
)

func TestMaskToRegex(t *testing.T) {
	t.Parallel()

	var testCases = []struct { //nolint:gofumpt
		Input  string
		Expect string
	}{
		{ // Empty should be fine
			Input:  "",
			Expect: "^$",
		},
		{ // EVERYONE!
			Input:  "*!*@*",
			Expect: "^.*!.*@.*$",
		},
		{
			Input:  "",
			Expect: "^$",
		},
		{
			Input:  "",
			Expect: "^$",
		},
		{ // Escape the slash
			Input:  "a\\\\b",
			Expect: "^a\\\\b$",
		},
		{ // Escape a *
			Input:  "a\\*b",
			Expect: "^a\\*b$",
		},
		{ // Escape a ?
			Input:  "a\\?b",
			Expect: "^a\\?b$",
		},
		{ // Single slash in the middle of a string should be a slash
			Input:  "a\\b",
			Expect: "^a\\\\b$",
		},
		{ // Single slash should just match a single slash
			Input:  "\\",
			Expect: "^\\\\$",
		},
		{
			Input:  "\\a?",
			Expect: "^\\\\a.$",
		},
	}

	for _, testCase := range testCases {
		ret, err := irc.MaskToRegex(testCase.Input)
		assert.NoError(t, err)
		assert.Equal(t, testCase.Expect, ret.String())
	}
}
