package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/goccy/go-yaml"
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

// run controls the main flow of the command.
func (o *options) run(ctx context.Context) error {
	// if o.outdir does not exist, create it
	if _, err := os.Stat(o.outdir); os.IsNotExist(err) {
		if err := os.Mkdir(o.outdir, 0755); err != nil {
			return fmt.Errorf("failed to create a directory %s: %v", o.outdir, err)
		}
		fmt.Printf("Created a directory %s\n", o.outdir)
	}

	client, err := newDefaultECSClient()
	if err != nil {
		return fmt.Errorf("failed to create ECS client: %v", err)
	}

	serviceArns, err := client.ListServices(ctx, o.cluster)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d services\n", len(serviceArns))

	for i, serviceArn := range serviceArns {
		service, err := client.DescribeService(ctx, o.cluster, serviceArn)
		if err != nil {
			return err
		}

		taskDef, err := client.DescribeTaskDefinition(ctx, *service.TaskDefinition)
		if err != nil {
			return err
		}

		formatService(service)
		formatTaskDefinition(taskDef)

		if err := o.export(service, taskDef); err != nil {
			return err
		}

		fmt.Printf(" %d. Export succeeded: %s\n", (i + 1), *service.ServiceName)
	}

	fmt.Println("Successfully finished exporting.")
	return nil
}

// export exports the service and the taskDefinition as yaml files.
// The file names are {outdir}/{serviceName}/servicedef.yaml and {outdir}/{serviceName}/taskdef.yaml.
func (o *options) export(service *types.Service, taskdef *types.TaskDefinition) error {
	// create a directory
	dir := fmt.Sprintf("%s/%s", o.outdir, *service.ServiceName)
	if err := os.Mkdir(dir, 0755); err != nil {
		return fmt.Errorf("failed to create a directory %s: %w", *service.ServiceName, err)
	}

	// Service
	svcBytes, err := yaml.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service %s: %w", *service.ServiceName, err)
	}
	svcPath := fmt.Sprintf("%s/servicedef.yaml", dir)
	if err := os.WriteFile(svcPath, svcBytes, 0644); err != nil {
		return fmt.Errorf("failed to write service yaml file of %s: %w", *service.ServiceName, err)
	}

	// Task Definition
	tdBytes, err := yaml.Marshal(taskdef)
	if err != nil {
		return fmt.Errorf("failed to marshal taskDefinition %s: %w", *taskdef.Family, err)
	}
	tdPath := fmt.Sprintf("%s/taskdef.yaml", dir)
	if err := os.WriteFile(tdPath, tdBytes, 0644); err != nil {
		return fmt.Errorf("failed to write taskDefinition yaml file of %s: %w", *taskdef.Family, err)
	}

	return nil
}

// formatService removes unnecessary fields.
func formatService(service *types.Service) {
	service.CreatedAt = nil
	service.CreatedBy = nil
	service.Deployments = nil
	service.Events = nil

	service.LoadBalancers = nil

	service.PendingCount = 0
	service.RunningCount = 0
	service.Status = nil
	service.TaskSets = nil
}

// formatTaskDefinition removes unnecessary fields.
func formatTaskDefinition(taskDef *types.TaskDefinition) {
	taskDef.DeregisteredAt = nil
	taskDef.RegisteredAt = nil
	taskDef.RegisteredBy = nil

	// taskDef.Status = nil // TODO: how to remove this field?
}

type ecsClient struct {
	client *ecs.Client
}

// newDefaultECSClient creates a new ECS Client with SDK's default configuration.
func newDefaultECSClient() (*ecsClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		fmt.Printf("unable to load SDK's default config, %v", err)
		return nil, err
	}

	return &ecsClient{client: ecs.NewFromConfig(cfg)}, nil
}

// ListServices lists services in the cluster.
func (c *ecsClient) ListServices(ctx context.Context, cluster string) ([]string, error) {
	res, err := c.client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: &cluster,
	})
	if err != nil {
		return nil, err
	}
	return res.ServiceArns, nil
}

// DescribeService describes the service.
func (c *ecsClient) DescribeService(ctx context.Context, cluster, service string) (*types.Service, error) {
	res, err := c.client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
		Include:  []types.ServiceField{types.ServiceFieldTags},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Services) == 0 {
		return nil, fmt.Errorf("service %s not found", service)
	}
	return &res.Services[0], nil
}

// DescribeTaskDefinition describes the taskDefinition.
func (c *ecsClient) DescribeTaskDefinition(ctx context.Context, taskDefArn string) (*types.TaskDefinition, error) {
	res, err := c.client.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: &taskDefArn,
		Include:        []types.TaskDefinitionField{types.TaskDefinitionFieldTags},
	})
	if err != nil {
		return nil, err
	}
	return res.TaskDefinition, nil
}
