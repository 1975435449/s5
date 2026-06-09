package server

import (
	"testing"

	"ehang.io/nps/lib/file"
)

func TestAddTaskReturnsErrorWhenListenerCannotBindServerIP(t *testing.T) {
	task := &file.Tunnel{
		Id:       990001,
		Mode:     "tcp",
		Port:     0,
		ServerIp: "192.0.2.1",
		Client: &file.Client{
			Id: 1,
			Cnf: &file.Config{
				Compress: false,
				Crypt:    false,
			},
			Flow: &file.Flow{},
		},
		Flow:   &file.Flow{},
		Target: &file.Target{TargetStr: "127.0.0.1:80"},
	}
	defer RunList.Delete(task.Id)

	if err := AddTask(task); err == nil {
		t.Fatal("expected AddTask to return listener bind error")
	}
	if _, ok := RunList.Load(task.Id); ok {
		t.Fatal("failed task must not remain in RunList")
	}
}
