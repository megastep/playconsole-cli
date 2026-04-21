package bundles

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/AndroidPoet/playconsole-cli/internal/cli"
)

func TestRunUploadRejectsStageWithoutCommit(t *testing.T) {
	cli.SetPackageName("com.example.app")
	t.Cleanup(func() {
		cli.SetPackageName("")
	})

	cmd := &cobra.Command{Use: "upload"}
	cmd.Flags().Bool("commit", true, "")
	cli.AddStageFlag(cmd)

	if err := cmd.Flags().Set("commit", "false"); err != nil {
		t.Fatalf("set commit flag: %v", err)
	}
	if err := cmd.Flags().Set("stage", "true"); err != nil {
		t.Fatalf("set stage flag: %v", err)
	}

	err := runUpload(cmd, nil)
	wantErr := "conflicting edit submission flags: --stage conflicts with --commit=false; use a single --edit-mode=live|stage|open flag"
	if err == nil || err.Error() != wantErr {
		t.Fatalf("runUpload() error = %v, want %q", err, wantErr)
	}
}
