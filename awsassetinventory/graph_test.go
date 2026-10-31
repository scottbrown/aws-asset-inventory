package awsassetinventory

import "testing"

func TestServiceName(t *testing.T) {
	tests := map[ResourceType]string{
		"AWS::EC2::Instance":   "EC2",
		"AWS::S3::Bucket":      "S3",
		"AWS::IAM::Role":       "IAM",
		"Custom::Thing::Other": "Thing",
		"Unknown":              "Unknown",
	}
	for rt, want := range tests {
		if got := ServiceName(rt); got != want {
			t.Errorf("ServiceName(%q) = %q, want %q", rt, got, want)
		}
	}
}

func TestBuildGraph(t *testing.T) {
	inv := NewInventory("test", []Region{"us-east-1", "us-west-2"})
	inv.AddResource(Resource{
		ResourceType: "AWS::EC2::Instance", ResourceID: "i-1", Region: "us-east-1",
		Relationships: []Relationship{
			{ResourceType: "AWS::EC2::VPC", ResourceID: "vpc-1", Name: "Is contained in Vpc"},
			{ResourceType: "AWS::EC2::Volume", ResourceID: "vol-1", Name: "Is attached to Volume"},
			{ResourceType: "AWS::IAM::Role", ResourceID: "role-1", Name: "Is associated with Role"},
			{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1", Name: "self"},
			{ResourceType: "AWS::EC2::Subnet", ResourceID: "", Name: "empty"},
		},
	})
	inv.AddResource(Resource{
		ResourceType: "AWS::EC2::VPC", ResourceID: "vpc-1", Region: "us-east-1",
		Relationships: []Relationship{
			{ResourceType: "AWS::EC2::Instance", ResourceID: "i-1", Name: "Contains Instance"},
		},
	})
	// Global resource recorded in another region.
	inv.AddResource(Resource{ResourceType: "AWS::IAM::Role", ResourceID: "role-1", Region: "us-west-2"})
	// Duplicate resource is ignored.
	inv.AddResource(Resource{ResourceType: "AWS::IAM::Role", ResourceID: "role-1", Region: "us-west-2"})

	g := BuildGraph(inv)

	nodes := make(map[string]GraphNode)
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	if len(g.Nodes) != 4 {
		t.Fatalf("BuildGraph() nodes = %d, want 4 (3 resources + 1 stub): %+v", len(g.Nodes), g.Nodes)
	}

	stub, ok := nodes["us-east-1|AWS::EC2::Volume|vol-1"]
	if !ok || !stub.Stub {
		t.Errorf("volume should be a stub node, got %+v", stub)
	}
	if nodes["us-east-1|AWS::EC2::VPC|vpc-1"].Stub {
		t.Error("VPC is in the inventory and should not be a stub")
	}

	if len(g.Links) != 3 {
		t.Fatalf("BuildGraph() links = %d, want 3: %+v", len(g.Links), g.Links)
	}
	links := make(map[string]string)
	for _, l := range g.Links {
		links[l.Source+" -> "+l.Target] = l.Label
	}
	if links["us-east-1|AWS::EC2::Instance|i-1 -> us-east-1|AWS::EC2::VPC|vpc-1"] != "Is contained in Vpc" {
		t.Errorf("missing instance -> VPC link: %+v", links)
	}
	if _, ok := links["us-east-1|AWS::EC2::Instance|i-1 -> us-west-2|AWS::IAM::Role|role-1"]; !ok {
		t.Errorf("role should be found in another region: %+v", links)
	}

	if d := nodes["us-east-1|AWS::EC2::Instance|i-1"].Degree; d != 3 {
		t.Errorf("instance degree = %d, want 3", d)
	}
	if d := nodes["us-east-1|AWS::EC2::VPC|vpc-1"].Degree; d != 1 {
		t.Errorf("VPC degree = %d, want 1 (reverse relationship is not a second link)", d)
	}
}

func TestBuildGraph_Empty(t *testing.T) {
	g := BuildGraph(NewInventory("test", nil))
	if g.Nodes == nil || g.Links == nil {
		t.Error("BuildGraph() should return empty, non-nil slices so the JSON has arrays")
	}
}
