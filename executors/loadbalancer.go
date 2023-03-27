package executors

import (
	"context"
)

type Condition struct {
	Field  string   `json:"Field"`
	Values []string `json:"Values"`
}
type Action struct {
	TargetGroupName string `json:"TargetGroupName"`
	Type            string `json:"Type"`
}

type Rule struct {
	Conditions []Condition `json:"Conditions"`
	Actions    []Action    `json:"Actions"`
	Priority   int         `json:"Priority"`
}

func ExecuteLoadbalancer(dir string, vars map[string]any, ctx context.Context) (resultChan <-chan TaskResult) {
	play := "loadbalancer.yml"
	return Execute(dir+"/ansible/"+play, vars, ctx)
}
