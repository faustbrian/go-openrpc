package openrpc_test

import (
	"fmt"

	openrpc "github.com/faustbrian/go-openrpc"
	"github.com/faustbrian/go-openrpc/builder"
)

func Example() {
	version, err := openrpc.ParseVersion("1.4.1")
	if err != nil {
		fmt.Println(err)
		return
	}
	info, err := openrpc.NewInfo(openrpc.InfoInput{
		Title:   "Calculator",
		Version: "1.0.0",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	method, err := openrpc.NewMethod(openrpc.MethodInput{
		Name:   "add",
		Params: []openrpc.ContentDescriptorOrReference{},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	documentBuilder, err := builder.NewDocument(version, info)
	if err != nil {
		fmt.Println(err)
		return
	}
	documentBuilder, err = documentBuilder.WithMethod(method)
	if err != nil {
		fmt.Println(err)
		return
	}
	document, err := documentBuilder.Build()
	if err != nil {
		fmt.Println(err)
		return
	}
	encoded, err := openrpc.MarshalCanonical(document)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(encoded))
	// Output: {"info":{"title":"Calculator","version":"1.0.0"},"methods":[{"name":"add","params":[]}],"openrpc":"1.4.1"}
}
