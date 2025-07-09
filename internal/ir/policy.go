package ir


// RLSPolicy represents a Row Level Security policy
type RLSPolicy struct {
	Schema     string        `json:"schema"`
	Table      string        `json:"table"`
	Name       string        `json:"name"`
	Command    PolicyCommand `json:"command"` // SELECT, INSERT, UPDATE, DELETE, ALL
	Permissive bool          `json:"permissive"`
	Roles      []string      `json:"roles,omitempty"`
	Using      string        `json:"using,omitempty"`      // USING expression
	WithCheck  string        `json:"with_check,omitempty"` // WITH CHECK expression
	Comment    string        `json:"comment,omitempty"`
}

// PolicyCommand represents the command for which the policy applies
type PolicyCommand string

const (
	PolicyCommandAll    PolicyCommand = "ALL"
	PolicyCommandSelect PolicyCommand = "SELECT"
	PolicyCommandInsert PolicyCommand = "INSERT"
	PolicyCommandUpdate PolicyCommand = "UPDATE"
	PolicyCommandDelete PolicyCommand = "DELETE"
)

