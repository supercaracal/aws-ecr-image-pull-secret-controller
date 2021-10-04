package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	controllers "github.com/supercaracal/kubernetes-controller-template/internal/controller"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	cfg, err := buildConfig(masterURL, kubeconfig)
	if err != nil {
		klog.Fatal("Error building kubernetes config: ", err)
	}

	ctrl, err := controllers.NewCustomController(cfg)
	if err != nil {
		klog.Fatal("Error building custom controller: ", err)
	}

	if err := ctrl.Run(setUpSignalHandler()); err != nil {
		klog.Fatal("Error running controller: ", err)
	}
}

func init() {
	flag.StringVar(
		&masterURL,
		"master",
		"",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.",
	)

	flag.StringVar(
		&kubeconfig,
		"kubeconfig",
		"",
		"Path to a kubeconfig. Only required if out-of-cluster.",
	)
}

func buildConfig(masterURL, kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		return rest.InClusterConfig()
	}

	return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
}

func setUpSignalHandler() <-chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1)
	}()

	return stop
}
