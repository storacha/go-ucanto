package schema_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/storacha-network/go-ucanto/core/schema"
	"github.com/stretchr/testify/require"
)

func TestURIDefaultReader(t *testing.T) {
	testCases := []struct {
		source   string
		output   string
		errMatch *regexp.Regexp
	}{
		{
			source:   "",
			errMatch: regexp.MustCompile("Invalid URI"),
		},
		{
			source: "did:key:zAlice",
			output: "did:key:zAlice",
		},
		{
			source: "mailto:alice@mail.net",
			output: "mailto:alice@mail.net",
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("schema.URI.read(%s)", testCase.source), func(t *testing.T) {
			output, err := schema.URI().Read(testCase.source)
			if testCase.errMatch == nil {
				require.NoError(t, err)
				require.Equal(t, testCase.output, output.String())
			} else {
				require.Regexp(t, testCase.errMatch, err.Error())
			}
		})
	}
}

func TestURIReaderWithProtocol(t *testing.T) {
	testCases := []struct {
		source   any
		protocol string
		output   string
		errMatch *regexp.Regexp
	}{
		{nil, "did:", "", regexp.MustCompile("Expected URI but got <nil>")},
		{"", "did:", "", regexp.MustCompile("Invalid URI")},
		{"did:key:zAlice", "did:", "did:key:zAlice", nil},
		{"did:key:zAlice", "mailto:", "", regexp.MustCompile("Expected mailto: URI instead got did:key:zAlice")},
		{"mailto:alice@mail.net", "mailto:", "mailto:alice@mail.net", nil},
		{"mailto:alice@mail.net", "did:", "", regexp.MustCompile("Expected did: URI instead got mailto:alice@mail.net")},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("schema.URI(WithProtocol(%s)).read(%s)", testCase.protocol, testCase.source), func(t *testing.T) {
			output, err := schema.URI(schema.WithProtocol((testCase.protocol))).Read(testCase.source)
			if testCase.errMatch == nil {
				require.NoError(t, err)
				require.Equal(t, testCase.output, output.String())
			} else {
				require.Regexp(t, testCase.errMatch, err.Error())
			}
		})
	}
	//	for (const [input, protocol, expect] of dataset) {
	//	  test(`URI.match(${JSON.stringify({
	//	    protocol,
	//	  })}).read(${JSON.stringify(input)})}}`, () => {
	//	    matchResult(URI.match({ protocol }).read(input), expect)
	//	  })
	//	}
}
