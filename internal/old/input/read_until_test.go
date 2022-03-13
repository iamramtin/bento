package input

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/manager/mock"
	"github.com/benthosdev/benthos/v4/internal/message"
)

func TestReadUntilErrs(t *testing.T) {
	conf := NewConfig()
	conf.Type = TypeReadUntil

	inConf := NewConfig()
	conf.ReadUntil.Input = &inConf

	_, err := New(conf, mock.NewManager(), log.Noop(), metrics.Noop())
	assert.EqualError(t, err, "failed to create input 'read_until': a check query is required")
}

func TestReadUntilInput(t *testing.T) {
	content := []byte(`foo
bar
baz`)

	tmpfile, err := os.CreateTemp("", "benthos_read_until_test")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	inconf := NewConfig()
	inconf.Type = "file"
	inconf.File.Paths = []string{tmpfile.Name()}

	t.Run("ReadUntilBasic", func(te *testing.T) {
		testReadUntilBasic(inconf, te)
	})
	t.Run("ReadUntilRetry", func(te *testing.T) {
		testReadUntilRetry(inconf, te)
	})
}

func testReadUntilBasic(inConf Config, t *testing.T) {
	tCtx, done := context.WithTimeout(context.Background(), time.Second*5)
	defer done()

	rConf := NewConfig()
	rConf.Type = "read_until"
	rConf.ReadUntil.Input = &inConf
	rConf.ReadUntil.Check = `content() == "bar"`

	in, err := New(rConf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	expMsgs := []string{
		"foo",
		"bar",
	}

	for i, expMsg := range expMsgs {
		var tran message.Transaction
		var open bool
		select {
		case tran, open = <-in.TransactionChan():
			if !open {
				t.Fatal("transaction chan closed")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}

		if exp, act := expMsg, string(tran.Payload.Get(0).Get()); exp != act {
			t.Errorf("Wrong message contents: %v != %v", act, exp)
		}
		if i == len(expMsgs)-1 {
			if exp, act := "final", tran.Payload.Get(0).MetaGet("benthos_read_until"); exp != act {
				t.Errorf("Metadata missing from final message: %v != %v", act, exp)
			}
		} else if exp, act := "", tran.Payload.Get(0).MetaGet("benthos_read_until"); exp != act {
			t.Errorf("Metadata final message metadata added to non-final message: %v", act)
		}
		require.NoError(t, tran.Ack(tCtx, nil))
	}

	// Should close automatically now
	select {
	case _, open := <-in.TransactionChan():
		if open {
			t.Fatal("transaction chan not closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if err = in.WaitForClose(time.Second); err != nil {
		t.Fatal(err)
	}
}

func testReadUntilRetry(inConf Config, t *testing.T) {
	tCtx, done := context.WithTimeout(context.Background(), time.Second*5)
	defer done()

	rConf := NewConfig()
	rConf.Type = "read_until"
	rConf.ReadUntil.Input = &inConf
	rConf.ReadUntil.Check = `content() == "bar"`

	in, err := New(rConf, mock.NewManager(), log.Noop(), metrics.Noop())
	if err != nil {
		t.Fatal(err)
	}

	expMsgs := map[string]struct{}{
		"foo": {},
		"bar": {},
	}

	var tran message.Transaction
	var open bool

	resFns := []func(context.Context, error) error{}
	i := 0
	for len(expMsgs) > 0 && i < 10 {
		// First try
		select {
		case tran, open = <-in.TransactionChan():
			if !open {
				t.Fatalf("transaction chan closed at %v", i)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}

		i++
		act := string(tran.Payload.Get(0).Get())
		if _, exists := expMsgs[act]; !exists {
			t.Errorf("Unexpected message contents '%v': %v", i, act)
		} else {
			delete(expMsgs, act)
		}
		{
			tmpTran := tran
			resFns = append(resFns, tmpTran.Ack)
		}
	}

	select {
	case <-in.TransactionChan():
		t.Error("Unexpected transaction")
		return
	case <-time.After(time.Millisecond * 500):
	}

	for _, rFn := range resFns {
		require.NoError(t, rFn(tCtx, errors.New("failed")))
	}

	expMsgs = map[string]struct{}{
		"foo": {},
		"bar": {},
		"baz": {},
	}

remainingLoop:
	for len(expMsgs) > 0 {
		// Second try
		select {
		case tran, open = <-in.TransactionChan():
			if !open {
				break remainingLoop
			}
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}

		act := string(tran.Payload.Get(0).Get())
		if _, exists := expMsgs[act]; !exists {
			t.Errorf("Unexpected message contents '%v': %v", i, act)
		} else {
			delete(expMsgs, act)
		}
		require.NoError(t, tran.Ack(tCtx, nil))
	}
	if len(expMsgs) == 3 {
		t.Error("Expected at least one extra message")
	}

	// Should close automatically now
	select {
	case _, open := <-in.TransactionChan():
		if open {
			t.Fatal("transaction chan not closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	if err = in.WaitForClose(time.Second); err != nil {
		t.Fatal(err)
	}
}
