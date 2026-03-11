package main

import "time"

type Incident struct {
	ID, Title, Priority, Namespace, Deployment, Team, Description string
	StartTime  time.Time
	RawDetails map[string]string
}

type Pod struct {
	Name, Namespace, Status, Ready string
}
