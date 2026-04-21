package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AndroidPoet/playconsole-cli/internal/api"
)

const stageFlagUsage = "commit the edit and stage changes in Play Console without sending for review"

func AddStageFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("stage", false, stageFlagUsage)
}

func GetCommitOptions(cmd *cobra.Command) (api.CommitOptions, error) {
	stage, err := cmd.Flags().GetBool("stage")
	if err != nil {
		return api.CommitOptions{}, err
	}

	if !stage {
		return api.CommitOptions{}, nil
	}

	if cmd.Flags().Lookup("commit") != nil {
		autoCommit, err := cmd.Flags().GetBool("commit")
		if err != nil {
			return api.CommitOptions{}, err
		}
		if !autoCommit {
			return api.CommitOptions{}, fmt.Errorf("--stage requires --commit=true")
		}
	}

	return api.CommitOptions{ChangesNotSentForReview: true}, nil
}
