package irc_test

import (
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/a-random-lemurian/go-irc"
)

var newM = irc.Message{
	Tags: irc.Tags{"is-cat": "1", "is-dog": "0"},
	Prefix: &irc.Prefix{
		Name: "lemuria",
		User: "lemuria",
		Host: "lemuria.ph",
	},
	Command: "PRIVMSG",
	Params:  []string{"#lemuria", "meow"},
}

func BenchmarkParseMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		irc.MustParseMessage("@tag1=something :nick!user@host PRIVMSG #channel :some message")
	}
}

func BenchmarkStringMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = newM.String()
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}

func BenchmarkStringMessageAlphabetized(b *testing.B) {
	b.StopTimer() // we need to warm up the messages for sorting first
	irc.AlphabetizeTagMaps = true

    rand.Seed(time.Now().UnixNano())

	manyKeys := newM.Copy()

	for i := 0; i < 100; i++ {
		manyKeys.Tags[RandStringRunes(rand.Intn(18)+3)] = RandStringRunes(rand.Intn(18)+3)
	}

	b.StartTimer()
	defer b.StopTimer()
	for i := 0; i < b.N; i++ {
		_ = newM.String()
	}
}

func TestParseMessage(t *testing.T) {
	t.Parallel()

	var messageTests = []struct { //nolint:gofumpt
		Input string
		Err   error
	}{
		{
			Input: "",
			Err:   irc.ErrZeroLengthMessage,
		},
		{
			Input: "@asdf",
			Err:   irc.ErrMissingDataAfterTags,
		},
		{
			Input: ":asdf",
			Err:   irc.ErrMissingDataAfterPrefix,
		},
		{
			Input: " :",
			Err:   irc.ErrMissingCommand,
		},
		{
			Input: "PING :asdf",
		},
	}

	for i, test := range messageTests {
		m, err := irc.ParseMessage(test.Input)
		assert.Equal(t, test.Err, err, "%d. Error didn't match expected", i)

		if test.Err != nil {
			assert.Nil(t, m, "%d. Didn't get nil message", i)
		} else {
			assert.NotNil(t, m, "%d. Got nil message", i)
		}
	}
}

func TestMustParseMessage(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		irc.MustParseMessage("")
	}, "Didn't get expected panic")

	assert.NotPanics(t, func() {
		irc.MustParseMessage("PING :asdf")
	}, "Got unexpected panic")
}

func TestMessageParam(t *testing.T) {
	t.Parallel()

	m := irc.MustParseMessage("PING :test")
	assert.Equal(t, m.Param(0), "test")
	assert.Equal(t, m.Param(-1), "")
	assert.Equal(t, m.Param(2), "")
}

func TestMessageTrailing(t *testing.T) {
	t.Parallel()

	m := irc.MustParseMessage("PING :helloworld")
	assert.Equal(t, "helloworld", m.Trailing())

	m = irc.MustParseMessage("PING")
	assert.Equal(t, "", m.Trailing())
}

func TestMessageCopy(t *testing.T) {
	t.Parallel()

	m := irc.MustParseMessage("@tag=val :user@host PING :helloworld")

	// Ensure copied messages are equal
	c := m.Copy()
	assert.EqualValues(t, m, c, "Copied values are not equal")

	// Ensure messages with modified tags don't match
	c = m.Copy()
	for k := range c.Tags {
		c.Tags[k] += "junk"
	}
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with modified tags should not match")

	// Ensure messages with modified prefix don't match
	c = m.Copy()
	c.Prefix.Name += "junk"
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with modified identity should not match")

	// Ensure messages with modified params don't match
	c = m.Copy()
	c.Params = append(c.Params, "junk")
	assert.False(t, assert.ObjectsAreEqualValues(m, c), "Copied with additional params should not match")

	// The message itself doesn't matter, we just need to make sure we
	// don't error if the user does something crazy and makes Params
	// nil.
	m = irc.MustParseMessage("PING :hello world")
	m.Prefix = nil
	c = m.Copy()
	assert.EqualValues(t, m, c, "nil prefix copy failed")

	// Ensure an empty Params is copied as nil
	m = irc.MustParseMessage("PING")
	m.Params = []string{}
	c = m.Copy()
	assert.Nil(t, c.Params, "Expected nil for empty params")
}

// Everything beyond here comes from the testcases repo

type MsgSplitTests struct {
	Tests []struct {
		Desc  string
		Input string
		Atoms struct {
			Source *string
			Verb   string
			Params []string
			Tags   map[string]interface{}
		}
	}
}

