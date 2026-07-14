// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import "os"

// osExit indirects os.Exit so tests can verify the requested exit code
// without actually terminating the test process. Production callers
// behave exactly as before.
var osExit = os.Exit
