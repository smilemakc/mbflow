package builder

import (
	"fmt"
)

// TransformType sets the transformation type.
// Valid types: passthrough, expression, jq, template
func TransformType(ttype string) NodeOption {
	return func(nb *NodeBuilder) error {
		validTypes := map[string]bool{
			"passthrough": true,
			"expression":  true,
			"jq":          true,
			"template":    true,
		}
		if !validTypes[ttype] {
			return fmt.Errorf("invalid transform type: %s (valid: passthrough, expression, jq, template)", ttype)
		}
		nb.config["type"] = ttype
		return nil
	}
}

// TransformExpression sets the expression for expression-type transforms.
// Uses expr-lang syntax.
func TransformExpression(expr string) NodeOption {
	return func(nb *NodeBuilder) error {
		if expr == "" {
			return fmt.Errorf("expression cannot be empty")
		}
		nb.config["expression"] = expr
		return nil
	}
}

// TransformJQ sets the JQ filter for jq-type transforms.
func TransformJQ(filter string) NodeOption {
	return func(nb *NodeBuilder) error {
		if filter == "" {
			return fmt.Errorf("JQ filter cannot be empty")
		}
		nb.config["filter"] = filter
		return nil
	}
}

// TransformTemplate sets the template for template-type transforms.
func TransformTemplate(tmpl string) NodeOption {
	return func(nb *NodeBuilder) error {
		if tmpl == "" {
			return fmt.Errorf("template cannot be empty")
		}
		nb.config["template"] = tmpl
		return nil
	}
}

// TransformMapping sets field mappings for transform operations.
func TransformMapping(mapping map[string]string) NodeOption {
	return func(nb *NodeBuilder) error {
		nb.config["mapping"] = mapping
		return nil
	}
}

// NewPassthroughNode creates a new passthrough transform node.
// Passthrough nodes simply pass the input to output without modification.
func NewPassthroughNode(id, name string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{TransformType("passthrough")}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "transform", name, allOpts...)
}

// NewExpressionNode creates a new expression transform node.
// Uses expr-lang for transformations.
func NewExpressionNode(id, name, expr string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		TransformType("expression"),
		TransformExpression(expr),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "transform", name, allOpts...)
}

// NewJQNode creates a new JQ transform node.
// Uses JQ syntax for JSON transformations.
func NewJQNode(id, name, filter string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		TransformType("jq"),
		TransformJQ(filter),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "transform", name, allOpts...)
}

// NewTemplateNode creates a new template transform node.
// Uses template syntax for transformations.
func NewTemplateNode(id, name, tmpl string, opts ...NodeOption) *NodeBuilder {
	allOpts := []NodeOption{
		TransformType("template"),
		TransformTemplate(tmpl),
	}
	allOpts = append(allOpts, opts...)
	return NewNode(id, "transform", name, allOpts...)
}

// NewTransformNode creates a new generic transform node.
// You must specify the type using TransformType option.
func NewTransformNode(id, name string, opts ...NodeOption) *NodeBuilder {
	return NewNode(id, "transform", name, opts...)
}
