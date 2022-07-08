package log

import (
	"bufio"
	"bytes"
	"context"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

type nopCloser struct {
	io.Writer
	closed chan struct{}
}

func (nc *nopCloser) Close() error {
	nc.closed <- struct{}{}
	return nil
}

func TestFileHookFromConfigLine(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		line       string
		err        bool
		errMessage string
		res        fileHook
	}{
		{
			line: "file",
			err:  true,
			res: fileHook{
				levels: logrus.AllLevels,
			},
		},
		{
			line: "file=/tiger.log,level=info",
			err:  false,
			res: fileHook{
				path:   "/tiger.log",
				levels: logrus.AllLevels[:5],
			},
		},
		{
			line: "file=/a/c/",
			err:  true,
		},
		{
			line:       "file=,level=info",
			err:        true,
			errMessage: "filepath must not be empty",
		},
		{
			line: "file=/tmp/tiger.log,level=tea",
			err:  true,
		},
		{
			line: "file=/tmp/tiger.log,unknown",
			err:  true,
		},
		{
			line: "file=/tmp/tiger.log,level=",
			err:  true,
		},
		{
			line: "file=/tmp/tiger.log,level=,",
			err:  true,
		},
		{
			line:       "file=/tmp/tiger.log,unknown=something",
			err:        true,
			errMessage: "unknown logfile config key unknown",
		},
		{
			line:       "unknown=something",
			err:        true,
			errMessage: "logfile configuration should be in the form `file=path-to-local-file` but is `unknown=something`",
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.line, func(t *testing.T) {
			t.Parallel()

			getCwd := func() (string, error) {
				return "/", nil
			}

			res, err := FileHookFromConfigLine(
				context.Background(), afero.NewMemMapFs(), getCwd, logrus.New(), test.line, make(chan struct{}),
			)

			if test.err {
				require.Error(t, err)

				if test.errMessage != "" {
					require.Equal(t, test.errMessage, err.Error())
				}

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, res.(*fileHook).w)
		})
	}
}

func TestFileHookFire(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	nc := &nopCloser{
		Writer: &buffer,
		closed: make(chan struct{}),
	}

	hook := &fileHook{
		loglines: make(chan []byte),
		w:        nc,
		bw:       bufio.NewWriter(nc),
		levels:   logrus.AllLevels,
		done:     make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())

	hook.loglines = hook.loop(ctx)

	logger := logrus.New()
	logger.AddHook(hook)
	logger.SetOutput(io.Discard)

	logger.Info("example log line")

	time.Sleep(10 * time.Millisecond)

	cancel()
	<-nc.closed

	assert.Contains(t, buffer.String(), "example log line")
}

func Test_getLevels(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		level  string
		err    bool
		levels []logrus.Level
	}{
		{
			level: "info",
			err:   false,
			levels: []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.WarnLevel,
				logrus.InfoLevel,
			},
		},
		{
			level: "error",
			err:   false,
			levels: []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
			},
		},
		{
			level:  "tea",
			err:    true,
			levels: nil,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.level, func(t *testing.T) {
			t.Parallel()

			levels, err := parseLevels(test.level)

			if test.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.levels, levels)
		})
	}
}
