package cmd

import (
	"bytes"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestCmdOutMutexSerializesProgressWrites(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cmdFprintf(cmd, "entity-%d: %d\n", n, n)
		}(i)
	}
	wg.Wait()

	if buf.Len() == 0 {
		t.Fatal("expected progress output")
	}
}
