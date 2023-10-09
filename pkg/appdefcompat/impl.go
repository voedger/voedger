/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"

	"github.com/voedger/voedger/pkg/appdef"
)

var constrains = []NodeConstraint{
	{"Types", ConstraintInsertOnly},
	{"Fields", ConstraintAppendOnly},
}

func checkBackwardCompatibility(old, new appdef.IAppDef) CompatibilityErrors {
	return CompatibilityErrors{
		Errors: compareNodes(buildTree(old), buildTree(new), constrains),
	}
}

func buildTree(app appdef.IAppDef) (parentNode *CompatibilityTreeNode) {
	return buildTreeNode(nil, app)
}

func buildTreeNode(parentNode *CompatibilityTreeNode, item interface{}) (node *CompatibilityTreeNode) {
	switch t := item.(type) {
	case appdef.IAppDef:
		node = buildAppDefNode(parentNode, t)
	case appdef.IWorkspace:
		node = buildWorkspaceNode(parentNode, t)
	case appdef.IView:
		node = buildViewNode(parentNode, t)
	case appdef.IWithTypes:
		node = buildTypesNode(parentNode, t)
	// TODO: add buildProjectorNode when proper appdef interface will be ready
	default:
		node = buildQNameNode(parentNode, item.(appdef.IType))
	}
	return
}

func newNode(parentNode *CompatibilityTreeNode, name string, value interface{}) (node *CompatibilityTreeNode) {
	node = new(CompatibilityTreeNode)
	node.ParentNode = parentNode
	node.Name = name
	node.Value = value
	node.Props = make([]*CompatibilityTreeNode, 0)
	return
}

func buildQNameNode(parentNode *CompatibilityTreeNode, item appdef.IType) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	if t, ok := item.(appdef.IWithAbstract); ok {
		node.Props = append(node.Props, buildAbstractNode(node, t))
	}
	if t, ok := item.(appdef.IFields); ok {
		node.Props = append(node.Props, buildFieldsNode(node, t))
	}
	if t, ok := item.(appdef.IContainers); ok {
		node.Props = append(node.Props, buildContainersNode(node, t))
	}
	return
}

func buildAppDefNode(parentNode *CompatibilityTreeNode, item appdef.IAppDef) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "AppDef", nil)
	node.Props = append(node.Props, buildTypesNode(node, item))
	return
}

func buildAbstractNode(parentNode *CompatibilityTreeNode, item appdef.IWithAbstract) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "Abstract", item.Abstract())
	return
}

func buildContainerNode(parentNode *CompatibilityTreeNode, item appdef.IContainer) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.Name(), item.QName().String())
	return
}

func buildFieldNode(parentNode *CompatibilityTreeNode, item appdef.IField) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.Name(), item.DataKind())
	return
}

func buildFieldsNode(parentNode *CompatibilityTreeNode, item appdef.IFields) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "Fields", nil)
	item.Fields(func(field appdef.IField) {
		node.Props = append(node.Props, buildFieldNode(node, field))
	})
	return
}

func buildContainersNode(parentNode *CompatibilityTreeNode, item appdef.IContainers) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "Containers", nil)
	item.Containers(func(container appdef.IContainer) {
		node.Props = append(node.Props, buildContainerNode(node, container))
	})
	return
}

func buildWorkspaceNode(parentNode *CompatibilityTreeNode, item appdef.IWorkspace) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildTypesNode(node, item.(appdef.IWithTypes)),
		buildDescriptorNode(node, item.Descriptor()),
		// TODO: add buildInheritanceNode with ancestors in Props
	)
	return
}

func buildTypesNode(parentNode *CompatibilityTreeNode, item appdef.IWithTypes) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "Types", nil)
	item.Types(func(t appdef.IType) {
		node.Props = append(node.Props, buildTreeNode(node, t))
	})
	return
}

func buildDescriptorNode(parentNode *CompatibilityTreeNode, item appdef.QName) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, "Descriptor", item.String())
	return
}

func buildPartKeyFieldsNode(parentNode *CompatibilityTreeNode, item appdef.IViewPartKey) (node *CompatibilityTreeNode) {
	node = buildFieldsNode(parentNode, item.(appdef.IFields))
	node.Name = "PartKeyFields"
	return
}

