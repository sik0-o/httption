package httption

import (
	"net/http"
	"testing"
	"time"

	"go.uber.org/zap"
)

type TestActionResp struct {
	CreatedAt string
	Proxy     string
}

type TestAction struct {
	BaseAction
}

func NewTestAction(client *http.Client) TestAction {
	ta := TestAction{
		*NewBaseAction(
			client,
			"GET",
			"http://localhost:4488/api/proxy",
		),
	}

	ta.name = "TestAction"

	// ta.needRetry = true

	return ta
}

func (da *TestAction) Do(opts ...Option) error {
	maxRetry := uint(5)
	opts = append(opts, WithRetry(&maxRetry, nil, nil), WithRetryDelay(2*time.Second))

	return da.BaseAction.Do(opts...)
}

func (da *TestAction) Result(result *TestActionResp) error {
	if err := da.BaseAction.Result(result); err != nil {
		return err
	}

	return nil
}

func wzp(f func(options ...zap.Option) (*zap.Logger, error)) *zap.Logger {
	l, err := f()
	if err == nil {
		return l
	}

	return nil
}

func Test_TestAction(t *testing.T) {
	httpClient := &http.Client{}
	ta := NewTestAction(httpClient)

	if err := ta.Do(WithLogger(wzp(zap.NewProduction))); err != nil {
		t.Fatal(err)
	}

	taResp := &TestActionResp{}
	if err := ta.Result(taResp); err != nil {
		t.Fatal(err)
	}

	t.Log("TestAction Response", taResp)
}
