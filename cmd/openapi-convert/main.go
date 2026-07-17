package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"sigs.k8s.io/yaml"
)

func main() {
	input := flag.String("input", "docs/swagger.json", "Swagger 2 input file")
	jsonOutput := flag.String("json", "docs/swagger.json", "OpenAPI 3 JSON output file")
	yamlOutput := flag.String("yaml", "docs/swagger.yaml", "OpenAPI 3 YAML output file")
	flag.Parse()

	if err := convert(*input, *jsonOutput, *yamlOutput); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func convert(input, jsonOutput, yamlOutput string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read Swagger document: %w", err)
	}

	var swagger openapi2.T
	if err := json.Unmarshal(source, &swagger); err != nil {
		return fmt.Errorf("parse Swagger document: %w", err)
	}

	document, err := openapi2conv.ToV3(&swagger)
	if err != nil {
		return fmt.Errorf("convert Swagger document: %w", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		return fmt.Errorf("validate OpenAPI document: %w", err)
	}

	jsonDocument, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize OpenAPI JSON: %w", err)
	}
	jsonDocument = append(jsonDocument, 10)

	yamlDocument, err := yaml.JSONToYAML(jsonDocument)
	if err != nil {
		return fmt.Errorf("serialize OpenAPI YAML: %w", err)
	}

	if err := os.WriteFile(jsonOutput, jsonDocument, 0o644); err != nil {
		return fmt.Errorf("write OpenAPI JSON: %w", err)
	}
	if err := os.WriteFile(yamlOutput, yamlDocument, 0o644); err != nil {
		return fmt.Errorf("write OpenAPI YAML: %w", err)
	}

	return nil
}
