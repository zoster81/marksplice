package publictest

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestM110ExtensionOverlayIsExplicitNamespacedAndReadOnly(t *testing.T) {
	t.Parallel()

	source := []byte("before [[page]] after\n")
	baseline, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := baseline.ExtensionNodes(); got != nil {
		t.Fatalf("baseline ExtensionNodes() = %+v, want nil", got)
	}

	calls := 0
	extension := marksplice.Extension{
		ID: "example.org/wiki",
		Recognize: func(input marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
			calls++
			if got := input.Text(); got != string(source) {
				t.Fatalf("ExtensionSource.Text() = %q, want %q", got, source)
			}
			return []marksplice.ExtensionMatch{{
				Kind:  "wikilink",
				Range: marksplice.Range{Start: 7, End: 15},
				Attributes: []marksplice.ExtensionAttribute{
					{Name: "target", Value: "page"},
				},
			}}, nil
		},
	}
	document, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{
		Extensions: []marksplice.Extension{extension},
		ExtensionLimits: marksplice.ExtensionLimits{
			MaxNodes:         4,
			MaxMetadataBytes: 256,
		},
	})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("recognizer calls = %d, want 1", calls)
	}

	nodes := document.ExtensionNodes()
	if len(nodes) != 1 {
		t.Fatalf("ExtensionNodes() count = %d, want 1", len(nodes))
	}
	node := nodes[0]
	if got := node.ExtensionID(); got != "example.org/wiki" {
		t.Fatalf("ExtensionID() = %q", got)
	}
	if got := node.Kind(); got != "wikilink" {
		t.Fatalf("Kind() = %q", got)
	}
	if got := node.Range(); got != (marksplice.Range{Start: 7, End: 15}) {
		t.Fatalf("Range() = %+v", got)
	}
	if got, ok := node.Attribute("target"); !ok || got != "page" {
		t.Fatalf("Attribute(target) = %q/%v", got, ok)
	}
	attributes := node.Attributes()
	if !reflect.DeepEqual(attributes, []marksplice.ExtensionAttribute{{Name: "target", Value: "page"}}) {
		t.Fatalf("Attributes() = %+v", attributes)
	}
	attributes[0].Value = "changed"
	if got, ok := document.ExtensionNodes()[0].Attribute("target"); !ok || got != "page" {
		t.Fatalf("caller mutation leaked into extension metadata: %q/%v", got, ok)
	}
	source[7] = 'X'
	owned, ok := document.SourceRange(node.Range())
	if !ok || string(owned) != "[[page]]" {
		t.Fatalf("caller source mutation leaked into document snapshot: %q/%v", owned, ok)
	}

	coreNodes := document.Nodes()
	if len(coreNodes) != 1 || coreNodes[0].Kind() != marksplice.KindParagraph {
		t.Fatalf("core Nodes() = %+v, want unchanged paragraph", coreNodes)
	}
}

