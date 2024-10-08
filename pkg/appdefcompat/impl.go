/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package appdefcompat

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/slices"

	"github.com/voedger/voedger/pkg/appdef"
)

var constrains = []NodeConstraint{
	{NodeNameTypes, ConstraintInsertOnly},
	{NodeNameFields, ConstraintAppendOnly},
	{NodeNameUniqueFields, ConstraintOrderChangeOnly},
	{NodeNamePartKeyFields, ConstraintNonModifiable},
	{NodeNameClustColsFields, ConstraintNonModifiable},
	{NodeNameCommandArgs, ConstraintNonModifiable},
	{NodeNameCommandResult, ConstraintNonModifiable},
	{NodeNamePackages, ConstraintAppendOnly | ConstraintOrderChangeOnly},
}

func checkBackwardCompatibility(oldAppDef, newAppDef appdef.IAppDef) (cerrs *CompatibilityErrors) {
	return &CompatibilityErrors{
		Errors: compareNodes(buildTree(oldAppDef), buildTree(newAppDef), constrains),
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
		buildUniquesNode(node, item),
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
		buildQNameNode(node, item.Param(), NodeNameCommandArgs, true),
		buildQNameNode(node, item.UnloggedParam(), NodeNameUnloggedArgs, true),
		buildQNameNode(node, item.Result(), NodeNameCommandResult, true),
	)
	return
}

func buildQNameNode(parentNode *CompatibilityTreeNode, item appdef.IType, name string, qNameOnly bool) (node *CompatibilityTreeNode) {
	var value interface{}
	if item != nil {
		value = item.QName().String()
	}
	node = newNode(parentNode, name, value)
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
	node.Props = append(node.Props, buildPackagesNode(node, item))
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
		for _, field := range fieldsObj.Fields() {
			node.Props = append(node.Props, buildFieldNode(node, field))
		}
	}
	return
}

func buildUniqueFieldsNode(parentNode *CompatibilityTreeNode, item appdef.IUnique) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameUniqueFields, nil)
	for _, f := range item.Fields() {
		node.Props = append(node.Props, buildFieldNode(node, f))
	}
	return
}

func buildUniqueNode(parentNode *CompatibilityTreeNode, item appdef.IUnique) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, item.Name().String(), nil)
	node.Props = append(node.Props,
		buildUniqueFieldsNode(node, item),
	)
	return
}

func buildUniquesNode(parentNode *CompatibilityTreeNode, item appdef.IUniques) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameUniques, nil)
	for _, unique := range item.Uniques() {
		node.Props = append(node.Props, buildUniqueNode(node, unique))
	}
	return
}

func buildContainersNode(parentNode *CompatibilityTreeNode, item appdef.IContainers) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNameContainers, nil)
	for _, container := range item.Containers() {
		node.Props = append(node.Props, buildContainerNode(node, container))
	}
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
	for t := range item.Types {
		if qNamesOnly {
			node.Props = append(node.Props, buildQNameNode(node, t, t.QName().String(), true))
		} else {
			node.Props = append(node.Props, buildTreeNode(node, t))
		}
	}
	return
}

func buildPackagesNode(parentNode *CompatibilityTreeNode, item appdef.IAppDef) (node *CompatibilityTreeNode) {
	node = newNode(parentNode, NodeNamePackages, nil)
	for localName, fullPath := range item.Packages {
		node.Props = append(node.Props, newNode(node, fullPath, localName))
	}
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

func compareNodes(oldNode, newNode *CompatibilityTreeNode, constrains []NodeConstraint) (cerrs []CompatibilityError) {
	if !cmp.Equal(oldNode.Value, newNode.Value) {
		cerrs = append(cerrs, newCompatibilityError(ConstraintValueMatch, oldNode.Path(), ErrorTypeValueChanged))
	}
	m := matchNodes(oldNode.Props, newNode.Props)
	cerrs = append(cerrs, checkConstraint(oldNode.Path(), m, findConstraint(oldNode.Name, constrains))...)
	for _, pair := range m.MatchedNodePairs {
		cerrs = append(cerrs, compareNodes(pair[0], pair[1], constrains)...)
	}
	return
}

func findConstraint(nodeName string, constrains []NodeConstraint) (constraint Constraint) {
	constraint = ConstraintAllAllowed
	for _, c := range constrains {
		if c.NodeName == nodeName {
			return c.Constraint
		}
	}
	return
}

func checkConstraint(oldTreePath []string, m *matchNodesResult, constraint Constraint) (cerrs []CompatibilityError) {
	if constraint == ConstraintAllAllowed {
		return
	}
	if len(m.DeletedNodeNames) == 0 && m.InsertedNodeCount > 0 {
		if constraint == ConstraintNonModifiable || constraint&ConstraintAppendOnly > 0 {
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
		if constraint == ConstraintNonModifiable || constraint&ConstraintAppendOnly > 0 || constraint&ConstraintInsertOnly > 0 {
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

	if constraint&ConstraintOrderChangeOnly == 0 {
		if len(m.ReorderedNodeNames) > 0 && len(m.DeletedNodeNames) == 0 {
			if constraint == ConstraintNonModifiable || constraint&ConstraintAppendOnly > 0 {
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
	}

	if constraint&ConstraintOrderChangeOnly > 0 {
		if m.AppendedNodeCount > 0 || len(m.DeletedNodeNames) > 0 || m.InsertedNodeCount > 0 {
			cerrs = append(cerrs, newCompatibilityError(constraint, oldTreePath, ErrorTypeNodeModified))
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

func (r *PathReader) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(r.rootPath, name))
}

func (r *PathReader) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(filepath.Join(r.rootPath, name))
}

func (r *PathReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.rootPath, name))
}
