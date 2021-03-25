package main

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "gin-demo",
		Short: "gin-demo - launch a simple gin-demo instance that serves some endpoints",
		Long:  ``,
		RunE:  run,
	}

	return &cmd
}

func main() {
	cmd := NewCmd()

	ctx := context.Background()

	if err := cmd.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ccmd *cobra.Command, args []string) error {
	fmt.Println("hello world")

	return nil
}
