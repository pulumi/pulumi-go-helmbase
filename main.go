package helmbase

import (
	"fmt"
	"os"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

func main() {
	err := p.RunProvider("helmBase", "0.1.0", provider())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func provider() p.Provider {
	return infer.Provider(infer.Options{
		Functions: []infer.InferredFunction{},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"helmBase": "index",
		},
	})
}
