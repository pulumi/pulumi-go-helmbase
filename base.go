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
	FieldHelmStatusOutput = "status"
	FieldHelmOptionsInput = "helmOptions"
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
	SetOutputs(out helmv3.ReleaseStatusOutput)
	// DefaultChartName returns the default name for this chart.
	DefaultChartName() string
	// DefaultRepo returns the default Helm repo URL for this chart.
	DefaultRepoURL() string
}

// ReleaseType added because it was deprecated upstream.
type ReleaseType struct {
	// If set, installation process purges chart on fail. `skipAwait` will be disabled automatically if atomic is used.
	Atomic pulumi.BoolPtrInput `pulumi:"atomic"`
	// Chart name to be installed. A path may be used.
	Chart pulumi.StringInput `pulumi:"chart"`
	// Allow deletion of new resources created in this upgrade when upgrade fails.
	CleanupOnFail pulumi.BoolPtrInput `pulumi:"cleanupOnFail"`
	// Create the namespace if it does not exist.
	CreateNamespace pulumi.BoolPtrInput `pulumi:"createNamespace"`
	// Run helm dependency update before installing the chart.
	DependencyUpdate pulumi.BoolPtrInput `pulumi:"dependencyUpdate"`
	// Add a custom description
	Description pulumi.StringPtrInput `pulumi:"description"`
	// Use chart development versions, too. Equivalent to version '>0.0.0-0'. If `version` is set, this is ignored.
	Devel pulumi.BoolPtrInput `pulumi:"devel"`
	// Prevent CRD hooks from, running, but run other hooks.  See helm install --no-crd-hook
	DisableCRDHooks pulumi.BoolPtrInput `pulumi:"disableCRDHooks"`
	// If set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema
	DisableOpenapiValidation pulumi.BoolPtrInput `pulumi:"disableOpenapiValidation"`
	// Prevent hooks from running.
	DisableWebhooks pulumi.BoolPtrInput `pulumi:"disableWebhooks"`
	// Force resource update through delete/recreate if needed.
	ForceUpdate pulumi.BoolPtrInput `pulumi:"forceUpdate"`
	// Location of public keys used for verification. Used only if `verify` is true
	Keyring pulumi.StringPtrInput `pulumi:"keyring"`
	// Run helm lint when planning.
	Lint pulumi.BoolPtrInput `pulumi:"lint"`
	// The rendered manifests as JSON. Not yet supported.
	Manifest pulumi.MapInput `pulumi:"manifest"`
	// Limit the maximum number of revisions saved per release. Use 0 for no limit.
	MaxHistory pulumi.IntPtrInput `pulumi:"maxHistory"`
	// Release name.
	Name pulumi.StringPtrInput `pulumi:"name"`
	// Namespace to install the release into.
	Namespace pulumi.StringPtrInput `pulumi:"namespace"`
	// Postrender command to run.
	Postrender pulumi.StringPtrInput `pulumi:"postrender"`
	// Perform pods restart during upgrade/rollback.
	RecreatePods pulumi.BoolPtrInput `pulumi:"recreatePods"`
	// If set, render subchart notes along with the parent.
	RenderSubchartNotes pulumi.BoolPtrInput `pulumi:"renderSubchartNotes"`
	// Re-use the given name, even if that name is already used. This is unsafe in production
	Replace pulumi.BoolPtrInput `pulumi:"replace"`
	// Specification defining the Helm chart repository to use.
	RepositoryOpts helmv3.RepositoryOptsArgs `pulumi:"repositoryOpts"`
	// When upgrading, reset the values to the ones built into the chart.
	ResetValues pulumi.BoolPtrInput `pulumi:"resetValues"`
	// Names of resources created by the release grouped by "kind/version".
	ResourceNames pulumi.StringArrayMapInput `pulumi:"resourceNames"`
	// When upgrading, reuse the last release's values and merge in any overrides. If 'resetValues' is specified, this is ignored
	ReuseValues pulumi.BoolPtrInput `pulumi:"reuseValues"`
	// By default, the provider waits until all resources are in a ready state before marking the release as successful. Setting this to true will skip such await logic.
	SkipAwait pulumi.BoolPtrInput `pulumi:"skipAwait"`
	// If set, no CRDs will be installed. By default, CRDs are installed if not already present.
	SkipCrds pulumi.BoolPtrInput `pulumi:"skipCrds"`
	// Status of the deployed release.
	Status helmv3.ReleaseStatus `pulumi:"status"`
	// Time in seconds to wait for any individual kubernetes operation.
	Timeout pulumi.IntPtrInput `pulumi:"timeout"`
	// List of assets (raw yaml files). Content is read and merged with values. Not yet supported.
	ValueYamlFiles []pulumi.AssetOrArchive `pulumi:"valueYamlFiles"`
	// Custom values set for the release.
	Values map[string]interface{} `pulumi:"values"`
	// Verify the package before installing it.
	Verify pulumi.BoolPtrInput `pulumi:"verify"`
	// Specify the exact chart version to install. If this is not specified, the latest version is installed.
	Version pulumi.StringPtrInput `pulumi:"version"`
	// Will wait until all Jobs have been completed before marking the release as successful. This is ignored if `skipAwait` is enabled.
	WaitForJobs pulumi.BoolPtrInput `pulumi:"waitForJobs"`
}

