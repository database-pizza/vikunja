# Workspace Task Tracking

The database.pizza workspace uses Vikunja at
`https://tasks.database.pizza` to track work. The API credential and base URL
are local-only values in the workspace-root `.env`; never add either to tracked
files, logs, or command output.

Run these commands from the workspace root after loading the local environment:

```sh
set -a
. ./.env
set +a
```

## List Tasks

List available projects to obtain a project ID:

```sh
curl -fsS \
  -H "Authorization: Bearer $VIKUNJA_API_TOKEN" \
  "$VIKUNJA_API_URL/projects?per_page=100" |
  jq -r '.items[] | "\(.id)\t\(.title)"'
```

List incomplete tasks across all projects:

```sh
curl -fsSG \
  -H "Authorization: Bearer $VIKUNJA_API_TOKEN" \
  --data-urlencode 'filter=done = false' \
  --data-urlencode 'per_page=100' \
  "$VIKUNJA_API_URL/tasks" |
  jq -r '.items[] | "\(.id)\t\(.title)"'
```

## Create A Task

Replace `<project-id>` and the title with the intended values:

```sh
curl -fsS -X POST \
  -H "Authorization: Bearer $VIKUNJA_API_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"title":"Task title"}' \
  "$VIKUNJA_API_URL/projects/<project-id>/tasks"
```

## Complete A Task

Replace `<task-id>` with the numeric task ID:

```sh
curl -fsS -X PATCH \
  -H "Authorization: Bearer $VIKUNJA_API_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"done":true}' \
  "$VIKUNJA_API_URL/tasks/<task-id>"
```

Create, modify, or complete tasks only when the user explicitly asks. Reading
task status is safe when it directly answers the user's request.
