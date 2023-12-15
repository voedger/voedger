/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */
package appdefcompat

import (
	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/appdef"
)

var constrains = []NodeConstraint{
	{NodeNameTypes, ConstraintInsertOnly},
	{NodeNameFields, ConstraintAppendOnly},
	{NodeNameUniqueFields, ConstraintNonModifiable},
	{NodeNamePartKeyFields, ConstraintNonModifiable},
	{NodeNameClustColsFields, ConstraintNonModifiable},
	{NodeNameCommandArgs, ConstraintNonModifiable},
	{NodeNameCommandResult, ConstraintNonModifiable},
}

func checkBackwardCompatibility(old, new appdef.IAppDef) (cerrs *CompatibilityErrors) {
	return &CompatibilityErrors{
		Errors: compareNodes(buildTree(old), buildTree(new), constrains),
	}
}

func ignoreCompatibilityErrors(cerrs *CompatibilityErrors, pathsToIgnore [][]string) (cerrsOut *CompatibilityErrors) {
	cerrsOut = &CompatibilityErrors{}
	for _, cerr := range cerrs.Errors {
		found := false
		for _, pathToIgnore := range pathsToIgnore {
			if slices.Equal(cerr.OldTreePath, pathToIgnore) {
				found = true
				break
			}
		}
		if !found {
			cerrsOut.Errors = append(cerrsOut.Errors, cerr)
		}
	}
	return
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
		node = buildTypesNode(parentNode, t, false)
	case appdef.IQuery:
		node = buildQueryNode(parentNode, t)
	case appdef.ICommand:
		node = buildCommandNode(parentNode, t)
	case appdef.IDoc:
		node = buildTableNode(parentNode, t)
	// TODO: add buildProjectorNode when proper appdef interface will be ready
	default:
		node = buildQNameNode(parentNode, item.(appdef.IType), item.(appdef.IType).QName().String(), false)
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

func buildQueryNode(parentNode *CompatibilityTreeNode, item appdef.IQuery) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildFieldsNode(node, item.Param(), NodeNameQueryArgs),
		buildFieldsNode(node, item.Result(), NodeNameQueryResult),
	)
	return
}

func buildTableNode(parentNode *CompatibilityTreeNode, item appdef.IDoc) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildUniqueFieldsNode(node, item),
		buildFieldsNode(node, item, NodeNameFields),
		buildContainersNode(node, item),
		buildAbstractNode(node, item),
		// TODO: implement buildInheritsNode(node, item)
	)
	return
}

func buildCommandNode(parentNode *CompatibilityTreeNode, item appdef.ICommand) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildFieldsNode(node, item.Param(), NodeNameCommandArgs),
		buildFieldsNode(node, item.UnloggedParam(), NodeNameUnloggedArgs),
		buildFieldsNode(node, item.Result(), NodeNameCommandResult),
	)
	return
}

func buildQNameNode(parentNode *CompatibilityTreeNode, item appdef.IType, name string, qNameOnly bool) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, name, nil)
	if !qNameOnly {
		if t, ok := item.(appdef.IWithAbstract); ok {
			node.Props = append(node.Props, buildAbstractNode(node, t))
		}
		if t, ok := item.(appdef.IFields); ok {
			node.Props = append(node.Props, buildFieldsNode(node, t, NodeNameFields))
		}
		if t, ok := item.(appdef.IContainers); ok {
			node.Props = append(node.Props, buildContainersNode(node, t))
		}
	}
	return
}

func buildAppDefNode(parentNode *CompatibilityTreeNode, item appdef.IAppDef) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameAppDef, nil)
	node.Props = append(node.Props, buildTypesNode(node, item, false))
	return
}

func buildAbstractNode(parentNode *CompatibilityTreeNode, item appdef.IWithAbstract) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameAbstract, item.Abstract())
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

func buildFieldsNode(parentNode *CompatibilityTreeNode, item interface{}, nodeName string) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, nodeName, nil)
	if item == nil {
		return
	}
	if fieldsObj, ok := item.(appdef.IFields); ok {
		fieldsObj.Fields(func(field appdef.IField) {
			node.Props = append(node.Props, buildFieldNode(node, field))
		})
	}
	return
}

func buildUniqueFieldNode(parentNode *CompatibilityTreeNode, item appdef.IUnique) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.Name(), item.ID())
	fieldsNode := newNode(node, NodeNameFields, nil)
	for _, f := range item.Fields() {
		fieldsNode.Props = append(fieldsNode.Props, buildFieldNode(node, f))
	}
	node.Props = append(node.Props,
		fieldsNode, // Fields node
		buildQNameNode(node, item.ParentStructure(), NodeNameParent, false), // Parent node
	)
	return
}

func buildUniqueFieldsNode(parentNode *CompatibilityTreeNode, item appdef.IUniques) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameUniqueFields, nil)
	item.Uniques(func(field appdef.IUnique) {
		node.Props = append(node.Props, buildUniqueFieldNode(node, field))
	})
	return
}

