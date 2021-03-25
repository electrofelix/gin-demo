package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/electrofelix/gin-demo/controller"
	"github.com/electrofelix/gin-demo/server"
	"github.com/electrofelix/gin-demo/service"
)

func NewCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:          "gin-demo",
		Short:        "gin-demo - launch a simple gin-demo instance that serves some endpoints",
		Long:         ``,
		SilenceUsage: true,
		RunE:         run,
	}

	return &cmd
}

func main() {
	cmd := NewCmd()

	// should be replaced with a context that triggers graceful shutdown of any
	// requests/responses in flight based on signals.
	ctx := context.Background()

	if err := cmd.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}

func loadAWSConfig(ccmd *cobra.Command) (aws.Config, error) {
	// this is a bit horrible, but it works for now.
	// ideally should use viper to pick up parts from the
	// environment, or unpick whats required to set the
	// endpoint from the env
	cfg, err := config.LoadDefaultConfig(
		ccmd.Context(),
		config.WithRegion("us-west-2"),
		config.WithEndpointResolver(
			aws.EndpointResolverFunc(
				func(service, region string) (aws.Endpoint, error) {
					if service == dynamodb.ServiceID {
						return aws.Endpoint{
							URL: "http://localhost:8000",
						}, nil
					}

					return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
				},
			),
		),
		config.WithCredentialsProvider(
			aws.CredentialsProviderFunc(
				func(c context.Context) (aws.Credentials, error) {
					return aws.Credentials{
						AccessKeyID:     "AK1",
						SecretAccessKey: "SK1",
					}, nil
				},
			),
		),
	)
	if err != nil {
		fmt.Fprintf(ccmd.ErrOrStderr(), "unable to load SDK config, %v", err)

		return aws.Config{}, err
	}

	return cfg, nil
}

func run(ccmd *cobra.Command, args []string) error {

	awsCfg, err := loadAWSConfig(ccmd)
	dbClient := dynamodb.NewFromConfig(awsCfg)
	// table should be provided via a config option
	svc := service.New(dbClient, "user-table")

	err = svc.InitializeTable(ccmd.Context())
	if err != nil {
		return err
	}

	ctrl := controller.New(svc)
	s := server.New(ctrl)

	// register to allow some signals to provide a context that will indicate shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancelFunc := context.WithCancel(ccmd.Context())
	go func ()  {
		<-quit
		cancelFunc()
	}()

	return s.Start(ctx)
}
