package main

import "time"

type Incident struct {
	ID, Title, Priority, Namespace, Deployment, Team, Description, Cluster string
	StartTime  time.Time
	Tags       []string
	RawDetails map[string]string
}

type Pod struct {
	Name, Namespace, Status, Ready string
}
