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
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/provider"
)

const (
	FieldHelmReleaseOutput = "helmRelease"
	FieldHelmOptionsInput  = "helmOptions"
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
	SetOutputs(rel *helmv3.Release)
	// DefaultChartName returns the default name for this chart.
	DefaultChartName() string
	// DefaultRepo returns the default Helm repo URL for this chart.
	DefaultRepoURL() string
}

// ChartArgs is a properly annotated structure (with `pulumi:""` and `json:""` tags)
// which carries the strongly typed argument payload for the given Chart resource.
type ChartArgs interface {
	R() **helmv3.ReleaseType
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
		*relArgs = &helmv3.ReleaseType{}
	}
	InitDefaults(*relArgs, c.DefaultChartName(), c.DefaultRepoURL(), args)

	// Create the actual underlying Helm Chart resource.
	rel, err := helmv3.NewRelease(ctx, name+"-helm", To(*relArgs), pulumi.Parent(c))
	if err != nil {
		return nil, err
	}
	c.SetOutputs(rel)

	// Finally, register the resulting Helm Release as a component output.
	if err := ctx.RegisterResourceOutputs(c, pulumi.Map{
		FieldHelmReleaseOutput: rel,
	}); err != nil {
		return nil, err
	}

	return provider.NewConstructResult(c)
}

// InitDefaults copies the default chart, repo, and values onto the args struct.
func InitDefaults(args *helmv3.ReleaseType, chart, repo string, values interface{}) {
	// Most strongly typed charts will have a default chart name as well as a default
	// repository location. If available, set those. The user might override these,
	// so only initialize them if they're empty.
	if args.Chart == "" {
		args.Chart = chart
	}
	if args.RepositoryOpts.Repo == nil {
		args.RepositoryOpts.Repo = &repo
	}

	// Blit the strongly typed values onto the weakly typed values, so that the Helm
	// Release is constructed properly. In the event a value is present in both, the
	// strongly typed values override the weakly typed map.
	if args.Values == nil {
		args.Values = make(map[string]interface{})
	}

	// Decode the structure into the target map so we can copy it over to the values
	// map, which is what the Helm Release expects. We use the `pulumi:"x"`
	// tags to drive the naming of the resulting properties.
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &args.Values,
		TagName: "pulumi",
	})
	if err != nil {
		panic(err)
	}
	if err = d.Decode(values); err != nil {
		panic(err)
	}

	// Delete the HelmOptions input value -- it's not helpful and would cause a cycle.
	delete(args.Values, FieldHelmOptionsInput)
}

func toBoolPtr(p *bool) pulumi.BoolPtrInput {
	if p == nil {
		return nil
	}
	return pulumi.BoolPtr(*p)
}

func toIntPtr(p *int) pulumi.IntPtrInput {
	if p == nil {
		return nil
	}
	return pulumi.IntPtr(*p)
}

func toStringPtr(p *string) pulumi.StringPtrInput {
	if p == nil {
		return nil
	}
	return pulumi.StringPtr(*p)
}

func toAssetOrArchiveArray(a []pulumi.AssetOrArchive) pulumi.AssetOrArchiveArray {
	var res pulumi.AssetOrArchiveArray
	// TODO: ?!?!?!
	// cannot use e (variable of type pulumi.AssetOrArchive) as pulumi.AssetOrArchiveInput value in argument to append
	/*
		for _, e := range a {
			res = append(res, e)
		}
	*/
	return res
}

// To turns the args struct into a Helm-ready ReleaseArgs struct.
func To(args *helmv3.ReleaseType) *helmv3.ReleaseArgs {
	// Create the Helm Release args.
	// TODO: it would be nice to do this automatically, e.g. using reflection, etc.
	//     This is caused by the helm.ReleaseArgs type not actually having the struct
	//     tags we need to use it directly (not clear why this is the case!)
	//     https://github.com/pulumi/pulumi/issues/8112
	return &helmv3.ReleaseArgs{
		Atomic:                   toBoolPtr(args.Atomic),
		Chart:                    pulumi.String(args.Chart),
		CleanupOnFail:            toBoolPtr(args.CleanupOnFail),
		CreateNamespace:          toBoolPtr(args.CreateNamespace),
		DependencyUpdate:         toBoolPtr(args.DependencyUpdate),
		Description:              toStringPtr(args.Description),
		Devel:                    toBoolPtr(args.Devel),
		DisableCRDHooks:          toBoolPtr(args.DisableCRDHooks),
		DisableOpenapiValidation: toBoolPtr(args.DisableOpenapiValidation),
		DisableWebhooks:          toBoolPtr(args.DisableWebhooks),
		ForceUpdate:              toBoolPtr(args.ForceUpdate),
		Keyring:                  toStringPtr(args.Keyring),
		Lint:                     toBoolPtr(args.Lint),
		Manifest:                 pulumi.ToMap(args.Manifest),
		MaxHistory:               toIntPtr(args.MaxHistory),
		Name:                     toStringPtr(args.Name),
		Namespace:                toStringPtr(args.Namespace),
		Postrender:               toStringPtr(args.Postrender),
		RecreatePods:             toBoolPtr(args.RecreatePods),
		RenderSubchartNotes:      toBoolPtr(args.RenderSubchartNotes),
		Replace:                  toBoolPtr(args.Replace),
		RepositoryOpts: &helmv3.RepositoryOptsArgs{
			CaFile:   toStringPtr(args.RepositoryOpts.CaFile),
			CertFile: toStringPtr(args.RepositoryOpts.CertFile),
			KeyFile:  toStringPtr(args.RepositoryOpts.KeyFile),
			Password: toStringPtr(args.RepositoryOpts.Password),
			Repo:     toStringPtr(args.RepositoryOpts.Repo),
			Username: toStringPtr(args.RepositoryOpts.Username),
		},
		ResetValues:    toBoolPtr(args.ResetValues),
		ResourceNames:  pulumi.ToStringArrayMap(args.ResourceNames),
		ReuseValues:    toBoolPtr(args.ReuseValues),
		SkipAwait:      toBoolPtr(args.SkipAwait),
		SkipCrds:       toBoolPtr(args.SkipCrds),
		Timeout:        toIntPtr(args.Timeout),
		ValueYamlFiles: toAssetOrArchiveArray(args.ValueYamlFiles),
		Values:         pulumi.ToMap(args.Values),
		Verify:         toBoolPtr(args.Verify),
		Version:        toStringPtr(args.Version),
		WaitForJobs:    toBoolPtr(args.WaitForJobs),
	}
}