// ChartArgs is a properly annotated structure (with `pulumi:""` and `json:""` tags)
// which carries the strongly typed argument payload for the given Chart resource.
type ChartArgs interface {
	R() **ReleaseType
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
	if err := ctx.RegisterComponentResource(typ, name, c, opts); err != nil {
		return nil, err
	}

	// Provide default values for the Helm Release, including the chart name, repository
	// to pull from, and blitting the strongly typed values into the weakly typed map.
	relArgs := args.R()
	if *relArgs == nil {
		*relArgs = &ReleaseType{}
	}
	InitDefaults(*relArgs, c.DefaultChartName(), c.DefaultRepoURL(), args)

	// Create the actual underlying Helm Chart resource.
	rel, err := helmv3.NewRelease(ctx, name+"-helm", To(*relArgs), pulumi.Parent(c))
	if err != nil {
		return nil, err
	}
	c.SetOutputs(rel.Status)

	// Finally, register the resulting Helm Release as a component output.
	if err := ctx.RegisterResourceOutputs(c, pulumi.Map{
		FieldHelmStatusOutput: rel,
	}); err != nil {
		return nil, err
	}

	return provider.NewConstructResult(c)
}

// InitDefaults copies the default chart, repo, and values onto the args struct.
func InitDefaults(args *ReleaseType, chart, repo string, values interface{}) {
	// Most strongly typed charts will have a default chart name as well as a default
	// repository location. If available, set those. The user might override these,
	// so only initialize them if they're empty.
	if args.Chart == nil {
		args.Chart = pulumi.String(chart)
	}
	if args.RepositoryOpts.Repo == nil {
		args.RepositoryOpts.Repo = toStringPtr(&repo)
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
func To(args *ReleaseType) *helmv3.ReleaseArgs {
	// Create the Helm Release args.
	// TODO: it would be nice to do this automatically, e.g. using reflection, etc.
	//     This is caused by the helm.ReleaseArgs type not actually having the struct
	//     tags we need to use it directly (not clear why this is the case!)
	//     https://github.com/pulumi/pulumi/issues/8112
	return &helmv3.ReleaseArgs{
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
		ValueYamlFiles:           toAssetOrArchiveArray(args.ValueYamlFiles),
		Values:                   pulumi.ToMap(args.Values),
		Verify:                   args.Verify,
		Version:                  args.Version,
		WaitForJobs:              args.WaitForJobs,
	}
}
