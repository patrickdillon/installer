package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform-plugin-sdk/tfdiags"
)

// EvalGraphBuilder implements GraphBuilder and constructs a graph suitable
// for evaluating in-memory values (input variables, local values, output
// values) in the state without any other side-effects.
//
// This graph is used only in weird cases, such as the "terraform console"
// CLI command, where we need to evaluate expressions against the state
// without taking any other actions.
//
// The generated graph will include nodes for providers, resources, etc
// just to allow indirect dependencies to be resolved, but these nodes will
// not take any actions themselves since we assume that their parts of the
// state, if any, are already complete.
//
// Although the providers are never configured, they must still be available
// in order to obtain schema information used for type checking, etc.
type EvalGraphBuilder struct {
	// Config is the configuration tree.
	Config *configs.Config

	// State is the current state
	State *states.State

	// Components is a factory for the plug-in components (providers and
	// provisioners) available for use.
	Components contextComponentFactory

	// Schemas is the repository of schemas we will draw from to analyse
	// the configuration.
	Schemas *Schemas
}

// See GraphBuilder
func (b *EvalGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	return (&BasicGraphBuilder{
		Steps:    b.Steps(),
		Validate: true,
		Name:     "EvalGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *EvalGraphBuilder) Steps() []GraphTransformer {
	concreteProvider := func(a *NodeAbstractProvider) dag.Vertex {
		return &NodeEvalableProvider{
			NodeAbstractProvider: a,
		}
	}

	steps := []GraphTransformer{
		// Creates all the data resources that aren't in the state. This will also
		// add any orphans from scaling in as destroy nodes.
		&ConfigTransformer{
			Config: b.Config,
		},

		// Add dynamic values
		&RootVariableTransformer{Config: b.Config},
		&ModuleVariableTransformer{Config: b.Config},
		&LocalTransformer{Config: b.Config},
		&OutputTransformer{Config: b.Config},

		// Attach the configuration to any resources
		&AttachResourceConfigTransformer{Config: b.Config},

		// Attach the state
		&AttachStateTransformer{State: b.State},

		TransformProviders(b.Components.ResourceProviders(), concreteProvider, b.Config),

		// Must attach schemas before ReferenceTransformer so that we can
		// analyze the configuration to find references.
		&AttachSchemaTransformer{Schemas: b.Schemas, Config: b.Config},

		// Create expansion nodes for all of the module calls. This must
		// come after all other transformers that create nodes representing
		// objects that can belong to modules.
		&ModuleExpansionTransformer{Config: b.Config},

		// Connect so that the references are ready for targeting. We'll
		// have to connect again later for providers and so on.
		&ReferenceTransformer{},

		// Although we don't configure providers, we do still start them up
		// to get their schemas, and so we must shut them down again here.
		&CloseProviderTransformer{},

		// Close root module
		&CloseRootModuleTransformer{},

		// Remove redundant edges to simplify the graph.
		&TransitiveReductionTransformer{},
	}

	return steps
}
