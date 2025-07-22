package main

import (
	"log"

	"github.com/coredgeio/workflow-manager/pkg/workflow/builder"
)

func main() {
	wBuilder := &builder.WorkflowBuilder{
		Name:      "demo-12334we-das",
		Namespace: "workflow-manager",
		Nodes: map[string]*builder.WorkflowNodesType{
			"demo-module-1": {
				Image: "192.168.100.173:31210/catalog/58bfa8dc-4112-4676-89b8-0cd0e1cd5cac:latest",
				Inputs: []*builder.WorkflowInputType{
					{
						Name: "in1",
					},
					{
						Name: "in2",
					},
				},
				Outputs: []*builder.WorkflowOutputType{
					{
						Name:      "out1",
						ValueFrom: "/tmp/out1.txt",
					},
				},
			},
		},
		Steps: []*builder.WorkflowStepType{
			{
				Nodes: []*builder.WorkflowStepNode{
					{
						NodeId: "demo-module-1_1",
						Module: "demo-module-1",
						Inputs: []*builder.WorkflowStepInputType{
							{
								Name:  "in1",
								Value: "abc123",
							},
							{
								Name: "in2",
								Source: &builder.WorkflowStepInputSource{
									Source:    "test",
									SourceVar: "test-input",
								},
							},
						},
					},
				},
			},
			{
				Nodes: []*builder.WorkflowStepNode{
					{
						NodeId: "demo-module-1_2",
						Module: "demo-module-1",
						Inputs: []*builder.WorkflowStepInputType{
							{
								Name:  "in1",
								Value: "abc123",
							},
							{
								Name: "in2",
								Source: &builder.WorkflowStepInputSource{
									Source:    "test",
									SourceVar: "test-input",
								},
							},
						},
					},
				},
			},
		},
	}

	str, err := wBuilder.GetTemplate()
	if err != nil {
		log.Println("got error:", err)
	} else {
		log.Println("got file:", str)
	}
}
