package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetCommitOptions(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	AddStageFlag(cmd)

	options, err := GetCommitOptions(cmd)
	if err != nil {
		t.Fatalf("GetCommitOptions() error = %v", err)
	}
	if options.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = true, want false")
	}

	if err := cmd.Flags().Set("stage", "true"); err != nil {
		t.Fatalf("set stage flag: %v", err)
	}

	options, err = GetCommitOptions(cmd)
	if err != nil {
		t.Fatalf("GetCommitOptions() error = %v", err)
	}
	if !options.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = false, want true")
	}
}

func TestGetCommitOptionsWithoutStageFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	options, err := GetCommitOptions(cmd)
	if err != nil {
		t.Fatalf("GetCommitOptions() error = %v", err)
	}
	if options.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = true, want false")
	}
}

func TestGetCommitOptionsRejectsStageWithoutCommit(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("commit", true, "")
	AddStageFlag(cmd)

	if err := cmd.Flags().Set("commit", "false"); err != nil {
		t.Fatalf("set commit flag: %v", err)
	}
	if err := cmd.Flags().Set("stage", "true"); err != nil {
		t.Fatalf("set stage flag: %v", err)
	}

	_, err := GetCommitOptions(cmd)
	if err == nil || err.Error() != "--stage requires --commit=true" {
		t.Fatalf("GetCommitOptions() error = %v, want %q", err, "--stage requires --commit=true")
	}
}
