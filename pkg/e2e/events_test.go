//go:build e2e

/*
   Copyright 2020 Docker Compose CLI authors

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

package e2e

import (
	"testing"
	"time"

	"gotest.tools/v3/icmd"
)

func TestEventsUnknownService(t *testing.T) {
	c := NewParallelCLI(t)

	cmd := c.NewDockerComposeCmd(t, "-f", "./fixtures/simple-composefile/compose.yaml", "events", "not-a-service")
	cmd.Timeout = 10 * time.Second
	res := icmd.RunCmd(cmd)
	res.Assert(t, icmd.Expected{ExitCode: 1, Err: "no such service: not-a-service"})
}

// Regression test for https://github.com/docker/compose/issues/14442:
// `events --until` printed "EOF" and exited 1 when the Engine closed the
// stream at the deadline, because io.EOF was reported as a failure.
func TestEventsUntilExitsCleanly(t *testing.T) {
	until := time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339)
	NewScenario(t, "events --until must exit 0 when the Engine closes the stream at the deadline").
		Step("up starts a service to stream events from",
			ComposeCmd("up", "-d"),
			ServiceState("app", "running")).
		Step("events --until returns success once the deadline closes the stream",
			ComposeCmd("events", "--until", until).Within(30*time.Second),
			ExitCode(0))
}
