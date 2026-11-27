// Package api holds the server types and strict-server interface generated from
// the OpenAPI specification. The specification is itself generated from the
// TypeSpec definition in the repository's api directory.
//
// Run `go generate ./internal/api` after changing api/main.tsp. Only the first
// step needs Node (run `npm install` in api/ once); a plain `go build` uses the
// committed output.
//
// The oapi-codegen go:generate form follows the oapi-codegen README:
// https://github.com/oapi-codegen/oapi-codegen#installing ("go tool" usage).
package api

//go:generate npm --prefix ../../api run build
//go:generate go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi.yaml
