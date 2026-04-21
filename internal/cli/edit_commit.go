package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/googleapi"

	"github.com/AndroidPoet/playconsole-cli/internal/api"
	"github.com/AndroidPoet/playconsole-cli/internal/output"
)

type EditSubmissionMode string

const (
	EditSubmissionModeLive  EditSubmissionMode = "live"
	EditSubmissionModeStage EditSubmissionMode = "stage"
	EditSubmissionModeOpen  EditSubmissionMode = "open"
)

const stageFlagUsage = "commit the edit and stage changes in Play Console without sending for review"

type EditSubmission struct {
	Mode          EditSubmissionMode
	CommitOptions api.CommitOptions
	LeaveOpen     bool
}

type editCommitter interface {
	ID() string
	Commit() error
	CommitWithOptions(options api.CommitOptions) error
}

// AddStageFlag adds the --stage flag to cmd.
func AddStageFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("stage", false, stageFlagUsage)
}

func ParseEditSubmissionMode(value string) (EditSubmissionMode, error) {
	switch mode := EditSubmissionMode(strings.ToLower(value)); mode {
	case EditSubmissionModeLive, EditSubmissionModeStage, EditSubmissionModeOpen:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid --edit-mode %q: must be one of live, stage, open", value)
	}
}

func GetEditSubmission(cmd *cobra.Command, allowOpen bool) (EditSubmission, error) {
	mode, err := resolveEditSubmissionMode(cmd)
	if err != nil {
		return EditSubmission{}, err
	}

	if mode == EditSubmissionModeOpen && !allowOpen {
		return EditSubmission{}, fmt.Errorf("--edit-mode=open is not supported for %s; use --edit-mode=live or --edit-mode=stage", cmd.CommandPath())
	}

	submission := EditSubmission{
		Mode:      mode,
		LeaveOpen: mode == EditSubmissionModeOpen,
	}
	if mode == EditSubmissionModeStage {
		submission.CommitOptions = api.CommitOptions{ChangesNotSentForReview: true}
	}

	return submission, nil
}

func ApplyEditSubmission(edit *api.Edit, submission EditSubmission) error {
	return applyEditSubmission(edit, submission)
}

func applyEditSubmission(edit editCommitter, submission EditSubmission) error {
	if submission.LeaveOpen {
		output.PrintEditOpen(edit.ID())
		return nil
	}

	if err := edit.CommitWithOptions(submission.CommitOptions); err != nil {
		if submission.Mode == EditSubmissionModeStage && isAutoReviewCommitError(err) {
			output.PrintWarning("Play Console auto-sends these changes for review; retrying commit without staging")
			if retryErr := edit.Commit(); retryErr != nil {
				return retryErr
			}
			output.PrintSuccess("Edit committed (Play Console sent changes for review automatically)")
			return nil
		}
		return err
	}

	output.PrintEditCommitSuccess(submission.Mode == EditSubmissionModeStage)
	return nil
}

func isAutoReviewCommitError(err error) bool {
	var apiErr *googleapi.Error
	ok := errors.As(err, &apiErr)
	if !ok {
		return false
	}
	if apiErr.Code != 400 {
		return false
	}

	return strings.Contains(apiErr.Message, "Changes are sent for review automatically") &&
		strings.Contains(apiErr.Message, "changesNotSentForReview must not be set")
}

func resolveEditSubmissionMode(cmd *cobra.Command) (EditSubmissionMode, error) {
	var (
		selectedMode   EditSubmissionMode
		selectedSource string
	)

	recordMode := func(mode EditSubmissionMode, source string) error {
		if selectedSource != "" && selectedMode != mode {
			return fmt.Errorf("conflicting edit submission flags: %s conflicts with %s; use a single --edit-mode=live|stage|open flag", selectedSource, source)
		}
		selectedMode = mode
		selectedSource = source
		return nil
	}

	if flag := cmd.Flags().Lookup("edit-mode"); flag != nil && flag.Changed {
		mode, err := ParseEditSubmissionMode(flag.Value.String())
		if err != nil {
			return "", err
		}
		if err := recordMode(mode, "--edit-mode="+string(mode)); err != nil {
			return "", err
		}
	}

	if flag := cmd.Flags().Lookup("stage"); flag != nil && flag.Changed {
		stage, err := cmd.Flags().GetBool("stage")
		if err != nil {
			return "", err
		}
		if stage {
			if err := recordMode(EditSubmissionModeStage, "--stage"); err != nil {
				return "", err
			}
		}
	}

	if flag := cmd.Flags().Lookup("commit"); flag != nil && flag.Changed {
		autoCommit, err := cmd.Flags().GetBool("commit")
		if err != nil {
			return "", err
		}
		if !autoCommit {
			if err := recordMode(EditSubmissionModeOpen, "--commit=false"); err != nil {
				return "", err
			}
		}
	}

	if selectedSource != "" {
		return selectedMode, nil
	}

	if viper.IsSet("edit-mode") {
		return ParseEditSubmissionMode(viper.GetString("edit-mode"))
	}

	return EditSubmissionModeLive, nil
}
