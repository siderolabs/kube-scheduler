package main

import (
	"os"

	"k8s.io/component-base/cli"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	_ "github.com/siderolabs/kube-scheduler/apis/config/scheme"
	"github.com/siderolabs/kube-scheduler/pkg/plugins/emissions"
)

func main() {
	command := app.NewSchedulerCommand(app.WithPlugin(emissions.Name, emissions.New))
	code := cli.Run(command)
	os.Exit(code)
}
