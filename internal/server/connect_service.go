package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	v1 "github.com/ananthakumaran/paisa/internal/gen/paisa/v1"
	"github.com/ananthakumaran/paisa/internal/gen/paisa/v1/paisav1connect"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/utils"
	"google.golang.org/protobuf/types/known/structpb"
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

func (s *paisaServiceServer) GetConfig(
	_ context.Context,
	_ *connect.Request[v1.GetConfigRequest],
) (*connect.Response[v1.GetConfigResponse], error) {
	requestDB, _ := beginRequestTelemetry(s.db)

	var now *string
	if utils.IsNowDefined() {
		n := utils.Now().Format("2006-01-02T15:04:05Z07:00")
		now = &n
	}

	lastPriceUpdate, _ := metadata.GetOrDefault(requestDB, model.LastPriceSyncKey, "")

	journalPath := config.GetJournalPath()
	files, err := ledger.Cli().Files(journalPath)
	if err != nil {
		files = []string{journalPath}
	}
	currentHash, _ := utils.SHA256Files(files)
	lastHash, _ := metadata.GetOrDefault(requestDB, model.JournalHashKey, "")
	isJournalDirty := currentHash != lastHash

	configStruct, err := toProtoStruct(config.GetConfig())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	schemaStruct, err := toProtoStruct(config.GetSchema())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.GetConfigResponse{
		Config:          configStruct,
		Schema:          schemaStruct,
		Accounts:        accounting.AllAccounts(requestDB),
		LastPriceUpdate: lastPriceUpdate,
		IsJournalDirty:  isJournalDirty,
		Now:             now,
	}), nil
}

func (s *paisaServiceServer) UpdateConfig(
	_ context.Context,
	req *connect.Request[v1.UpdateConfigRequest],
) (*connect.Response[v1.UpdateConfigResponse], error) {
	if req.Msg.GetConfig() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("config is required"))
	}

	configJSON, err := req.Msg.GetConfig().MarshalJSON()
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := config.SaveConfig(configJSON); err != nil {
		connectErr := connect.NewError(connect.CodeInvalidArgument, err)
		connectErr.Meta().Set("x-http-code", "400")
		return nil, connectErr
	}

	return connect.NewResponse(&v1.UpdateConfigResponse{Success: true}), nil
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

func toProtoStruct(v any) (*structpb.Struct, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(payload, &asMap); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return structpb.NewStruct(asMap)
}
