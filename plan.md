# WAR Operator TUI

Team: Apocalypse Now

We want a Golang CLI using cobra and bubble tea to make it interactive.

## User story

An incident breaks up in OpsGenie, the oncall guardian executes this CLI to operate the incident and start an investigation.
It launches de tool running `war`. The TUI lists all the ongoing incidents that the user can view, ordered by priority and from oldest to newest.
The user selects one of the incidents from the list. The TUI shows the details, including:

- Title of the alert
- Key metadata like namespace, deployment/job, start time
- Extra information like description

Then he gets some available actions:

- Escalate: which allows the user to open a new alert on a different team
- Investigate: using the metadata, find the affected pods and stream their logs (use kubectl for this)
- Create war: create a new Slack channel with this format `war-<team>-<user_input>` (infer the team from the metadata and ask for user input)

## Plan
Create a plan to implement this tool.

Define the features and create a spec file per feature.

Use the askUserTool to prompt the user for relevant info.