func TestMsgSplit(t *testing.T) {
	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/msg-split.yaml")
	require.NoError(t, err)

	var splitTests MsgSplitTests
	err = yaml.Unmarshal(data, &splitTests)
	require.NoError(t, err)

	for _, test := range splitTests.Tests {
		msg, err := irc.ParseMessage(test.Input)
		assert.NoError(t, err, "%s: Failed to parse: %s (%s)", test.Desc, test.Input, err)

		assert.Equal(t,
			strings.ToUpper(test.Atoms.Verb), msg.Command,
			"%s: Wrong command for input: %s", test.Desc, test.Input,
		)
		assert.Equal(t,
			test.Atoms.Params, msg.Params,
			"%s: Wrong params for input: %s", test.Desc, test.Input,
		)

		if test.Atoms.Source != nil {
			assert.Equal(t, *test.Atoms.Source, msg.Prefix.String())
		}

		assert.Equal(t,
			len(test.Atoms.Tags), len(msg.Tags),
			"%s: Wrong number of tags",
			test.Desc,
		)

		for k, v := range test.Atoms.Tags {
			tag, ok := msg.Tags[k]
			assert.True(t, ok, "Missing tag")
			if v == nil {
				assert.EqualValues(t, "", tag, "%s: Tag %q differs: %s != \"\"", test.Desc, k, tag)
			} else {
				assert.EqualValues(t, v, tag, "%s: Tag %q differs: %s != %s", test.Desc, k, v, tag)
			}
		}
	}
}

type MsgJoinTests struct {
	Tests []struct {
		Desc  string
		Atoms struct {
			Source string
			Verb   string
			Params []string
			Tags   map[string]interface{}
		}
		Matches []string
	}
}

func TestMsgJoin(t *testing.T) {
	var ok bool

	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/msg-join.yaml")
	require.NoError(t, err)

	var splitTests MsgJoinTests
	err = yaml.Unmarshal(data, &splitTests)
	require.NoError(t, err)

	for _, test := range splitTests.Tests {
		msg := &irc.Message{
			Prefix:  irc.ParsePrefix(test.Atoms.Source),
			Command: test.Atoms.Verb,
			Params:  test.Atoms.Params,
			Tags:    make(map[string]string),
		}

		for k, v := range test.Atoms.Tags {
			if v == nil {
				msg.Tags[k] = ""
			} else {
				msg.Tags[k], ok = v.(string)
				assert.True(t, ok)
			}
		}

		assert.Contains(t, test.Matches, msg.String())
	}
}

type UserhostSplitTests struct {
	Tests []struct {
		Desc   string
		Source string
		Atoms  struct {
			Nick string
			User string
			Host string
		}
	}
}

func TestUserhostSplit(t *testing.T) {
	t.Parallel()

	data, err := ioutil.ReadFile("./_testcases/tests/userhost-split.yaml")
	require.NoError(t, err)

	var userhostTests UserhostSplitTests
	err = yaml.Unmarshal(data, &userhostTests)
	require.NoError(t, err)

	for _, test := range userhostTests.Tests {
		prefix := irc.ParsePrefix(test.Source)

		assert.Equal(t,
			test.Atoms.Nick, prefix.Name,
			"%s: Name did not match for input: %q", test.Desc, test.Source,
		)
		assert.Equal(t,
			test.Atoms.User, prefix.User,
			"%s: User did not match for input: %q", test.Desc, test.Source,
		)
		assert.Equal(t,
			test.Atoms.Host, prefix.Host,
			"%s: Host did not match for input: %q", test.Desc, test.Source,
		)
	}
}

func TestOriginallyParsedMessage(t *testing.T) {
	t.Parallel()

	original := "@is-cat=1;is-dog=0 :lemuria!lemuria@lemuria.ph PRIVMSG #lemuria meow"
	m := irc.MustParseMessage(original)
	assert.EqualValues(t, original, m.String())
}

func TestNotOriginallyParsedMessage(t *testing.T) {
	t.Parallel()

	assert.NotEqualValues(t, newM.String(), "")
}

func TestOriginallyParsedThenEditedMessage(t *testing.T) {
	t.Parallel()

	original := "@is-cat=1;is-dog=0 :lemuria!lemuria@lemuria.ph PRIVMSG #lemuria meow"
	m := irc.MustParseMessage(original)

	m.Tags["is-cat-lover"] = "1"

	assert.Contains(t, m.String(), "is-cat-lover=1")
}
