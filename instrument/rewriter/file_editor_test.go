package rewriter

import (
	"testing"

	"gotest.tools/assert"
)

func TestRewriter(t *testing.T) {
	cases := []struct {
		name            string
		content         string
		edits           []Edit
		expectedContent string
		hasErr          bool
	}{
		{
			name:    "case1-2add-begin",
			content: "abcdefg",
			edits: []Edit{
				{
					OpType:   EditTypeAdd,
					BeginPos: 0,
					EndPos:   0,
					Content:  []byte("aa"),
				},
				{
					OpType:   EditTypeAdd,
					BeginPos: 0,
					EndPos:   0,
					Content:  []byte("bb"),
				},
			},
			expectedContent: "aabbabcdefg",
			hasErr:          false,
		},
		{
			name:    "case2-one-pos-add-del",
			content: "abcdefg",
			edits: []Edit{
				{
					OpType:   EditTypeAdd,
					BeginPos: 2,
					EndPos:   2,
					Content:  []byte("ee"),
				},
				{
					OpType:   EditTypeDel,
					BeginPos: 2,
					EndPos:   3,
				},
			},
			expectedContent: "abeeefg",
			hasErr:          false,
		},
		{
			name:    "case3-one-pos-replace",
			content: "abcdefg",
			edits: []Edit{
				{
					OpType:   EditTypeReplace,
					BeginPos: 3,
					EndPos:   5,
					Content:  []byte("xx"),
				},
			},
			expectedContent: "abcxxg",
			hasErr:          false,
		},
		{
			name:    "case3-one-error",
			content: "abcdefg",
			edits: []Edit{
				{
					OpType:   EditTypeReplace,
					BeginPos: 3,
					EndPos:   5,
					Content:  []byte("xx"),
				},
				{
					OpType:   EditType(-1),
					BeginPos: 3,
					EndPos:   5,
				},
			},
			expectedContent: "",
			hasErr:          true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := FileRewriter{
				Content: []byte(c.content),
				Edits:   c.edits,
			}
			content, err := f.Rewrite()
			assert.Equal(t, c.hasErr, err != nil)
			assert.Equal(t, string(content), c.expectedContent)
		})
	}
}
