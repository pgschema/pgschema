package util

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// SetupSharedEmbeddedPostgres creates a shared embedded PostgreSQL instance for test suites.
// This instance can be reused across multiple test cases to significantly improve test performance.
//
// Usage example:
//
//	func TestMain(m *testing.M) {
//	    // Create shared embedded postgres for all tests
//	    embeddedPG := util.SetupSharedEmbeddedPostgres(nil, embeddedpostgres.PostgresVersion("17.5.0"))
//	    defer embeddedPG.Stop()
//
//	    // Run tests
//	    code := m.Run()
//	    os.Exit(code)
//	}
//
//	func TestMyFeature(t *testing.T) {
//	    config := &plan.PlanConfig{
//	        // ... other config ...
//	        EmbeddedPG: embeddedPG,  // Reuse shared instance
//	    }
//	    plan, err := plan.GeneratePlan(config)
//	    // ...
//	}
func SetupSharedEmbeddedPostgres(t testing.TB, version embeddedpostgres.PostgresVersion) *EmbeddedPostgres {
	config := &EmbeddedPostgresConfig{
		Version:  version,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
	}

	embeddedPG, err := StartEmbeddedPostgres(config)
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to start shared embedded PostgreSQL: %v", err)
		} else {
			panic("Failed to start shared embedded PostgreSQL: " + err.Error())
		}
	}

	return embeddedPG
}
