package containers

import (
	"context"
	"testing"
)

type fakeRunner struct {
	outputs map[string]string
}

func (f *fakeRunner) Run(ctx context.Context, args ...string) (string, error) {
	key := joinArgs(args)
	return f.outputs[key], nil
}

func TestListCapsules(t *testing.T) {
	fr := &fakeRunner{outputs: map[string]string{
		"docker ps -a --filter label=krellin.kind=capsule --format {{.ID}}|{{.Names}}": "id1|krellin-repo1\nid2|krellin-repo2",
		"docker inspect -f {{json .Config.Labels}} id1":                                             "{\"krellin.repo_id\":\"repo1\"}",
		"docker inspect -f {{json .Config.Labels}} id2":                                             "{\"krellin.repo_id\":\"repo2\"}",
		"docker inspect -f {{.SizeRw}} id1":                                                         "100",
		"docker inspect -f {{.SizeRw}} id2":                                                         "200",
	}}
	c := New(fr)
	items, err := c.ListCapsules(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].RepoID != "repo1" || items[1].RepoID != "repo2" {
		t.Fatalf("unexpected repo ids: %+v", items)
	}
	if items[0].SizeBytes != 100 || items[1].SizeBytes != 200 {
		t.Fatalf("unexpected sizes: %+v", items)
	}
}

func TestListImages(t *testing.T) {
	fr := &fakeRunner{outputs: map[string]string{
		"docker images --filter label=krellin.kind --format {{.ID}}|{{.Repository}}|{{.Tag}}": "img1|repo|tag",
		"docker inspect -f {{json .Config.Labels}} img1":                                          "{\"krellin.kind\":\"freeze\"}",
		"docker inspect -f {{.Size}} img1":                                                       "1234",
	}}
	c := New(fr)
	items, err := c.ListImages(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || items[0].Repository != "repo" {
		t.Fatalf("unexpected items: %+v", items)
	}
	if items[0].SizeBytes != 1234 {
		t.Fatalf("unexpected size: %+v", items[0])
	}
}