func buildClustColsFieldsNode(parentNode *CompatibilityTreeNode, item appdef.IViewClustCols) (node *CompatibilityTreeNode) {
	node = buildFieldsNode(parentNode, item.(appdef.IFields))
	node.Name = "ClustColsFields"
	return
}

func buildViewNode(parentNode *CompatibilityTreeNode, item appdef.IView) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildPartKeyFieldsNode(node, item.Key().PartKey()),
		buildClustColsFieldsNode(node, item.Key().ClustCols()),
		buildFieldsNode(node, item.(appdef.IFields)),
	)
	return
}

func compareNodes(old, new *CompatibilityTreeNode, constrains []NodeConstraint) (cerrs []CompatibilityError) {
	if !cmp.Equal(old.Value, new.Value) {
		cerrs = append(cerrs, newCompatibilityError("", old.Path(), fmt.Sprintf(validationErrorFmt, errMsgMismatch, strings.Join(old.Path(), pathDelimiter))))
	}

	m := matchNodes(old.Props, new.Props)
	for _, constraint := range constrains {
		if constraint.NodeName == old.Name {
			cerrs = append(cerrs, checkConstraint(old.Path(), m, constraint.Constraint)...)
			break
		}
	}
	for _, pair := range m.MatchedNodePairs {
		cerrs = append(cerrs, compareNodes(pair[0], pair[1], constrains)...)
	}
	return
}

func checkConstraint(oldTreePath []string, m matchNodesResult, constraint Constraint) (cerrs []CompatibilityError) {
	if m.InsertedNodeCount > 0 {
		cerrs = append(cerrs, newCompatibilityError(constraint, oldTreePath, fmt.Sprintf(validationErrorFmt, errMsgNodeInserted, strings.Join(oldTreePath, pathDelimiter))))
	}

	if constraint == ConstraintNonModifiable || constraint == ConstraintAppendOnly || constraint == ConstraintInsertOnly {
		for _, deletedQName := range m.DeletedNodeNames {
			path := append(oldTreePath, deletedQName)
			cerrs = append(cerrs, newCompatibilityError(constraint, path, fmt.Sprintf(validationErrorFmt, errMsgNodeRemoved, strings.Join(path, pathDelimiter))))
		}
	}

	if len(m.ReorderedNodeNames) > 0 && len(m.DeletedNodeNames) == 0 {
		if constraint == ConstraintNonModifiable || constraint == ConstraintAppendOnly {
			cerrs = append(cerrs, newCompatibilityError(constraint, oldTreePath, fmt.Sprintf(validationErrorFmt, errMsgOrderChanged, strings.Join(oldTreePath, pathDelimiter))))
		}
	}
	return
}

// Helper function to find a node by name in a slice of nodes
func findNodeByName(nodes []*CompatibilityTreeNode, name string) (foundNode *CompatibilityTreeNode, index int) {
	index = -1
	for i, node := range nodes {
		if node.Name == name {
			index = i
			foundNode = node
		}
	}
	return
}

// matchNodes matches nodes in two CompatibilityTreeNode slices and categorizes them.
func matchNodes(oldNodes, newNodes []*CompatibilityTreeNode) matchNodesResult {
	result := matchNodesResult{
		InsertedNodeCount:  0,
		DeletedNodeNames:   []string{},
		AppendedNodeCount:  0,
		MatchedNodePairs:   [][2]*CompatibilityTreeNode{},
		ReorderedNodeNames: []string{},
	}

	// Compare old nodes with new nodes
	for i, oldNode := range oldNodes {
		newNode, index := findNodeByName(newNodes, oldNode.Name)

		if newNode == nil {
			result.DeletedNodeNames = append(result.DeletedNodeNames, oldNode.Name)
		} else {
			if i != index {
				result.ReorderedNodeNames = append(result.ReorderedNodeNames, newNode.Name)
			}
			result.MatchedNodePairs = append(result.MatchedNodePairs, [2]*CompatibilityTreeNode{oldNode, newNode})
		}
	}

	// Check for appended new nodes
	for i, newNode := range newNodes {
		oldNode, _ := findNodeByName(oldNodes, newNode.Name)
		if oldNode == nil {
			if i >= len(oldNodes) {
				result.AppendedNodeCount++
			} else {
				result.InsertedNodeCount++
			}
		}
	}

	return result
}
