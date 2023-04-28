package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redhatinsights/platform-changelog-go/internal/config"
	"github.com/redhatinsights/platform-changelog-go/internal/db"
	l "github.com/redhatinsights/platform-changelog-go/internal/logging"
	"github.com/redhatinsights/platform-changelog-go/internal/metrics"
	"github.com/redhatinsights/platform-changelog-go/internal/models"
)

// This endpoint is different than the github and gitlab endpoints
// This will be used as a part of the Jenkins pipeline
// on each push to a monitored branch (configured in app-interface)

type GithubPayload *struct {
	Timestamp *time.Time     `json:"timestamp"`
	App       string         `json:"app"`
	Repo      string         `json:"repo,omitempty"`
	MergedBy  string         `json:"merged_by,omitempty"`
	Commits   []GithubCommit `json:"commits"`
}

type GithubCommit struct {
	Timestamp *time.Time `json:"timestamp"`
	Ref       string     `json:"ref"`
	Author    string     `json:"author,omitempty"`
	Message   string     `json:"message,omitempty"`
}

func decodeGithubJSONBody(w http.ResponseWriter, r *http.Request) (GithubPayload, error) {
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("invalid Content-Type")
	}

	if r.Body == nil {
		return nil, fmt.Errorf("json body required")
	}

	var payload GithubPayload

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if !dec.More() {
		return nil, fmt.Errorf("empty json body provided")
	}

	err := dec.Decode(&payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (eh *EndpointHandler) GithubJenkins(w http.ResponseWriter, r *http.Request) {
	metrics.IncJenkins("github", r.Method, r.UserAgent(), false)

	payload, err := decodeGithubJSONBody(w, r)
	if err != nil {
		l.Log.Error(err)
		metrics.IncJenkins("github", r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	defer r.Body.Close()

	err = validateGithubPayload(payload)

	if err != nil {
		l.Log.Error(err)
		metrics.IncJenkins("github", r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	commits, err := convertGithubPayloadToTimelines(eh.conn, payload)

	if err != nil {
		l.Log.Error(err)
		metrics.IncJenkins("github", r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = eh.conn.CreateCommitEntry(commits)

	if err != nil {
		l.Log.Error(err)
		metrics.IncJenkins("github", r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeResponse(w, http.StatusOK, `{"msg": "Tekton info received"}`)
}

// Validate the payload contains necessary data
func validateGithubPayload(payload GithubPayload) error {
	if payload.Timestamp == nil {
		return fmt.Errorf("timestamp is required")
	}

	if payload.App == "" {
		return fmt.Errorf("app is required")
	}

	if payload.Commits == nil {
		return fmt.Errorf("commits is required")
	}

	if len(payload.Commits) == 0 {
		return fmt.Errorf("commits should not be empty")
	}

	for _, commit := range payload.Commits {
		if commit.Timestamp == nil {
			return fmt.Errorf("all commits need a timestamp")
		}

		if commit.Ref == "" {
			return fmt.Errorf("all commits need a ref")
		}
	}

	return nil
}

// Converting from TektonPayload struct to Timeline model
func convertGithubPayloadToTimelines(conn db.DBConnector, payload GithubPayload) ([]models.Timelines, error) {
	services := config.Get().Services

	var commits []models.Timelines
	// Validate that the app specified is onboarded
	for key, service := range services {
		if service.Namespace == payload.App {
			s, _, err := conn.GetServiceByName(key)

			if err != nil {
				return commits, err
			}

			for _, commit := range payload.Commits {
				commits = append(commits, models.Timelines{
					ServiceID: s.ID,
					Timestamp: *commit.Timestamp,
					Type:      "commit",
					Repo:      s.Name,
					Ref:       commit.Ref,
					Author:    commit.Author,
					MergedBy:  payload.MergedBy,
					Message:   commit.Message,
				})
			}

			return commits, nil
		}
	}

	return commits, fmt.Errorf("app %s not onboarded", payload.App)
}
