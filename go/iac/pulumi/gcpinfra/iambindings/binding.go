package iambindings

import "errors"

var (
	// ErrNameRequired means the binding has no unique identifier.
	ErrNameRequired = errors.New("iambindings: Name is required")
	// ErrProjectIDRequired means the binding has no target project.
	ErrProjectIDRequired = errors.New("iambindings: ProjectID is required")
	// ErrRoleRequired means the binding has no IAM role.
	ErrRoleRequired = errors.New("iambindings: Role is required")
	// ErrMemberRequired means the binding has no IAM member.
	ErrMemberRequired = errors.New("iambindings: Member is required")
)

// Binding grants a member a role on a project.
type Binding struct {
	// Name is a unique identifier for this binding.
	Name string `json:"name"`
	// ProjectID is the target project.
	ProjectID string `json:"projectId"`
	// Role is the IAM role (e.g. "roles/secretmanager.secretAccessor").
	Role string `json:"role"`
	// Member is the IAM member (e.g. "serviceAccount:foo@bar.iam.gserviceaccount.com").
	Member string `json:"member"`
}

// Validate checks that the binding is complete.
func (b Binding) Validate() error {
	if b.Name == "" {
		return ErrNameRequired
	}
	if b.ProjectID == "" {
		return ErrProjectIDRequired
	}
	if b.Role == "" {
		return ErrRoleRequired
	}
	if b.Member == "" {
		return ErrMemberRequired
	}

	return nil
}
