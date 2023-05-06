package chatgpt

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func strToPtr(s string) *string {
	return &s
}
func TestIsErrorTooManyRequest(t *testing.T) {
	type testCase struct {
		name           string
		err            error
		expectedOutput bool
	}

	testCases := []testCase{
		{
			name:           "error is nil",
			err:            nil,
			expectedOutput: false,
		},
		{
			name:           "error is not APIError",
			err:            errors.New("some error"),
			expectedOutput: false,
		},
		{
			name:           "error code is nil",
			err:            &openai.APIError{},
			expectedOutput: false,
		},
		{
			name: "error code is not number",
			err: &openai.APIError{
				Code: strToPtr("abc"),
			},
			expectedOutput: false,
		},
		{
			name: "error code is http.StatusTooManyRequests",
			err: &openai.APIError{
				Code: strToPtr(strconv.Itoa(http.StatusTooManyRequests)),
			},
			expectedOutput: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := IsErrorTooManyRequest(tc.err)
			if output != tc.expectedOutput {
				t.Errorf("Expected IsErrorTooManyRequest(%v) to be %v, but got %v", tc.err, tc.expectedOutput, output)
			}
		})
	}

}