func TestM110ExtensionValidationIsFailClosedAndBudgeted(t *testing.T) {
	t.Parallel()

	source := []byte("text\n")
	callbackErr := errors.New("recognizer failed")
	tests := []struct {
		name      string
		options   marksplice.ParseOptions
		wantCause error
	}{
		{
			name: "zero limits",
			options: marksplice.ParseOptions{Extensions: []marksplice.Extension{{
				ID: "example.org/zero",
				Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
					return nil, nil
				},
			}}},
			wantCause: marksplice.ErrInvalidExtension,
		},
		{
			name: "duplicate id",
			options: marksplice.ParseOptions{
				Extensions: []marksplice.Extension{
					{ID: "example.org/dup", Recognize: noM110Matches},
					{ID: "example.org/dup", Recognize: noM110Matches},
				},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 64},
			},
			wantCause: marksplice.ErrInvalidExtension,
		},
		{
			name: "invalid range",
			options: marksplice.ParseOptions{
				Extensions: []marksplice.Extension{{
					ID: "example.org/range",
					Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
						return []marksplice.ExtensionMatch{{Kind: "bad", Range: marksplice.Range{Start: 0, End: 99}}}, nil
					},
				}},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 64},
			},
			wantCause: marksplice.ErrInvalidExtension,
		},
		{
			name: "node budget",
			options: marksplice.ParseOptions{
				Extensions: []marksplice.Extension{{
					ID: "example.org/nodes",
					Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
						return []marksplice.ExtensionMatch{
							{Kind: "one", Range: marksplice.Range{Start: 0, End: 1}},
							{Kind: "two", Range: marksplice.Range{Start: 1, End: 2}},
						}, nil
					},
				}},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 1, MaxMetadataBytes: 64},
			},
			wantCause: marksplice.ErrInvalidExtension,
		},
		{
			name: "metadata budget",
			options: marksplice.ParseOptions{
				Extensions: []marksplice.Extension{{
					ID: "example.org/meta",
					Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
						return []marksplice.ExtensionMatch{{
							Kind:       "node",
							Range:      marksplice.Range{Start: 0, End: 1},
							Attributes: []marksplice.ExtensionAttribute{{Name: "key", Value: "long-value"}},
						}}, nil
					},
				}},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 4},
			},
			wantCause: marksplice.ErrInvalidExtension,
		},
		{
			name: "recognizer error preserves cause",
			options: marksplice.ParseOptions{
				Extensions: []marksplice.Extension{{
					ID: "example.org/error",
					Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
						return nil, callbackErr
					},
				}},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 64},
			},
			wantCause: callbackErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := marksplice.ParseWithOptions(source, test.options)
			if !errors.Is(err, marksplice.ErrInvalidExtension) {
				t.Fatalf("ParseWithOptions() error = %v, want ErrInvalidExtension", err)
			}
			if test.wantCause != nil && test.wantCause != marksplice.ErrInvalidExtension && !errors.Is(err, test.wantCause) {
				t.Fatalf("ParseWithOptions() error = %v, want cause %v", err, test.wantCause)
			}
		})
	}
}

func TestM110ExtensionConfigurationIsPrevalidatedBeforeCallbacks(t *testing.T) {
	t.Parallel()

	calls := 0
	recognizer := func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
		calls++
		return nil, nil
	}
	invalid := []marksplice.ParseOptions{
		{
			Extensions:      []marksplice.Extension{{ID: "bad id", Recognize: recognizer}},
			ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 1, MaxMetadataBytes: 32},
		},
		{
			Extensions:      []marksplice.Extension{{ID: "example.org/nil"}},
			ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 1, MaxMetadataBytes: 32},
		},
		{
			Extensions: []marksplice.Extension{
				{ID: "example.org/dup", Recognize: recognizer},
				{ID: "example.org/dup", Recognize: recognizer},
			},
			ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 1, MaxMetadataBytes: 32},
		},
	}
	for index, options := range invalid {
		if _, err := marksplice.ParseWithOptions([]byte("text\n"), options); !errors.Is(err, marksplice.ErrInvalidExtension) {
			t.Fatalf("case %d error = %v, want ErrInvalidExtension", index, err)
		}
	}
	if calls != 0 {
		t.Fatalf("recognizers called during invalid configuration: %d", calls)
	}
}

func TestM110ExtensionOutputValidationAndPanicAreAllOrNothing(t *testing.T) {
	t.Parallel()

	panicCause := errors.New("panic cause")
	tests := []struct {
		name      string
		recognize marksplice.ExtensionRecognizer
		wantCause error
	}{
		{
			name: "invalid kind",
			recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				return []marksplice.ExtensionMatch{{Kind: "bad kind", Range: marksplice.Range{Start: 0, End: 1}}}, nil
			},
		},
		{
			name: "empty range",
			recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				return []marksplice.ExtensionMatch{{Kind: "node", Range: marksplice.Range{Start: 1, End: 1}}}, nil
			},
		},
		{
			name: "duplicate attribute",
			recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				return []marksplice.ExtensionMatch{{
					Kind:  "node",
					Range: marksplice.Range{Start: 0, End: 1},
					Attributes: []marksplice.ExtensionAttribute{
						{Name: "same", Value: "one"},
						{Name: "same", Value: "two"},
					},
				}}, nil
			},
		},
		{
			name: "invalid attribute value",
			recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				return []marksplice.ExtensionMatch{{
					Kind:       "node",
					Range:      marksplice.Range{Start: 0, End: 1},
					Attributes: []marksplice.ExtensionAttribute{{Name: "value", Value: "bad\x00value"}},
				}}, nil
			},
		},
		{
			name: "panic",
			recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				panic(panicCause)
			},
			wantCause: panicCause,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, err := marksplice.ParseWithOptions([]byte("text\n"), marksplice.ParseOptions{
				Extensions:      []marksplice.Extension{{ID: "example.org/output", Recognize: test.recognize}},
				ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 8, MaxMetadataBytes: 256},
			})
			if document != nil {
				t.Fatalf("ParseWithOptions() document = %+v, want nil on extension failure", document)
			}
			if !errors.Is(err, marksplice.ErrInvalidExtension) {
				t.Fatalf("ParseWithOptions() error = %v, want ErrInvalidExtension", err)
			}
			if test.wantCause != nil && !errors.Is(err, test.wantCause) {
				t.Fatalf("ParseWithOptions() error = %v, want cause %v", err, test.wantCause)
			}
		})
	}
}

