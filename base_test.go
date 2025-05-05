package helmbase

import (
	"context"
	"reflect"
	"testing"

	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/provider"
	"github.com/stretchr/testify/assert"
)

// MockResourceMonitor implements pulumi.MockResourceMonitor to mock resource creation
type MockResourceMonitor struct{}

// NewResource mocks the creation of a Pulumi resource
func (m *MockResourceMonitor) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	// Return a mock ID and outputs
	inputs := make(map[string]interface{})
	for k, v := range args.Inputs {
		inputs[string(k)] = v
	}
	return "mockResourceID", resource.NewPropertyMapFromMap(inputs), nil
}

// Call mocks the provider call function
func (m *MockResourceMonitor) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	// Return mock outputs for a provider call
	outputs := resource.PropertyMap{}
	for k, v := range args.Args {
		outputs[resource.PropertyKey(k)] = v
	}
	return outputs, nil
}

type MockComponentResource struct {
	pulumi.ResourceState
	Name pulumi.StringInput
}

func NewMockComponentResource(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) *MockComponentResource {
	// Initialize ResourceState
	res := &MockComponentResource{
		ResourceState: pulumi.ResourceState{},
		Name:          pulumi.String(name),
	}

	// Register the resource
	err := ctx.RegisterComponentResource("mock:index:Component", name, res, opts...)
	if err != nil {
		panic(pulumi.Sprintf("Failed to register component resource: %v", err))
	}

	// Ensure that the resource is correctly initialized
	pulumi.Sprintf("registered resource: %s", res.ResourceState.URN())
	return res
}

type MockChart struct {
	pulumi.ResourceState
	*MockComponentResource
	typeName      string
	chartName     string
	repoURL       string
	outputsStatus helmv3.ReleaseStatusOutput
}

func NewMockChart(ctx *pulumi.Context, typeName, chartName, repoURL string, opts ...pulumi.ResourceOption) *MockChart {
	return &MockChart{
		MockComponentResource: NewMockComponentResource(ctx, chartName, opts...),
		typeName:              typeName,
		chartName:             chartName,
		repoURL:               repoURL,
	}
}

func (m *MockChart) Type() string                              { return m.typeName }
func (m *MockChart) SetOutputs(out helmv3.ReleaseStatusOutput) { m.outputsStatus = out }
func (m *MockChart) DefaultChartName() string                  { return m.chartName }
func (m *MockChart) DefaultRepoURL() string                    { return m.repoURL }

type MockArgs struct {
	release *ReleaseTypeArgs
}

// ToReleaseTypeOutputWithContext implements ReleaseTypeInput.
func (m *MockArgs) ToReleaseTypeOutputWithContext(ctx context.Context) ReleaseTypeOutput {
	return pulumi.ToOutputWithContext(ctx, m).(ReleaseTypeOutput)
}

func (m *MockArgs) R() **ReleaseTypeArgs {
	return &m.release
}

func (m *MockArgs) ToChartOutput() ReleaseTypeOutput {
	return ReleaseTypeOutput{}
}

func (m *MockArgs) ToReleaseTypeOutput() ReleaseTypeOutput {
	return ReleaseTypeOutput{}
}

func (m *MockArgs) ElementType() reflect.Type {
	return reflect.TypeOf(m).Elem()
}

func TestConstruct(t *testing.T) {
	tests := []struct {
		name          string
		chart         *MockChart
		args          *MockArgs
		typeName      string
		expectedError bool
	}{
		{
			name:  "valid construction",
			chart: nil, // Will be initialized later
			args: &MockArgs{
				release: &ReleaseTypeArgs{
					Values: map[string]interface{}{},
				},
			},
			typeName:      "test:index:Chart",
			expectedError: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := pulumi.RunErr(func(ctx *pulumi.Context) error {
				// Initialize the mock chart with the Pulumi context
				tc.chart = NewMockChart(ctx, tc.typeName, "test-chart", "https://charts.test.org")

				// Ensure inputs are a compatible type
				inputs := provider.ConstructInputs{}

				// Call Construct
				assert.NotNil(t, tc.chart)
				ro := pulumi.Composite()
				result, err := Construct(ctx, tc.chart, tc.typeName, "test-name", tc.args, inputs, ro)

				// Validate results
				if tc.expectedError {
					assert.Error(t, err)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
				return err
			}, pulumi.WithMocks("test-project", "test-stack", &MockResourceMonitor{}))
			if err != nil {
				t.Fatalf("Pulumi.Run failed: %v", err)
			}
		})
	}
}

func TestInitDefaults(t *testing.T) {
	tests := []struct {
		name       string
		args       *ReleaseTypeArgs
		chart      string
		repo       string
		values     interface{}
		wantValues map[string]interface{}
	}{
		{
			name:  "empty args",
			args:  &ReleaseTypeArgs{},
			chart: "test-chart",
			repo:  "https://test-repo.com",
			values: struct {
				TestValue string `pulumi:"testValue"`
			}{
				TestValue: "test",
			},
			wantValues: map[string]interface{}{
				"testValue": "test",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			InitDefaults(tc.args, tc.chart, tc.repo, tc.values)

			assert.NotNil(t, tc.args.Chart)
			assert.NotNil(t, tc.args.RepositoryOpts.Repo)
			assert.Equal(t, tc.wantValues, tc.args.Values)
		})
	}
}
