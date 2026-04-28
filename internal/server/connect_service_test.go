package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAccountTree_Empty(t *testing.T) {
	nodes := buildAccountTree(nil)
	assert.Empty(t, nodes)
}

func TestBuildAccountTree_SingleAccount(t *testing.T) {
	nodes := buildAccountTree([]string{"Assets"})
	require.Len(t, nodes, 1)
	assert.Equal(t, "Assets", nodes[0].Name)
	assert.Equal(t, "Assets", nodes[0].FullName)
	assert.True(t, nodes[0].IsLeaf)
	assert.Empty(t, nodes[0].Children)
}

func TestBuildAccountTree_NestedAccounts(t *testing.T) {
	accounts := []string{
		"Assets:Checking:SBI",
		"Assets:Checking:HDFC",
		"Expenses:Food",
	}
	nodes := buildAccountTree(accounts)

	// Top-level: Assets, Expenses
	require.Len(t, nodes, 2)
	assert.Equal(t, "Assets", nodes[0].Name)
	assert.Equal(t, "Expenses", nodes[1].Name)

	// Assets -> Checking
	assets := nodes[0]
	require.Len(t, assets.Children, 1)
	checking := assets.Children[0]
	assert.Equal(t, "Checking", checking.Name)
	assert.Equal(t, "Assets:Checking", checking.FullName)
	assert.False(t, checking.IsLeaf, "intermediate node should not be a leaf")

	// Checking -> SBI, HDFC (sorted)
	require.Len(t, checking.Children, 2)
	assert.Equal(t, "HDFC", checking.Children[0].Name)
	assert.True(t, checking.Children[0].IsLeaf)
	assert.Equal(t, "SBI", checking.Children[1].Name)
	assert.True(t, checking.Children[1].IsLeaf)

	// Expenses -> Food
	expenses := nodes[1]
	require.Len(t, expenses.Children, 1)
	assert.Equal(t, "Food", expenses.Children[0].Name)
	assert.Equal(t, "Expenses:Food", expenses.Children[0].FullName)
	assert.True(t, expenses.Children[0].IsLeaf)
}

func TestBuildAccountTree_IntermediateNodeAlsoLeaf(t *testing.T) {
	// "Assets:Checking" appears both as a leaf and as an intermediate node.
	accounts := []string{
		"Assets:Checking",
		"Assets:Checking:SBI",
	}
	nodes := buildAccountTree(accounts)
	require.Len(t, nodes, 1)

	assets := nodes[0]
	require.Len(t, assets.Children, 1)

	checking := assets.Children[0]
	assert.Equal(t, "Checking", checking.Name)
	assert.True(t, checking.IsLeaf, "Checking is both an intermediate node and a leaf account")
	require.Len(t, checking.Children, 1)
	assert.Equal(t, "SBI", checking.Children[0].Name)
}

func TestBuildAccountTree_SortedAlphabetically(t *testing.T) {
	accounts := []string{
		"Income:Salary",
		"Income:Bonus",
		"Assets:Cash",
	}
	nodes := buildAccountTree(accounts)

	// Top-level sorted: Assets, Income
	require.Len(t, nodes, 2)
	assert.Equal(t, "Assets", nodes[0].Name)
	assert.Equal(t, "Income", nodes[1].Name)

	// Income children sorted: Bonus, Salary
	income := nodes[1]
	require.Len(t, income.Children, 2)
	assert.Equal(t, "Bonus", income.Children[0].Name)
	assert.Equal(t, "Salary", income.Children[1].Name)
}

func TestBuildAccountTree_FullNamePreserved(t *testing.T) {
	accounts := []string{"Assets:Investments:Stocks:AAPL"}
	nodes := buildAccountTree(accounts)

	require.Len(t, nodes, 1)
	assert.Equal(t, "Assets", nodes[0].FullName)

	inv := nodes[0].Children[0]
	assert.Equal(t, "Assets:Investments", inv.FullName)

	stocks := inv.Children[0]
	assert.Equal(t, "Assets:Investments:Stocks", stocks.FullName)

	aapl := stocks.Children[0]
	assert.Equal(t, "Assets:Investments:Stocks:AAPL", aapl.FullName)
	assert.True(t, aapl.IsLeaf)
}
