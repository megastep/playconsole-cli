package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/googleapi"

	"github.com/AndroidPoet/playconsole-cli/internal/api"
)

func TestGetEditSubmissionDefaultsToLive(t *testing.T) {
	cmd := newEditSubmissionTestCommand()

	submission, err := GetEditSubmission(cmd, true)
	if err != nil {
		t.Fatalf("GetEditSubmission() error = %v", err)
	}
	if submission.Mode != EditSubmissionModeLive {
		t.Fatalf("Mode = %q, want %q", submission.Mode, EditSubmissionModeLive)
	}
	if submission.LeaveOpen {
		t.Fatalf("LeaveOpen = true, want false")
	}
	if submission.CommitOptions.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = true, want false")
	}
}

func TestGetEditSubmissionStageAlias(t *testing.T) {
	cmd := newEditSubmissionTestCommand()
	if err := cmd.Flags().Set("stage", "true"); err != nil {
		t.Fatalf("set stage flag: %v", err)
	}

	submission, err := GetEditSubmission(cmd, true)
	if err != nil {
		t.Fatalf("GetEditSubmission() error = %v", err)
	}
	if submission.Mode != EditSubmissionModeStage {
		t.Fatalf("Mode = %q, want %q", submission.Mode, EditSubmissionModeStage)
	}
	if !submission.CommitOptions.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = false, want true")
	}
}

func TestGetEditSubmissionOpenAlias(t *testing.T) {
	cmd := newEditSubmissionTestCommand()
	if err := cmd.Flags().Set("commit", "false"); err != nil {
		t.Fatalf("set commit flag: %v", err)
	}

	submission, err := GetEditSubmission(cmd, true)
	if err != nil {
		t.Fatalf("GetEditSubmission() error = %v", err)
	}
	if submission.Mode != EditSubmissionModeOpen {
		t.Fatalf("Mode = %q, want %q", submission.Mode, EditSubmissionModeOpen)
	}
	if !submission.LeaveOpen {
		t.Fatalf("LeaveOpen = false, want true")
	}
}

func TestGetEditSubmissionEditModeFlag(t *testing.T) {
	cmd := newEditSubmissionTestCommand()
	if err := cmd.Flags().Set("edit-mode", "stage"); err != nil {
		t.Fatalf("set edit-mode flag: %v", err)
	}

	submission, err := GetEditSubmission(cmd, true)
	if err != nil {
		t.Fatalf("GetEditSubmission() error = %v", err)
	}
	if submission.Mode != EditSubmissionModeStage {
		t.Fatalf("Mode = %q, want %q", submission.Mode, EditSubmissionModeStage)
	}
}

func TestGetEditSubmissionFallsBackToConfiguredEditMode(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("edit-mode", "stage")

	cmd := newEditSubmissionTestCommand()

	submission, err := GetEditSubmission(cmd, true)
	if err != nil {
		t.Fatalf("GetEditSubmission() error = %v", err)
	}
	if submission.Mode != EditSubmissionModeStage {
		t.Fatalf("Mode = %q, want %q", submission.Mode, EditSubmissionModeStage)
	}
	if !submission.CommitOptions.ChangesNotSentForReview {
		t.Fatalf("ChangesNotSentForReview = false, want true")
	}
}

