# Platform Changelog API

## Overview

[Platform Changelog](https://changelog.stage.devshift.net) is a system for tracking changes as they occur
across the platform through different types of events, such as
webhooks, Tekton task runs, and Jenkins jobs.

This API provides JSON responses to the requesting entity, mainly the [Platform
Changelog Frontend](https://www.github.com/redhatinsights/platform-changelog).

Github and Gitlab webhooks are authenticated via secret token as described in the Git api.

## Architecture

Platform Changelog is a backend API that uses a database for storing
incoming events. The current implementation supports a Postgres database
and responds to incoming requests with JSON responses.

This app is intended to be used behind authentication such as OAuth Proxy. This allows the app to be publicly accessible on changelog.devshift.net, while also being authenticated. The setup for this is in the frontend repo.

Our App-SRE has created tooling for connecting to commit and deployment events 
(as designated in changelog as timelines).

On each commit merged to a monitored branch, we recieve a request from the corresponding Jenkins job.

On every deployment, we use a Tekton task to recieve informatin on the app including:
- App
- Namespace
- Environment
- Timestamp of completetd deployment
- Commit ref (in the future)

## REST API Endpoint

Refer to the [OpenAPI](https://github.com/RedHatInsights/platform-changelog-go/blob/main/schema/openapi.yaml) for parameter details.

### Services
`/api/v1/services/`
Gets a list of services with their most recent commit and deployment
`/api/v1/services/{name}`
Gets all of a services fields
`/api/v1/services/{name}/timelines`; `/api/v1/services/{name}/commits`; `/api/v1/services/{name}/deploys`
Gets all of a service's timelines (commits and deployments) or commits or deployments only

### Timelines
`/api/v1/commits/`
Gets all commits
`/api/v1/deploys/`
Gets all deploys
`/api/v1/timelines/`
Gets all timelines (commits and deployments)


### Posting Timelines
`/api/v1/github`
Sends commit information. Follow `make test-github` for an example request.
`/api/v1/tekton`
Sends deployment information. Follow `make test-tekton-task` for an example request.

`/api/v1/github-webhook`
Sends github commits from a webhook; authentication needed (as per Github api).
Follow the Makefile's `make test-github-webhook` for usage.
`/api/v1/gitlab-webhook`
Sends gitlab commits from a webhook; authentication needed (as per Github api)
Follow the Makefile's `make test-gitlab-webhook` for usage.

### Deleting data
The app has no DELETE requests; instead, we use a [cron-job](https://github.com/RedHatInsights/platform-changelog-go/blob/main/tools/cron-job.sh) to remove old timelines.

## Onboarding a Service

If your service is deployed through App-sre, it will be onboarded automatically as information is recieved. If it is not, then manual onboarding with Github or Gitlab webhooks is required

For manual onboarding, follow these steps:

1. Add your tenant to `internal/config/tenant.yaml` if it is not included.
  ```yaml
  tenant-name:
    name: Tenant Name
```

2. Add the service to `internal/config/services.yaml`.
  
  ```yaml
  service-name:
    display_name: "Service Name"
    tenant: <tenant>
    gh_repo: <https://github.com/org/repo>
    branch: master # branch to be monitored
    namespace: <namespace of the project>
```

3. Submit an MR to this repo. It will be approved by an owner.

## Development

A Makefile has been provided for most common operations to get the app up and running.
A compose file is also available for standing up the service in podman.

Docker can be substituted for podman if needed.

### Prequisites

    podman
    podman-compose
    Golang >= 1.16

### Launching

    $> make -B build
    $> make run-db
    $> make run-migration
    $> make run-api DEBUG=1

Note: The `DEBUG` argument allows us to send webhooks without needing the secret token.

### Launching with a Mock Database

    $> make -B build
    $> make run-api-mock DEBUG=1

Note: This is useful to avoid having to run the database locally, but this will not persist data between runs.

The API should now be up and available on `localhost:8000`. You should be able to
see the API in action by visiting `http://localhost:8000/api/v1/services`.

### Testing POST Requests to the API Manually

Launch the API as instructed above, then we can send test requests to the API.

The app is designed to take in commit and deployment data through `/api/v1/github` and `/api/v1/tekton` respectively. Using webhooks is also included, but they will not be used to track our platform.

Test json is provided in the `tests` directory in this repo.

To send the requests, you can use curl the following makefile commands: 
- `make test-github`
- `make test-github-webhook`
- `make test-gitlab-webhook`
- `make test-tekton-task`.

From there, you should be able to open a browser and see the results populated at: http://localhost:8000/api/v1/commits. There will be commits matching the webhook data that was sent.

## Running Unit Tests

Aside from the endpoint tests in the Makefile, our unit tests use the Ginkgo testing framework. The service is still in development, so there are not many tests available.

Use `make test` to run all unit tests.

# Get Help

This service is owned by the ConsoldeDot Pipeline team. If you have any questions, or
need support with this service, please contact the team on slack @crc-pipeline-team.

You can also raise an Issue in this repo for them to address.
