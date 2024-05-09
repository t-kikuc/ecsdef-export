package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type options struct {
	cluster string
	outdir  string
}

func main() {
	cmd := newCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "",
		Short: "Fetch configs of services and taskDefinitions from AWS and export them as yaml files",
		RunE: func(c *cobra.Command, args []string) error {
			return o.run(context.Background())
		},
	}

	cmd.Flags().StringVar(&o.cluster, "cluster", "", "The name of the ECS cluster to list services (required)")
	cmd.MarkFlagRequired("cluster")
	cmd.Flags().StringVar(&o.outdir, "outdir", "", "The root directory for output files (required)")
	cmd.MarkFlagRequired("outdir")

	return cmd
}

func (o *options) run(ctx context.Context) error {
	return nil
}
