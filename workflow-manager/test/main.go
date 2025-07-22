package main

import (
	"log"

	"github.com/coredgeio/gosdkclient"

	"github.com/coredgeio/workflow-manager/api/workflow"
)

func main() {
	c, _ := gosdkclient.CreateSdkClient("62d0150e-f59a-4ee9-90ec-d4c798bc5746",
		"CLQGchvDGW9PNZtM0dE3BCvzVG0TB8L-vuAAH4hs8e",
		"http://192.168.100.173:31210")
	client := workflow.NewModuleApiSdkClient(c)
	req := &workflow.ModulesListReq{
		Domain:  "default",
		Project: "default-project",
	}
	resp, err := client.ListModules(req)
	if err != nil {
		log.Fatalln("failed to get list of available modules", err)
	}
	log.Println("got resp", resp)
}
