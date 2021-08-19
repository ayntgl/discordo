package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

// NewGuildsTreeView creates and returns a new guilds treeview.
func NewGuildsTreeView(onGuildsTreeViewSelected func(*tview.TreeNode)) (treeV *tview.TreeView) {
	treeN := tview.NewTreeNode("")
	treeV = tview.NewTreeView()
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetSelectedFunc(onGuildsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return
}

// NewChannelsTreeView creates and returns a new channels treeview.
func NewChannelsTreeView(onChannelsTreeViewSelected func(*tview.TreeNode)) (treeV *tview.TreeView) {
	treeN := tview.NewTreeNode("")
	treeV = tview.NewTreeView()
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetSelectedFunc(onChannelsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft)

	return
}

// NewTextChannelTreeNode creates and returns a new text channel treenode.
func NewTextChannelTreeNode(c discord.Channel) (n *tview.TreeNode) {
	n = tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
		SetReference(c.ID)

	return
}

// GetTreeNodeByReference gets the TreeNode that has reference r from the given treeview.
func GetTreeNodeByReference(r interface{}, treeV *tview.TreeView) (mn *tview.TreeNode) {
	treeV.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

// CreateTopLevelTreeNodes creates treenodes for the top-level (orphan) channels.
func CreateTopLevelTreeNodes(rootN *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (c.ParentID == 0 || c.ParentID == discord.NullChannelID) {
			cn := NewTextChannelTreeNode(c)
			rootN.AddChild(cn)
			continue
		}
	}
}

// CreateSecondLevelTreeNodes creates treenodes for the second-level (category children) channels.
func CreateSecondLevelTreeNodes(channelsTreeView *tview.TreeView, rootN *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (c.ParentID != 0 && c.ParentID != discord.NullChannelID) {
			if pn := GetTreeNodeByReference(c.ParentID, channelsTreeView); pn != nil {
				cn := NewTextChannelTreeNode(c)
				pn.AddChild(cn)
			}
		}
	}
}
