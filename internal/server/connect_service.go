package server

import (
	"context"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/ananthakumaran/paisa/internal/accounting"
	v1 "github.com/ananthakumaran/paisa/internal/gen/paisa/v1"
	"github.com/ananthakumaran/paisa/internal/gen/paisa/v1/paisav1connect"
	"gorm.io/gorm"
)

// paisaServiceServer implements the Connect-Go PaisaServiceHandler.
type paisaServiceServer struct {
	db *gorm.DB
}

var _ paisav1connect.PaisaServiceHandler = (*paisaServiceServer)(nil)

// GetAccountTree returns the hierarchical account tree for all accounts that
// have at least one posting in the database.
func (s *paisaServiceServer) GetAccountTree(
	_ context.Context,
	_ *connect.Request[v1.GetAccountTreeRequest],
) (*connect.Response[v1.GetAccountTreeResponse], error) {
	accounts := accounting.AllAccounts(s.db)
	roots := buildAccountTree(accounts)
	return connect.NewResponse(&v1.GetAccountTreeResponse{
		Accounts: roots,
	}), nil
}

// entryNode is used during tree construction.
type entryNode struct {
	node     *v1.AccountNode
	children map[string]*entryNode
}

// buildAccountTree converts a flat list of colon-separated account names into a
// tree of AccountNode messages sorted alphabetically at each level.
func buildAccountTree(accounts []string) []*v1.AccountNode {
	leafSet := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		leafSet[a] = true
	}

	roots := map[string]*entryNode{}

	for _, account := range accounts {
		parts := strings.Split(account, ":")
		currentMap := roots
		var fullPath string

		for i, part := range parts {
			if fullPath == "" {
				fullPath = part
			} else {
				fullPath = fullPath + ":" + part
			}

			entry, exists := currentMap[part]
			if !exists {
				node := &v1.AccountNode{
					Name:     part,
					FullName: fullPath,
					IsLeaf:   leafSet[fullPath],
				}
				entry = &entryNode{
					node:     node,
					children: map[string]*entryNode{},
				}
				currentMap[part] = entry
			}

			// A segment that was previously added as an intermediate node may now
			// be the leaf of a shorter account path.
			if i == len(parts)-1 {
				entry.node.IsLeaf = true
			}

			currentMap = entry.children
		}
	}

	return flattenEntries(roots)
}

// flattenEntries recursively converts the internal entry map to a sorted slice
// of AccountNode messages.
func flattenEntries(m map[string]*entryNode) []*v1.AccountNode {
	nodes := make([]*v1.AccountNode, 0, len(m))
	for _, entry := range m {
		entry.node.Children = flattenEntries(entry.children)
		nodes = append(nodes, entry.node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	return nodes
}
