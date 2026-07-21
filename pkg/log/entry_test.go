package log

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomods/athens/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestEntrySystemErr_AthensError(t *testing.T) {
	lggr := New("none", logrus.DebugLevel, "json")
	var buf bytes.Buffer
	lggr.Out = &buf

	athensErr := errors.E("test.op", fmt.Errorf("something went wrong"), errors.M("github.com/gomods/athens"), errors.V("v1.0.0"))
	lggr.SystemErr(athensErr)

	output := buf.String()
	require.Contains(t, output, "something went wrong")
	require.Contains(t, output, "test.op")
}

func TestEntrySystemErr_RegularError(t *testing.T) {
	lggr := New("none", logrus.DebugLevel, "json")
	var buf bytes.Buffer
	lggr.Out = &buf

	lggr.SystemErr(fmt.Errorf("regular error"))

	output := buf.String()
	require.Contains(t, output, "regular error")
}

func TestEntryWithFields(t *testing.T) {
	lggr := New("none", logrus.DebugLevel, "json")
	var buf bytes.Buffer
	lggr.Out = &buf

	e := lggr.WithFields(map[string]any{"key": "value"})
	e.Infof("test message")

	output := buf.String()
	require.Contains(t, output, "key")
	require.Contains(t, output, "value")
}

func TestErrFields(t *testing.T) {
	err := errors.E("test.op", fmt.Errorf("test error"), errors.M("mymod"), errors.V("v1.0.0"))
	var athensErr errors.Error
	errors.AsErr(err, &athensErr)

	fields := errFields(athensErr)
	require.Equal(t, errors.Op("test.op"), fields["operation"])
	require.Equal(t, errors.M("mymod"), fields["module"])
	require.Equal(t, errors.V("v1.0.0"), fields["version"])
}

func TestLoggerSystemErr(t *testing.T) {
	lggr := New("none", logrus.DebugLevel, "json")
	var buf bytes.Buffer
	lggr.Out = &buf

	lggr.SystemErr(fmt.Errorf("logger system error"))
	output := buf.String()
	require.Contains(t, output, "logger system error")
}

func TestSortFields(t *testing.T) {
	fields := logrus.Fields{
		"z_field": "z",
		"a_field": "a",
		"m_field": "m",
	}
	sorted := sortFields(fields)
	require.Equal(t, []string{"a_field", "m_field", "z_field"}, sorted)
}

func TestSortFields_Empty(t *testing.T) {
	fields := logrus.Fields{}
	sorted := sortFields(fields)
	require.Empty(t, sorted)
}
