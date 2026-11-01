package handler

import (
	"strings"
	"testing"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"

	"entgo.io/ent/dialect"
)

func newTestApprovalStore(t *testing.T) *approval.EntStore {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	client, closeData, err := data.NewData(&data.Data{Database: data.Database{
		Driver:       dialect.SQLite,
		Source:       "file:" + name + "?mode=memory&cache=shared&_fk=1&_txlock=immediate",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		AutoMigrate:  true,
	}})
	if err != nil {
		t.Fatalf("create sqlite data source: %v", err)
	}
	t.Cleanup(closeData)
	return approval.NewEntStore(client, dialect.SQLite)
}

func newTestServiceContext(store approval.Store, runner agent.Runner) *svc.ServiceContext {
	if runner == nil {
		runner = agent.MockRunner{}
	}
	return svc.NewServiceContext(config.Config{}, runner, store)
}
