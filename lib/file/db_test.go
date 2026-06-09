package file

import (
	"os"
	"path/filepath"
	"testing"

	"ehang.io/nps/lib/common"
)

func newTestDb(t *testing.T) *DbUtils {
	t.Helper()
	dir := t.TempDir()
	return &DbUtils{JsonDb: &JsonDb{
		TaskFilePath:   filepath.Join(dir, "tasks.json"),
		HostFilePath:   filepath.Join(dir, "hosts.json"),
		ClientFilePath: filepath.Join(dir, "clients.json"),
		GlobalFilePath: filepath.Join(dir, "global.json"),
	}}
}

func TestDelClientDeletesOwnedTunnelsAndHosts(t *testing.T) {
	db := newTestDb(t)
	target := &Client{Id: 10, VerifyKey: "target"}
	other := &Client{Id: 11, VerifyKey: "other"}

	db.JsonDb.Clients.Store(target.Id, target)
	db.JsonDb.Clients.Store(other.Id, other)
	db.JsonDb.Tasks.Store(1, &Tunnel{Id: 1, Client: target})
	db.JsonDb.Tasks.Store(2, &Tunnel{Id: 2, Client: other})
	db.JsonDb.Hosts.Store(3, &Host{Id: 3, Client: target})
	db.JsonDb.Hosts.Store(4, &Host{Id: 4, Client: other})

	if err := db.DelClient(target.Id); err != nil {
		t.Fatalf("DelClient returned error: %v", err)
	}

	if _, ok := db.JsonDb.Clients.Load(target.Id); ok {
		t.Fatal("target client was not deleted")
	}
	if _, ok := db.JsonDb.Tasks.Load(1); ok {
		t.Fatal("target tunnel was not deleted")
	}
	if _, ok := db.JsonDb.Hosts.Load(3); ok {
		t.Fatal("target host was not deleted")
	}
	if _, ok := db.JsonDb.Clients.Load(other.Id); !ok {
		t.Fatal("unrelated client was deleted")
	}
	if _, ok := db.JsonDb.Tasks.Load(2); !ok {
		t.Fatal("unrelated tunnel was deleted")
	}
	if _, ok := db.JsonDb.Hosts.Load(4); !ok {
		t.Fatal("unrelated host was deleted")
	}
}

func TestDelTunnelsAndHostsByClientIdSkipsNilClient(t *testing.T) {
	db := newTestDb(t)
	target := &Client{Id: 10, VerifyKey: "target"}
	db.JsonDb.Tasks.Store(1, &Tunnel{Id: 1, Client: nil})
	db.JsonDb.Tasks.Store(2, &Tunnel{Id: 2, Client: target})
	db.JsonDb.Hosts.Store(3, &Host{Id: 3, Client: nil})
	db.JsonDb.Hosts.Store(4, &Host{Id: 4, Client: target})

	tasks, hosts := db.DelTunnelsAndHostsByClientId(target.Id, false)
	if tasks != 1 || hosts != 1 {
		t.Fatalf("deleted tasks=%d hosts=%d, want 1 and 1", tasks, hosts)
	}
	if _, ok := db.JsonDb.Tasks.Load(1); !ok {
		t.Fatal("nil-client tunnel should be left untouched")
	}
	if _, ok := db.JsonDb.Hosts.Load(3); !ok {
		t.Fatal("nil-client host should be left untouched")
	}
}

func TestLoadTaskAndHostSkipsRecordsWithoutValidClient(t *testing.T) {
	db := newTestDb(t)
	taskData := `{"Id":1}` + "\n" + common.CONN_DATA_SEQ + `{"Id":2,"Client":{"Id":99}}`
	hostData := `{"Id":3}` + "\n" + common.CONN_DATA_SEQ + `{"Id":4,"Client":{"Id":99}}`
	if err := os.WriteFile(db.JsonDb.TaskFilePath, []byte(taskData), 0600); err != nil {
		t.Fatalf("write task data: %v", err)
	}
	if err := os.WriteFile(db.JsonDb.HostFilePath, []byte(hostData), 0600); err != nil {
		t.Fatalf("write host data: %v", err)
	}

	db.JsonDb.LoadTaskFromJsonFile()
	db.JsonDb.LoadHostFromJsonFile()

	if _, ok := db.JsonDb.Tasks.Load(1); ok {
		t.Fatal("task without client should be skipped")
	}
	if _, ok := db.JsonDb.Tasks.Load(2); ok {
		t.Fatal("task with missing client should be skipped")
	}
	if _, ok := db.JsonDb.Hosts.Load(3); ok {
		t.Fatal("host without client should be skipped")
	}
	if _, ok := db.JsonDb.Hosts.Load(4); ok {
		t.Fatal("host with missing client should be skipped")
	}
}
