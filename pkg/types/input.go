package types

// InputType defines the type of input being sent to the agent.
type InputType string

const (
	InputTypeCancel       InputType = "cancel"        // InputTypeCancel indicates a cancellation request.
	InputTypeUserInput    InputType = "user_input"    // InputTypeUserInput indicates a simple text input from the user.
	InputTypeFormInput    InputType = "form_input"    // InputTypeFormInput indicates structured form data with multiple key-value pairs.
	InputTypeNotesRequest InputType = "notes_request" // InputTypeNotesRequest indicates a request for notes data.
)

// Input represents various types of input that can be sent to an agent.
type Input struct {
	// Metadata holds optional additional information about the input.
	Metadata map[string]interface{}

	// FormData holds structured key-value pairs for form-based input.
	// Only populated when Type is InputTypeFormInput.
	FormData map[string]string

	// Content is the text content for user input.
	// Only populated when Type is InputTypeUserInput.
	Content string

	// Type indicates the kind of input (cancel, user_input, form_input).
	Type InputType
}

// NewCancelInput creates a new cancellation input.
func NewCancelInput() *Input {
	return &Input{
		Type:     InputTypeCancel,
		Metadata: make(map[string]interface{}),
	}
}

// NewUserInput creates a new user text input.
func NewUserInput(content string) *Input {
	return &Input{
		Type:     InputTypeUserInput,
		Content:  content,
		Metadata: make(map[string]interface{}),
	}
}

// NewFormInput creates a new form input with the given data.
func NewFormInput(formData map[string]string) *Input {
	return &Input{
		Type:     InputTypeFormInput,
		FormData: formData,
		Metadata: make(map[string]interface{}),
	}
}

// WithMetadata adds metadata to the input and returns the input for chaining.
func (i *Input) WithMetadata(key string, value interface{}) *Input {
	if i.Metadata == nil {
		i.Metadata = make(map[string]interface{})
	}
	i.Metadata[key] = value
	return i
}

// IsCancel returns true if this is a cancellation input.
func (i *Input) IsCancel() bool {
	return i.Type == InputTypeCancel
}

// IsUserInput returns true if this is a user text input.
func (i *Input) IsUserInput() bool {
	return i.Type == InputTypeUserInput
}

// IsFormInput returns true if this is a form input.
func (i *Input) IsFormInput() bool {
	return i.Type == InputTypeFormInput
}

// IsNotesRequest returns true if this is a notes request input.
func (i *Input) IsNotesRequest() bool {
	return i.Type == InputTypeNotesRequest
}

// NotesRequestParams contains parameters for requesting notes data.
type NotesRequestParams struct {
	Tag              string // Optional tag filter
	IncludeScratched bool   // Include scratched notes
	Limit            int    // Max notes to return (default 10)
}

// NewNotesRequestInput creates a new notes request input.
func NewNotesRequestInput(params NotesRequestParams) *Input {
	// Set default limit if not specified
	if params.Limit == 0 {
		params.Limit = 10
	}

	return &Input{
		Type:     InputTypeNotesRequest,
		Metadata: map[string]interface{}{"params": params},
	}
}
