// Package csv provides a builder for csv data
package csv

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	headerConjunction      = "."
	DefaultFieldSeparator  = ","
	DefaultRecordSeparator = "\n"
	FieldSeparatorFlag     = "field-separator"
	RecordSeparatorFlag    = "record-separator"
)

type Node struct {
	root          *Node
	parent        *Node
	headLocal     string
	headFull      string
	tail          interface{}
	children      []*Node
	headToNodeMap map[string]*Node
}

// NewCSVNode returns instance of preconfigured CSV Node.
// It contains the information needed to generate a header for data in corresponding CSV field.
func NewCSVNode(
	root *Node,
	parent *Node,
	headLocal string,
	headFull string,
	tail interface{},
) Node {
	return Node{
		root:      root,
		parent:    parent,
		headLocal: headLocal,
		headFull:  headFull,
		tail:      tail,
	}
}

// NewCSVRoot returns instance of preconfigured CSV Node.
// The returned node is a root of the graph.
func NewCSVRoot(
	tail interface{},
) Node {
	root := Node{
		tail:          tail,
		headToNodeMap: make(map[string]*Node),
	}
	root.root = &root
	return root
}

// Marshal marshals the object into JSON then converts JSON to CSV then returns the CSV.
func Marshal(input interface{}, fieldSeparator, recordSeparator string) ([]byte, error) {
	type jsonRawOutput = map[string]interface{}
	var outputItemsRaw []jsonRawOutput

	jsonRawInput, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	// Check if json is array, if not convert to array because unmarshall requires array
	if jsonRawInput[0] != '[' {
		jsonRawInput = append([]byte{'['}, jsonRawInput...)
		jsonRawInput = append(jsonRawInput, ']')
	}

	if err = json.Unmarshal(jsonRawInput, &outputItemsRaw); err != nil {
		return nil, err
	}

	nodes := make([]*Node, len(outputItemsRaw))

	for i, tailRaw := range outputItemsRaw {
		newNode := NewCSVRoot(tailRaw)
		nodes[i] = &newNode
	}

	for _, node := range nodes {
		if err = node.ExpandTail(); err != nil {
			return nil, err
		}
	}

	var leaves []*Node
	for _, node := range nodes {
		nodeLeaves := node.GetLeaves()
		leaves = append(leaves, nodeLeaves...)
	}

	var output strings.Builder
	headers, err := getUniqueHeaders(leaves)
	if err != nil {
		return nil, err
	}
	output.WriteString(strings.Join(headers, fieldSeparator))
	output.WriteString(recordSeparator)
	for _, node := range nodes {
		recordTmp := ""
		for index, header := range headers {
			if index > 0 {
				recordTmp += fieldSeparator
			}
			if node.headToNodeMap[header] != nil && node.headToNodeMap[header].tail != nil {
				recordTmp += fmt.Sprintf("%v", node.headToNodeMap[header].tail)
			}
		}
		output.WriteString(recordTmp)
		output.WriteString(recordSeparator)
	}

	return []byte(output.String()), nil
}

// ExpandTail retrieves the values of all tails in the graph and creates the whole graph structure.
func (node *Node) ExpandTail() error {
	switch nodeTail := node.tail.(type) {
	case string:
		escapedCsvInjectionSigns := escapeCSVInjectionSigns(nodeTail)
		escapedDoubleQuotes := strings.ReplaceAll(escapedCsvInjectionSigns, `"`, `""`)
		enclosedField := fmt.Sprintf("%s%s%s", `"`, escapedDoubleQuotes, `"`)
		node.tail = enclosedField
	case float64, bool:
		node.tail = nodeTail
	case []interface{}:
		for childHeadInt, childTail := range nodeTail {
			childHeadStr := strconv.Itoa(childHeadInt)
			newHeadFull := generateFullHeader(node.headFull, headerConjunction, childHeadStr)
			newNode := NewCSVNode(node.root, node, childHeadStr, newHeadFull, childTail)
			node.children = append(node.children, &newNode)
			if err := newNode.ExpandTail(); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for childHead, childTail := range nodeTail {
			newHeadFull := generateFullHeader(node.headFull, headerConjunction, childHead)
			newNode := NewCSVNode(node.root, node, childHead, newHeadFull, childTail)
			node.children = append(node.children, &newNode)
			if err := newNode.ExpandTail(); err != nil {
				return err
			}
		}
	case nil:
		node.tail = nil
	default:
		return fmt.Errorf("error expanding the tail of csv node - type mismatch: %v", node.tail)
	}
	return nil
}

// GetLeaves returns all leaves from the sub-tree starting from the specific node.
func (node *Node) GetLeaves() []*Node {
	var leaves []*Node
	if len(node.children) == 0 {
		node.root.headToNodeMap[node.headFull] = node
		return []*Node{node}
	}
	for _, childNode := range node.children {
		childLeaves := childNode.GetLeaves()
		leaves = append(leaves, childLeaves...)
	}
	return leaves
}

// GetHeaders returns an array of headers of all child nodes.
func (node *Node) GetHeaders() ([]string, error) {
	var headersSet []string
	if len(node.children) == 0 {
		return []string{node.headFull}, nil
	}
	for _, childNode := range node.children {
		childHeaders, err := childNode.GetHeaders()
		if err != nil {
			return nil, err
		}
		headersSet = append(headersSet, childHeaders...)
	}

	return headersSet, nil
}

func getUniqueHeaders(nodes []*Node) ([]string, error) {
	var headersList []string
	for _, node := range nodes {
		nodeHeaders, err := node.GetHeaders()
		if err != nil {
			return nil, err
		}
		headersList = append(headersList, nodeHeaders...)
	}
	headersSet := removeDuplicates(headersList)
	return headersSet, nil
}

func removeDuplicates(values []string) []string {
	keys := make(map[string]struct{})
	var uniqueValues []string
	for _, value := range values {
		if _, ok := keys[value]; !ok {
			keys[value] = struct{}{}
			uniqueValues = append(uniqueValues, value)
		}
	}
	sort.Strings(uniqueValues)
	return uniqueValues
}

func generateFullHeader(prefix, conjunction, suffix string) string {
	if prefix != "" {
		return fmt.Sprintf("%s%s%s", prefix, conjunction, suffix)
	}
	return suffix
}

func escapeCSVInjectionSigns(input string) string {
	if strings.HasPrefix(input, "@") ||
		strings.HasPrefix(input, "=") ||
		strings.HasPrefix(input, "+") ||
		strings.HasPrefix(input, "-") ||
		strings.HasPrefix(input, "\x09") ||
		strings.HasPrefix(input, "\x0D") {
		return fmt.Sprintf("'%s", input)
	}
	return input
}
