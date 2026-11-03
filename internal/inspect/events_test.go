/*
   Copyright 2026 Docker Compose CLI authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package inspect

import (
	"errors"
	"io"
	"testing"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"

	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/mocks"
)

// The Engine client reports a completely read events stream as io.EOF on the
// error channel (see client.Events). The daemon closes the stream after the
// --until timestamp, so io.EOF is the normal end of `docker compose events`.
func TestEventsCleanStreamEndIsNotAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	apiClient := mocks.NewMockAPIClient(ctrl)
	errs := make(chan error, 1)
	errs <- io.EOF
	apiClient.EXPECT().Events(gomock.Any(), gomock.Any()).Return(client.EventsResult{
		Messages: make(chan events.Message),
		Err:      errs,
	})

	err := New(&fakeBackend{apiClient: apiClient}).Events(t.Context(), "proj", api.EventsOptions{
		Consumer: func(api.Event) error { return nil },
	})

	assert.NilError(t, err)
}

func TestEventsDeliversEventsBeforeCleanEnd(t *testing.T) {
	ctrl := gomock.NewController(t)
	apiClient := mocks.NewMockAPIClient(ctrl)

	messages := make(chan events.Message, 1)
	errs := make(chan error, 1)
	consumed := make(chan struct{})
	go func() {
		messages <- events.Message{
			Type:   events.ContainerEventType,
			Action: events.ActionStart,
			Actor: events.Actor{
				ID:         "abc",
				Attributes: map[string]string{api.ServiceLabel: "web"},
			},
		}
		// Only report the end of the stream once the event has been consumed,
		// so both channels are never ready at the same time.
		<-consumed
		errs <- io.EOF
	}()
	apiClient.EXPECT().Events(gomock.Any(), gomock.Any()).Return(client.EventsResult{
		Messages: messages,
		Err:      errs,
	})

	var got []api.Event
	err := New(&fakeBackend{apiClient: apiClient}).Events(t.Context(), "proj", api.EventsOptions{
		Consumer: func(event api.Event) error {
			got = append(got, event)
			close(consumed)
			return nil
		},
	})

	assert.NilError(t, err)
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].Service, "web")
	assert.Equal(t, got[0].Container, "abc")
	assert.Equal(t, got[0].Status, "start")
}

func TestEventsRealErrorIsReturned(t *testing.T) {
	ctrl := gomock.NewController(t)
	apiClient := mocks.NewMockAPIClient(ctrl)
	errs := make(chan error, 1)
	errs <- errors.New("connection reset")
	apiClient.EXPECT().Events(gomock.Any(), gomock.Any()).Return(client.EventsResult{
		Messages: make(chan events.Message),
		Err:      errs,
	})

	err := New(&fakeBackend{apiClient: apiClient}).Events(t.Context(), "proj", api.EventsOptions{})

	assert.ErrorContains(t, err, "connection reset")
}
