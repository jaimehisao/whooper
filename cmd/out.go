package cmd

import (
	"fmt"
	"io"
	"sync"

	"github.com/spf13/cobra"
)

// cmdOutMu serializes writes to command stdout across sync progress
// goroutines and the service HTTP listen goroutine.
var cmdOutMu sync.Mutex

func withCmdOut(cmd *cobra.Command, fn func(w io.Writer)) {
	cmdOutMu.Lock()
	defer cmdOutMu.Unlock()
	fn(cmd.OutOrStdout())
}

func cmdFprintf(cmd *cobra.Command, format string, args ...any) {
	withCmdOut(cmd, func(w io.Writer) {
		fmt.Fprintf(w, format, args...)
	})
}

func cmdFprintln(cmd *cobra.Command, args ...any) {
	withCmdOut(cmd, func(w io.Writer) {
		fmt.Fprintln(w, args...)
	})
}