func TestM110ParseWithZeroOptionsKeepsBaselineBehavior(t *testing.T) {
	t.Parallel()

	source := []byte("# Heading\n\nparagraph\n")
	baseline, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	withOptions, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}
	if !reflect.DeepEqual(withOptions.Nodes(), baseline.Nodes()) {
		t.Fatalf("ParseWithOptions().Nodes() = %+v, baseline = %+v", withOptions.Nodes(), baseline.Nodes())
	}
	if withOptions.ExtensionNodes() != nil {
		t.Fatalf("zero-options ExtensionNodes() = %+v, want nil", withOptions.ExtensionNodes())
	}
}

func TestM110ExtensionOrderAndOverlapRemainSeparateFromCore(t *testing.T) {
	t.Parallel()

	source := []byte("[[x]]\n")
	calls := make([]string, 0, 2)
	extension := func(id marksplice.ExtensionID, kind marksplice.ExtensionKind) marksplice.Extension {
		return marksplice.Extension{
			ID: id,
			Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				calls = append(calls, string(id))
				return []marksplice.ExtensionMatch{{Kind: kind, Range: marksplice.Range{Start: 0, End: 5}}}, nil
			},
		}
	}
	document, err := marksplice.ParseWithOptions(source, marksplice.ParseOptions{
		Extensions: []marksplice.Extension{
			extension("example.org/first", "one"),
			extension("example.org/second", "two"),
		},
		ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 128},
	})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"example.org/first", "example.org/second"}) {
		t.Fatalf("recognizer order = %v", calls)
	}
	nodes := document.ExtensionNodes()
	if len(nodes) != 2 || nodes[0].ExtensionID() != "example.org/first" || nodes[1].ExtensionID() != "example.org/second" {
		t.Fatalf("ExtensionNodes() = %+v, want registration order", nodes)
	}
	if core := document.Nodes(); len(core) != 1 || core[0].Kind() != marksplice.KindParagraph {
		t.Fatalf("core Nodes() changed by overlapping extension observations: %+v", core)
	}
}

func TestM110ExtensionNodesSupportConcurrentReads(t *testing.T) {
	t.Parallel()

	document, err := marksplice.ParseWithOptions([]byte("text\n"), marksplice.ParseOptions{
		Extensions: []marksplice.Extension{{
			ID: "example.org/concurrent",
			Recognize: func(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
				return []marksplice.ExtensionMatch{{
					Kind:       "node",
					Range:      marksplice.Range{Start: 0, End: 4},
					Attributes: []marksplice.ExtensionAttribute{{Name: "value", Value: "stable"}},
				}}, nil
			},
		}},
		ExtensionLimits: marksplice.ExtensionLimits{MaxNodes: 4, MaxMetadataBytes: 128},
	})
	if err != nil {
		t.Fatalf("ParseWithOptions() error = %v", err)
	}

	const workers = 8
	const iterations = 32
	failures := make(chan string, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				nodes := document.ExtensionNodes()
				if len(nodes) != 1 {
					failures <- "unexpected node count"
					return
				}
				attributes := nodes[0].Attributes()
				attributes[0].Value = "caller-owned"
				if value, ok := document.ExtensionNodes()[0].Attribute("value"); !ok || value != "stable" {
					failures <- "extension metadata changed"
					return
				}
			}
		}()
	}
	group.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
}

func noM110Matches(marksplice.ExtensionSource) ([]marksplice.ExtensionMatch, error) {
	return nil, nil
}
