package analysis

import "testing"

const (
	widgetFile = "fixtures/widget-crud.yml"
	fooFile    = "fixtures/foo-crud.yml"
	barFile    = "fixtures/bar-crud.yml"
)

func TestMixin(t *testing.T) {

	primary, err := loadSpec(widgetFile)
	if err != nil {
		t.Fatalf("Could not load '%v': %v\n", widgetFile, err)
	}
	mixin1, err := loadSpec(fooFile)
	if err != nil {
		t.Fatalf("Could not load '%v': %v\n", fooFile, err)
	}
	mixin2, err := loadSpec(barFile)
	if err != nil {
		t.Fatalf("Could not load '%v': %v\n", barFile, err)
	}

	collisions := Mixin(primary, mixin1, mixin2)
	if len(collisions) != 12 {
		t.Errorf("TestMixin: Expected 10 collisions, got %v\n", len(collisions))
	}

	if len(primary.Paths.Paths) != 7 {
		t.Errorf("TestMixin: Expected 7 paths in merged, got %v\n", len(primary.Paths.Paths))
	}

	if len(primary.Definitions) != 8 {
		t.Errorf("TestMixin: Expected 8 definitions in merged, got %v\n", len(primary.Definitions))
	}

	if len(primary.Parameters) != 4 {
		t.Errorf("TestMixin: Expected 4 top level parameters in merged, got %v\n", len(primary.Parameters))
	}

	if len(primary.Responses) != 2 {
		t.Errorf("TestMixin: Expected 2 top level responses in merged, got %v\n", len(primary.Responses))
	}

}
