package readmodel

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/pkg/oddc"
)

type fakeModelResolver struct {
	modelID string
	result  oddc.Resolved
	err     error
	calls   int
}

func (f *fakeModelResolver) ResolveModel(
	modelID string,
	_ []oddc.Overlay,
	_ []oddc.Overlay,
) (oddc.Resolved, error) {
	f.calls++
	f.modelID = modelID

	return f.result, f.err
}

func TestCatalogResolvedSourceUsesInjectedModel(t *testing.T) {
	resolver := &fakeModelResolver{
		result: oddc.Resolved{
			ModelID: "model/synthetic",
			Resolved: map[string]any{
				"model": map[string]any{
					"id": "model/synthetic",
				},
			},
		},
	}

	source := CatalogResolvedSource{
		Root:    "/synthetic/oddc",
		ModelID: "model/synthetic",
		Load: func(
			root string,
		) (modelResolver, error) {
			if root != "/synthetic/oddc" {
				t.Fatalf(
					"root = %q",
					root,
				)
			}

			return resolver, nil
		},
	}

	got, err := source.Resolved(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("resolved ODDC object is nil")
	}

	if resolver.calls != 1 {
		t.Fatalf(
			"resolve calls = %d, want 1",
			resolver.calls,
		)
	}

	if resolver.modelID != "model/synthetic" {
		t.Fatalf(
			"model ID = %q",
			resolver.modelID,
		)
	}
}

func TestCatalogResolvedSourceRejectsMissingModel(
	t *testing.T,
) {
	source := CatalogResolvedSource{
		Root: "/synthetic/oddc",
		Load: func(
			string,
		) (modelResolver, error) {
			t.Fatal("loader called without model ID")
			return nil, nil
		},
	}

	if _, err := source.Resolved(
		context.Background(),
	); err == nil {
		t.Fatal("missing ODDC model ID accepted")
	}
}

func TestCatalogResolvedSourcePropagatesLoaderFailure(
	t *testing.T,
) {
	source := CatalogResolvedSource{
		Root:    "/synthetic/oddc",
		ModelID: "model/synthetic",
		Load: func(
			string,
		) (modelResolver, error) {
			return nil,
				errors.New("synthetic registry failure")
		},
	}

	_, err := source.Resolved(
		context.Background(),
	)
	if err == nil {
		t.Fatal("registry load failure became success")
	}

	if !strings.Contains(
		err.Error(),
		"synthetic registry failure",
	) {
		t.Fatalf(
			"unexpected error %q",
			err,
		)
	}
}

func TestCatalogResolvedSourcePropagatesResolveFailure(
	t *testing.T,
) {
	resolver := &fakeModelResolver{
		err: errors.New("synthetic resolve failure"),
	}

	source := CatalogResolvedSource{
		Root:    "/synthetic/oddc",
		ModelID: "model/synthetic",
		Load: func(
			string,
		) (modelResolver, error) {
			return resolver, nil
		},
	}

	_, err := source.Resolved(
		context.Background(),
	)
	if err == nil {
		t.Fatal("ODDC resolve failure became success")
	}

	if !strings.Contains(
		err.Error(),
		"synthetic resolve failure",
	) {
		t.Fatalf(
			"unexpected error %q",
			err,
		)
	}
}

func TestCatalogResolvedSourceHonorsCancellation(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	source := CatalogResolvedSource{
		Root:    "/synthetic/oddc",
		ModelID: "model/synthetic",
	}

	if _, err := source.Resolved(ctx); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf(
			"cancellation error = %v",
			err,
		)
	}
}

func TestRepositoryCatalogSourceResolvesCurrentModel(
	t *testing.T,
) {
	source := NewCatalogResolvedSource(
		"../../../oddc",
		"model/framework/laptop-13-amd-ryzen-7040",
	)

	resolved, err := source.Resolved(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	model, ok := resolved["model"].(map[string]any)
	if !ok {
		t.Fatalf(
			"resolved model metadata = %#v",
			resolved["model"],
		)
	}

	if model["id"] !=
		"model/framework/laptop-13-amd-ryzen-7040" {
		t.Fatalf(
			"model ID = %#v",
			model["id"],
		)
	}
}