func buildContainersNode(parentNode *CompatibilityTreeNode, item appdef.IContainers) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameContainers, nil)
	item.Containers(func(container appdef.IContainer) {
		node.Props = append(node.Props, buildContainerNode(node, container))
	})
	return
}

func buildWorkspaceNode(parentNode *CompatibilityTreeNode, item appdef.IWorkspace) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildTypesNode(node, item.(appdef.IWithTypes), true),
		buildDescriptorNode(node, item.Descriptor()),
		buildAbstractNode(node, item.(appdef.IWithAbstract)),
		// TODO: add buildInheritsNode with ancestors in Props
	)
	return
}

func buildTypesNode(parentNode *CompatibilityTreeNode, item appdef.IWithTypes, qNamesOnly bool) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameTypes, nil)
	item.Types(func(t appdef.IType) {
		if qNamesOnly {
			node.Props = append(node.Props, buildQNameNode(node, t, t.QName().String(), true))
		} else {
			node.Props = append(node.Props, buildTreeNode(node, t))
		}
	})
	return
}

func buildDescriptorNode(parentNode *CompatibilityTreeNode, item appdef.QName) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameDescriptor, item.String())
	return
}

func buildViewNode(parentNode *CompatibilityTreeNode, item appdef.IView) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.QName().String(), nil)
	node.Props = append(node.Props,
		buildFieldsNode(node, item.Key().PartKey(), NodeNamePartKeyFields),
		buildFieldsNode(node, item.Key().ClustCols(), NodeNameClustColsFields),
		buildFieldsNode(node, item.Value(), NodeNameFields),
	)
	return
}

func compareNodes(old, new *CompatibilityTreeNode, constrains []NodeConstraint) (cerrs []CompatibilityError) {
	if !cmp.Equal(old.Value, new.Value) {
		cerrs = append(cerrs, newCompatibilityError(ConstraintValueMatch, old.Path(), ErrorTypeValueChanged))
	}
	m := matchNodes(old.Props, new.Props)
	cerrs = append(cerrs, checkConstraint(old.Path(), m, findConstraint(old.Name, constrains))...)
	for _, pair := range m.MatchedNodePairs {
		cerrs = append(cerrs, compareNodes(pair[0], pair[1], constrains)...)
	}
	return
}

func findConstraint(nodeName string, constrains []NodeConstraint) (constraint Constraint) {
	for _, c := range constrains {
		if c.NodeName == nodeName {
			return c.Constraint
		}
	}
	return
}

func checkConstraint(oldTreePath []string, m *matchNodesResult, constraint Constraint) (cerrs []CompatibilityError) {
	if len(constraint) == 0 {
		return
	}
	if len(m.DeletedNodeNames) == 0 && m.InsertedNodeCount > 0 {
		if constraint == ConstraintNonModifiable || constraint == ConstraintAppendOnly {
			errorType := ErrorTypeNodeInserted
			if constraint == ConstraintNonModifiable {
				errorType = ErrorTypeNodeModified
			}
			cerrs = append(cerrs, newCompatibilityError(constraint, oldTreePath, errorType))
		}
	}

	if constraint == ConstraintNonModifiable {
		if m.AppendedNodeCount > 0 {
			cerrs = append(cerrs, newCompatibilityError(constraint, oldTreePath, ErrorTypeNodeModified))
		}
	}

	if len(m.DeletedNodeNames) > 0 {
		if constraint == ConstraintNonModifiable || constraint == ConstraintAppendOnly || constraint == ConstraintInsertOnly {
			errorType := ErrorTypeNodeRemoved
			if constraint == ConstraintNonModifiable {
				errorType = ErrorTypeNodeModified
			}
			for _, deletedQName := range m.DeletedNodeNames {
				path := append(oldTreePath, deletedQName)
				cerrs = append(cerrs, newCompatibilityError(constraint, path, errorType))
			}
		}
	}

	if len(m.ReorderedNodeNames) > 0 && len(m.DeletedNodeNames) == 0 {
		if constraint == ConstraintNonModifiable || constraint == ConstraintAppendOnly {
			errorType := ErrorTypeOrderChanged
			if constraint == ConstraintNonModifiable {
				errorType = ErrorTypeNodeModified
			}
			for _, reorderedNodeName := range m.ReorderedNodeNames {
				newOldPath := make([]string, len(oldTreePath)+1)
				copy(newOldPath, append(oldTreePath, reorderedNodeName))
				cerrs = append(cerrs, newCompatibilityError(constraint, newOldPath, errorType))
			}
		}
	}
	return
}

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
func matchNodes(oldNodes, newNodes []*CompatibilityTreeNode) *matchNodesResult {
	result := &matchNodesResult{
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
			if i > len(oldNodes)-1 {
				result.AppendedNodeCount++
			} else {
				result.InsertedNodeCount++
			}
		}
	}
	return result
}
