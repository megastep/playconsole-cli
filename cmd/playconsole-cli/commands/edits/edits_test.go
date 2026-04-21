package edits

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/AndroidPoet/playconsole-cli/internal/cli"
)

func TestRunCommitRejectsOpenEditMode(t *testing.T) {
	cli.SetPackageName("com.example.app")
	t.Cleanup(func() {
		cli.SetPackageName("")
	})

	editID = "edit-123"

	cmd := &cobra.Command{Use: "commit"}
	cmd.Flags().String("edit-mode", "live", "")
	if err := cmd.Flags().Set("edit-mode", "open"); err != nil {
		t.Fatalf("set edit-mode flag: %v", err)
	}

	err := runCommit(cmd, nil)
	wantErr := "--edit-mode=open is not supported for commit; use --edit-mode=live or --edit-mode=stage"
	if err == nil || err.Error() != wantErr {
		t.Fatalf("runCommit() error = %v, want %q", err, wantErr)
	}
}
