package client // import "github.com/docker/docker/client"

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/errdefs"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestContainerStopError(t *testing.T) {
	client := &Client{
		client: newMockClient(errorMock(http.StatusInternalServerError, "Server error")),
	}
	err := client.ContainerStop(context.Background(), "container_id", container.StopOptions{})
	assert.Check(t, is.ErrorType(err, errdefs.IsSystem))

	err = client.ContainerStop(context.Background(), "", container.StopOptions{})
	assert.Check(t, is.ErrorType(err, errdefs.IsInvalidParameter))
	assert.Check(t, is.ErrorContains(err, "value is empty"))

	err = client.ContainerStop(context.Background(), "    ", container.StopOptions{})
	assert.Check(t, is.ErrorType(err, errdefs.IsInvalidParameter))
	assert.Check(t, is.ErrorContains(err, "value is empty"))
}

// TestContainerStopConnectionError verifies that connection errors occurring
// during API-version negotiation are not shadowed by API-version errors.
//
// Regression test for https://github.com/docker/cli/issues/4890
func TestContainerStopConnectionError(t *testing.T) {
	client, err := NewClientWithOpts(WithAPIVersionNegotiation(), WithHost("tcp://no-such-host.invalid"))
	assert.NilError(t, err)

	err = client.ContainerStop(context.Background(), "container_id", container.StopOptions{})
	assert.Check(t, is.ErrorType(err, IsErrConnectionFailed))
}

func TestContainerStop(t *testing.T) {
	const expectedURL = "/v1.42/containers/container_id/stop"
	client := &Client{
		client: newMockClient(func(req *http.Request) (*http.Response, error) {
			if !strings.HasPrefix(req.URL.Path, expectedURL) {
				return nil, fmt.Errorf("expected URL '%s', got '%s'", expectedURL, req.URL)
			}
			s := req.URL.Query().Get("signal")
			if s != "SIGKILL" {
				return nil, fmt.Errorf("signal not set in URL query. Expected 'SIGKILL', got '%s'", s)
			}
			t := req.URL.Query().Get("t")
			if t != "100" {
				return nil, fmt.Errorf("t (timeout) not set in URL query properly. Expected '100', got %s", t)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(""))),
			}, nil
		}),
		version: "1.42",
	}
	timeout := 100
	err := client.ContainerStop(context.Background(), "container_id", container.StopOptions{
		Signal:  "SIGKILL",
		Timeout: &timeout,
	})
	if err != nil {
		t.Fatal(err)
	}
}