func TestGetEditSubmissionRejectsConflictingFlags(t *testing.T) {
	testCases := []struct {
		name       string
		configure  func(*cobra.Command) error
		wantErrMsg string
	}{
		{
			name: "edit mode live conflicts with stage",
			configure: func(cmd *cobra.Command) error {
				if err := cmd.Flags().Set("edit-mode", "live"); err != nil {
					return err
				}
				return cmd.Flags().Set("stage", "true")
			},
			wantErrMsg: "conflicting edit submission flags: --edit-mode=live conflicts with --stage; use a single --edit-mode=live|stage|open flag",
		},
		{
			name: "stage conflicts with commit false",
			configure: func(cmd *cobra.Command) error {
				if err := cmd.Flags().Set("stage", "true"); err != nil {
					return err
				}
				return cmd.Flags().Set("commit", "false")
			},
			wantErrMsg: "conflicting edit submission flags: --stage conflicts with --commit=false; use a single --edit-mode=live|stage|open flag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newEditSubmissionTestCommand()
			if err := tc.configure(cmd); err != nil {
				t.Fatalf("configure flags: %v", err)
			}

			_, err := GetEditSubmission(cmd, true)
			if err == nil || err.Error() != tc.wantErrMsg {
				t.Fatalf("GetEditSubmission() error = %v, want %q", err, tc.wantErrMsg)
			}
		})
	}
}

func TestGetEditSubmissionRejectsOpenWhenUnsupported(t *testing.T) {
	cmd := newEditSubmissionTestCommand()
	if err := cmd.Flags().Set("edit-mode", "open"); err != nil {
		t.Fatalf("set edit-mode flag: %v", err)
	}

	_, err := GetEditSubmission(cmd, false)
	wantErr := "--edit-mode=open is not supported for test; use --edit-mode=live or --edit-mode=stage"
	if err == nil || err.Error() != wantErr {
		t.Fatalf("GetEditSubmission() error = %v, want %q", err, wantErr)
	}
}

func TestParseEditSubmissionModeRejectsInvalidValue(t *testing.T) {
	_, err := ParseEditSubmissionMode("draft")
	wantErr := `invalid --edit-mode "draft": must be one of live, stage, open`
	if err == nil || err.Error() != wantErr {
		t.Fatalf("ParseEditSubmissionMode() error = %v, want %q", err, wantErr)
	}
}

func TestApplyEditSubmissionFallsBackWhenStageIsAutoReviewed(t *testing.T) {
	edit := &fakeEditCommitter{
		id: "edit-123",
		commitWithOptionsErr: &googleapi.Error{
			Code:    400,
			Message: "Changes are sent for review automatically. The query parameter changesNotSentForReview must not be set.",
		},
	}

	err := applyEditSubmission(edit, EditSubmission{
		Mode:          EditSubmissionModeStage,
		CommitOptions: api.CommitOptions{ChangesNotSentForReview: true},
	})
	if err != nil {
		t.Fatalf("applyEditSubmission() error = %v", err)
	}
	if edit.commitWithOptionsCalls != 1 {
		t.Fatalf("CommitWithOptions call count = %d, want 1", edit.commitWithOptionsCalls)
	}
	if !edit.lastCommitOptions.ChangesNotSentForReview {
		t.Fatalf("last CommitWithOptions.ChangesNotSentForReview = false, want true")
	}
	if edit.commitCalls != 1 {
		t.Fatalf("Commit call count = %d, want 1", edit.commitCalls)
	}
}

func TestIsAutoReviewCommitError(t *testing.T) {
	err := &googleapi.Error{
		Code:    400,
		Message: "Changes are sent for review automatically. The query parameter changesNotSentForReview must not be set.",
	}
	if !isAutoReviewCommitError(err) {
		t.Fatalf("isAutoReviewCommitError() = false, want true")
	}
}

func newEditSubmissionTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("edit-mode", "live", "")
	cmd.Flags().Bool("commit", true, "")
	AddStageFlag(cmd)
	return cmd
}

type fakeEditCommitter struct {
	id                     string
	commitCalls            int
	commitErr              error
	commitWithOptionsCalls int
	commitWithOptionsErr   error
	lastCommitOptions      api.CommitOptions
}

func (f *fakeEditCommitter) ID() string {
	return f.id
}

func (f *fakeEditCommitter) Commit() error {
	f.commitCalls++
	return f.commitErr
}

func (f *fakeEditCommitter) CommitWithOptions(options api.CommitOptions) error {
	f.commitWithOptionsCalls++
	f.lastCommitOptions = options
	return f.commitWithOptionsErr
}
