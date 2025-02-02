package endpoints_test

import (
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/redhatinsights/platform-changelog-go/internal/models"

	chi "github.com/go-chi/chi/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhatinsights/platform-changelog-go/internal/config"
	"github.com/redhatinsights/platform-changelog-go/internal/db"
	"github.com/redhatinsights/platform-changelog-go/internal/endpoints"
	"github.com/redhatinsights/platform-changelog-go/internal/logging"
)

var _ = Describe("Handler", func() {

	logging.InitLogger()

	Describe("Github Jenkins Run with empty body", func() {
		It("should return 400", func() {
			// create a mock db connection & endpoint handler

			var cfg config.Config = config.Config{
				DBImpl: "mock",
			}
			dbConnector := db.NewMockDBConnector(&cfg)
			handler := endpoints.NewHandler(dbConnector)

			// create a request
			req, err := http.NewRequest("POST", "/api/v1/github", nil)
			Expect(err).To(BeNil())

			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			router := chi.NewRouter()
			router.Post("/api/v1/github", handler.TektonTaskRun)

			router.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(Equal("json body required"))
		})
	})

	// test the TektonTaskRun function
	DescribeTable("Github Jenkins Run with JSON body", func(expected_status int, message string, data_path string) {

		f, err := os.Open(data_path)
		Expect(err).To(BeNil())

		defer f.Close()

		// create a mock db connection & endpoint handler
		var cfg config.Config = config.Config{
			DBImpl: "mock",
		}
		dbConnector := db.NewMockDBConnector(&cfg)
		handler := endpoints.NewHandler(dbConnector)

		// add rbac and cloud-connector services
		s := CreateService(dbConnector, "rbac", config.Service{
			DisplayName: "rbac",
			Tenant:      "insights",
			GHRepo:      "https://github.com/RedHatInsights/insights-rbac",
			Branch:      "master",
			Namespace:   "rbac-prod",
		})

		Expect(s.Name).To(Equal("rbac"))

		// create a request
		req, err := http.NewRequest("POST", "/api/v1/github", f)
		Expect(err).To(BeNil())

		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		router := chi.NewRouter()
		router.Post("/api/v1/github", handler.Github)

		router.ServeHTTP(rr, req)

		// Expect(rr.Code).To(Equal(expected_status))
		Expect(rr.Body.String()).To(ContainSubstring(message))
	},
		Entry("Valid", http.StatusOK, "Commit info received", "../../tests/jenkins/github_dump.json"),
		Entry("Empty", http.StatusBadRequest, "empty json body provided", "../../tests/empty.json"),
		Entry("Not onboarded", http.StatusBadRequest, "app platform-changelog is not onboarded", "../../tests/jenkins/not_onboarded.json"),
		Entry("Missing timestamp", http.StatusBadRequest, "timestamp is required", "../../tests/jenkins/missing_timestamp.json"),
		Entry("Missing app", http.StatusBadRequest, "app is required", "../../tests/jenkins/missing_app.json"),
		Entry("Missing commits", http.StatusBadRequest, "commits is required", "../../tests/jenkins/missing_commits.json"),
		Entry("Empty commits", http.StatusBadRequest, "commits should not be empty", "../../tests/jenkins/commits_empty.json"),
		Entry("Commit missing timestamp", http.StatusBadRequest, "all commits need a timestamp", "../../tests/jenkins/commit_missing_timestamp.json"),
		Entry("Commit missing ref", http.StatusBadRequest, "all commits need a ref", "../../tests/jenkins/commit_missing_ref.json"),
	)
})

func CreateService(conn db.DBConnector, name string, s config.Service) (service models.Services) {
	service, err := conn.CreateServiceTableEntry(name, s)

	Expect(err).To(BeNil())

	return service
}
