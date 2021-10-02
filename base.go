// Copyright 2021, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helmbase

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/provider"
)

// Chart represents a strongly typed Helm Chart resource. For the most part,
// it merely participates in the Pulumi resource lifecycle (by virtue of extending
// pulumi.ComponentResource), but it also offers a few specific helper methods.
type Chart interface {
	pulumi.ComponentResource
	// Type returns the fully qualified Pulumi type token for this resource.
	Type() string
	// SetOutputs registers the resulting Helm Release child resource, after it
	// has been created and registered. This contains the Status, among other things.
	SetOutputs(rel *helm.Release)
	// DefaultChartName returns the default name for this chart.
	DefaultChartName() string
	// DefaultRepo returns the default Helm repo URL for this chart.
	DefaultRepoURL() string
}

// ChartArgs is a properly annotated structure (with `pulumi:""` and `json:""` tags)
// which carries the strongly typed argument payload for the given Chart resource.
type ChartArgs interface {
	R() **ReleaseArgs
}

// Construct is the RPC call that initiates the creation of a new Chart component. It
// creates, registers, and returns the resulting component object. This contains most of
// the boilerplate so that the calling component can be relatively simple.
func Construct(ctx *pulumi.Context, c Chart, typ, name string,
	args ChartArgs, inputs provider.ConstructInputs, opts pulumi.ResourceOption) (*provider.ConstructResult, error) {

	// Ensure we have the right token.
	if et := c.Type(); typ != et {
		return nil, errors.Errorf("unknown resource type %s; expected %s", typ, et)
	}

	// Blit the inputs onto the arguments struct.
	if err := inputs.CopyTo(args); err != nil {
		return nil, errors.Wrap(err, "setting args")
	}

	// Register our component resource.
	err := ctx.RegisterComponentResource(typ, name, c, opts)
	if err != nil {
		return nil, err
	}

	// Provide default values for the Helm Release, including the chart name, repository
	// to pull from, and blitting the strongly typed values into the weakly typed map.
	relArgs := args.R()
	if *relArgs == nil {
		*relArgs = &ReleaseArgs{}
	}
	(*relArgs).InitDefaults(c.DefaultChartName(), c.DefaultRepoURL(), args)

	// Create the actual underlying Helm Chart resource.
	rel, err := helm.NewRelease(ctx, name+"-helm", (*relArgs).To(), pulumi.Parent(c))
	if err != nil {
		return nil, err
	}
	c.SetOutputs(rel)

	// Finally, register the resulting Helm Release as a component output.
	if err := ctx.RegisterResourceOutputs(c, pulumi.Map{
		"helmRelease": rel,
	}); err != nil {
		return nil, err
	}

	return provider.NewConstructResult(c)
}

// ReleaseArgs is a lot like helm.ReleaseArgs, except that it doesn't require the
// "chart" and "repositoryOpts" arguments, since there are sensible defaults for
// those thanks to this being a strongly typed chart component.
// TODO: wish we could just reuse the *helm.ReleaseArgs type; see
//     https://github.com/pulumi/pulumi/issues/8114 for details on why we can't.
type ReleaseArgs struct {
	helm.ReleaseTypeArgs
}

// InitDefaults copies the default chart, repo, and values onto the args struct.
func (args *ReleaseArgs) InitDefaults(chart, repo string, values interface{}) {
	// Most strongly typed charts will have a default chart name as well as a default
	// repository location. If available, set those. The user might override these,
	// so only initialize them if they're empty.
	if args.Chart == nil {
		args.Chart = pulumi.String(chart)
	}
	if args.RepositoryOpts == nil {
		args.RepositoryOpts = &helm.RepositoryOptsArgs{
			Repo: pulumi.String(repo),
		}
	}

	// Blit the strongly typed values onto the weakly typed values, so that the Helm
	// Release is constructed properly. In the event a value is present in both, the
	// strongly typed values override the weakly typed map.
	if args.Values == nil {
		args.Values = pulumi.Map{}
	}
	args.Values = args.Values.ToMapOutput().ApplyT(
		func(t interface{}) map[string]interface{} {
			// Decode the structure into a map so we can copy it over to the values
			// map, which is what the Helm Release expects. We use the `pulumi:"x"`
			// tags to drive the naming of the resulting properties.
			var src map[string]interface{}
			d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				TagName: "pulumi",
				Result:  &src,
			})
			if err != nil {
				panic(err)
			}
			if err = d.Decode(values); err != nil {
				panic(err)
			}

			dst := t.(map[string]interface{})
			for k, v := range src {
				// TODO: should we do something special about deeply merging sub-maps?
				dst[k] = v
			}
			return dst
		},
	).(pulumi.MapOutput)
}

// To turns the args struct into a Helm-ready ReleaseArgs struct.
func (args *ReleaseArgs) To() *helm.ReleaseArgs {
	// Create the Helm Release args.
	// TODO: it would be nice to do this automatically, e.g. using reflection, etc.
	//     This is caused by the helm.ReleaseArgs type not actually having the struct
	//     tags we need to use it directly (not clear why this is the case!)
	//     https://github.com/pulumi/pulumi/issues/8112
	return &helm.ReleaseArgs{
		Atomic:                   args.Atomic,
		Chart:                    args.Chart,
		CleanupOnFail:            args.CleanupOnFail,
		CreateNamespace:          args.CreateNamespace,
		DependencyUpdate:         args.DependencyUpdate,
		Description:              args.Description,
		Devel:                    args.Devel,
		DisableCRDHooks:          args.DisableCRDHooks,
		DisableOpenapiValidation: args.DisableOpenapiValidation,
		DisableWebhooks:          args.DisableWebhooks,
		ForceUpdate:              args.ForceUpdate,
		Keyring:                  args.Keyring,
		Lint:                     args.Lint,
		Manifest:                 args.Manifest,
		MaxHistory:               args.MaxHistory,
		Name:                     args.Name,
		Namespace:                args.Namespace,
		Postrender:               args.Postrender,
		RecreatePods:             args.RecreatePods,
		RenderSubchartNotes:      args.RenderSubchartNotes,
		Replace:                  args.Replace,
		RepositoryOpts:           args.RepositoryOpts,
		ResetValues:              args.ResetValues,
		ResourceNames:            args.ResourceNames,
		ReuseValues:              args.ReuseValues,
		SkipAwait:                args.SkipAwait,
		SkipCrds:                 args.SkipCrds,
		Timeout:                  args.Timeout,
		ValueYamlFiles:           args.ValueYamlFiles,
		Values:                   args.Values,
		Verify:                   args.Verify,
		Version:                  args.Version,
		WaitForJobs:              args.WaitForJobs,
	}
}
