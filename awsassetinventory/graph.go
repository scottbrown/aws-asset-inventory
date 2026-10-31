package awsassetinventory

import "strings"

// GraphNode is a resource in the relationship graph. A stub node is a resource that
// another resource refers to but that is not in the inventory.
type GraphNode struct {
	ID         string       `json:"id"`
	Type       ResourceType `json:"type"`
	Service    string       `json:"service"`
	ResourceID string       `json:"resourceId"`
	Name       string       `json:"name,omitempty"`
	Region     Region       `json:"region,omitempty"`
	AccountID  string       `json:"accountId,omitempty"`
	Degree     int          `json:"degree"`
	Stub       bool         `json:"stub,omitempty"`
}

// GraphLink connects two nodes in the relationship graph.
type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
}

// Graph is the inventory as nodes and links.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Links []GraphLink `json:"links"`
}

// ServiceName returns the service part of a resource type (EC2 for AWS::EC2::Instance).
func ServiceName(rt ResourceType) string {
	parts := strings.Split(string(rt), "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return string(rt)
}

// NodeID returns the graph node ID of a resource.
func NodeID(r Resource) string {
	return nodeID(r.Region, r.ResourceType, r.ResourceID)
}

func nodeID(region Region, rt ResourceType, id string) string {
	return string(region) + "|" + string(rt) + "|" + id
}

// BuildGraph converts the inventory into a graph. Relationships are looked up in the
// region of the source resource first, then in any region (for global resources).
// AWS Config often records a relationship in both directions, so each pair of
// resources gets at most one link.
func BuildGraph(inv *Inventory) *Graph {
	g := &Graph{
		Nodes: make([]GraphNode, 0, len(inv.Resources)),
		Links: make([]GraphLink, 0),
	}
	index := make(map[string]int)        // node ID -> position in g.Nodes
	anyRegion := make(map[string]string) // resourceKey -> node ID of the first match

	for _, r := range inv.Resources {
		id := NodeID(r)
		if _, dup := index[id]; dup {
			continue
		}
		index[id] = len(g.Nodes)
		g.Nodes = append(g.Nodes, GraphNode{
			ID:         id,
			Type:       r.ResourceType,
			Service:    ServiceName(r.ResourceType),
			ResourceID: r.ResourceID,
			Name:       r.ResourceName,
			Region:     r.Region,
			AccountID:  r.AccountID,
		})
		key := resourceKey(r.ResourceType, r.ResourceID)
		if _, ok := anyRegion[key]; !ok {
			anyRegion[key] = id
		}
	}

	seen := make(map[string]bool)
	for _, r := range inv.Resources {
		src := NodeID(r)
		for _, rel := range r.Relationships {
			if rel.ResourceID == "" {
				continue
			}

			dst := nodeID(r.Region, rel.ResourceType, rel.ResourceID)
			if _, ok := index[dst]; !ok {
				if other, ok := anyRegion[resourceKey(rel.ResourceType, rel.ResourceID)]; ok {
					dst = other
				} else {
					index[dst] = len(g.Nodes)
					g.Nodes = append(g.Nodes, GraphNode{
						ID:         dst,
						Type:       rel.ResourceType,
						Service:    ServiceName(rel.ResourceType),
						ResourceID: rel.ResourceID,
						Name:       rel.ResourceName,
						Region:     r.Region,
						Stub:       true,
					})
				}
			}
			if dst == src {
				continue
			}

			pair := src + "\x00" + dst
			if dst < src {
				pair = dst + "\x00" + src
			}
			if seen[pair] {
				continue
			}
			seen[pair] = true

			g.Links = append(g.Links, GraphLink{Source: src, Target: dst, Label: rel.Name})
			g.Nodes[index[src]].Degree++
			g.Nodes[index[dst]].Degree++
		}
	}

	return g
}
