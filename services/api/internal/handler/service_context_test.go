package handler

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"nexusai/services/api/internal/agent"
	"nexusai/services/api/internal/approval"
	"nexusai/services/api/internal/config"
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/svc"

	"entgo.io/ent/dialect"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/pathvar"
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
	return svc.NewServiceContext(&config.Config{}, runner, store, nil)
}

func withPathID(r *http.Request, id string) *http.Request {
	return pathvar.WithVars(r, map[string]string{"id": id})
}

func startTestServer(t *testing.T, store approval.Store) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate test port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release test port: %v", err)
	}

	server, err := rest.NewServer(rest.RestConf{Host: "127.0.0.1", Port: port})
	if err != nil {
		t.Fatalf("create test server: %v", err)
	}
	RegisterHandlers(server, newTestServiceContext(store, nil))
	go server.Start()
	t.Cleanup(server.Stop)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		response, err := http.Get(baseURL + "/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return baseURL
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("test server did not become ready")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
