package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"strings"

	"github.com/redhatinsights/platform-changelog-go/internal/config"
	"github.com/redhatinsights/platform-changelog-go/internal/db"
	l "github.com/redhatinsights/platform-changelog-go/internal/logging"
	"github.com/redhatinsights/platform-changelog-go/internal/metrics"
	"github.com/redhatinsights/platform-changelog-go/internal/models"
)

type TektonPayload *struct {
	Status      string     `json:"status"`
	Timestamp   *time.Time `json:"timestamp"`
	App         string     `json:"app"`
	Env         string     `json:"env"`
	TriggeredBy string     `json:"triggered_by"`
	Ref         string     `json:"ref,omitempty"`
}

func decodeTektonJSONBody(w http.ResponseWriter, r *http.Request) (TektonPayload, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return nil, fmt.Errorf("invalid Content-Type header: '%s' should contain 'application/json'", r.Header.Get("Content-Type"))
	}

	if r.Body == nil {
		return nil, fmt.Errorf("json body required")
	}

	var payload TektonPayload

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

func (eh *EndpointHandler) TektonTaskRun(w http.ResponseWriter, r *http.Request) {
	metrics.IncTekton(r.Method, r.UserAgent(), false)

	// log everything for now
	l.Log.Info("Tekton TaskRun received")
	l.Log.Info(r.Body)

	payload, err := decodeTektonJSONBody(w, r)
	if err != nil {
		l.Log.Error(err)
		metrics.IncTekton(r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	defer r.Body.Close()

	err = validateTektonPayload(payload)

	if err != nil {
		l.Log.Error(err)
		metrics.IncTekton(r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	deploy, err := convertTektonPayloadToTimeline(eh.conn, payload)

	if err != nil {
		l.Log.Error(err)
		metrics.IncTekton(r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = eh.conn.CreateDeployEntry(deploy)

	if err != nil {
		l.Log.Error(err)
		metrics.IncTekton(r.Method, r.UserAgent(), true)
		writeResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeResponse(w, http.StatusOK, `{"msg": "Tekton info received"}`)
}

// Validate the payload contains necessary data
// Timestamp, App, Status
func validateTektonPayload(payload TektonPayload) error {
	if payload.Timestamp == nil {
		return fmt.Errorf("timestamp is required")
	}

	if payload.App == "" {
		return fmt.Errorf("app is required")
	}

	if payload.Status == "" {
		return fmt.Errorf("status is required")
	}

	return nil
}

// Converting from TektonPayload struct to Timeline model
func convertTektonPayloadToTimeline(conn db.DBConnector, payload TektonPayload) (models.Timelines, error) {
	services := config.Get().Services

	var deploy models.Timelines
	name := payload.App
	// Validate that the app specified is onboarded
	if services[name] == (config.Service{}) {
		return deploy, fmt.Errorf("app %s is not onboarded", name)
	}

	s, _, err := conn.GetServiceByName(name)

	if err != nil {
		return deploy, err
	}

	deploy = models.Timelines{
		ServiceID:       s.ID,
		Timestamp:       *payload.Timestamp,
		Type:            "deploy",
		Repo:            s.Name,
		Ref:             payload.Ref,
		DeployNamespace: payload.App,
		Cluster:         payload.Env,
		TriggeredBy:     payload.TriggeredBy,
		Status:          payload.Status,
	}

	return deploy, nil
}
