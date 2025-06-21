package ir

// Enums for type safety

type TableType string

const (
	TableTypeBase TableType = "BASE_TABLE"
	TableTypeView TableType = "VIEW"
	TableTypeTemp TableType = "TEMPORARY"
)

type ConstraintType string

const (
	ConstraintTypePrimaryKey ConstraintType = "PRIMARY_KEY"
	ConstraintTypeUnique     ConstraintType = "UNIQUE"
	ConstraintTypeForeignKey ConstraintType = "FOREIGN_KEY"
	ConstraintTypeCheck      ConstraintType = "CHECK"
	ConstraintTypeExclusion  ConstraintType = "EXCLUSION"
)

type IndexType string

const (
	IndexTypeRegular    IndexType = "REGULAR"
	IndexTypePrimary    IndexType = "PRIMARY"
	IndexTypeUnique     IndexType = "UNIQUE"
	IndexTypePartial    IndexType = "PARTIAL"
	IndexTypeExpression IndexType = "EXPRESSION"
)

type TriggerTiming string

const (
	TriggerTimingBefore    TriggerTiming = "BEFORE"
	TriggerTimingAfter     TriggerTiming = "AFTER"
	TriggerTimingInsteadOf TriggerTiming = "INSTEAD_OF"
)

type TriggerEvent string

const (
	TriggerEventInsert   TriggerEvent = "INSERT"
	TriggerEventUpdate   TriggerEvent = "UPDATE"
	TriggerEventDelete   TriggerEvent = "DELETE"
	TriggerEventTruncate TriggerEvent = "TRUNCATE"
)

type TriggerLevel string

const (
	TriggerLevelRow       TriggerLevel = "ROW"
	TriggerLevelStatement TriggerLevel = "STATEMENT"
)

type PolicyCommand string

const (
	PolicyCommandAll    PolicyCommand = "ALL"
	PolicyCommandSelect PolicyCommand = "SELECT"
	PolicyCommandInsert PolicyCommand = "INSERT"
	PolicyCommandUpdate PolicyCommand = "UPDATE"
	PolicyCommandDelete PolicyCommand = "DELETE"
)

type DependencyType string

const (
	DependencyTypeTable    DependencyType = "TABLE"
	DependencyTypeView     DependencyType = "VIEW"
	DependencyTypeFunction DependencyType = "FUNCTION"
	DependencyTypeSequence DependencyType = "SEQUENCE"
)

// TableDependency represents a dependency between database objects
type TableDependency struct {
	Schema string         `json:"schema"`
	Name   string         `json:"name"`
	Type   DependencyType `json:"type"`
}

// PartitionAttachment represents a partition child table attachment
type PartitionAttachment struct {
	ParentSchema   string `json:"parent_schema"`
	ParentTable    string `json:"parent_table"`
	ChildSchema    string `json:"child_schema"`
	ChildTable     string `json:"child_table"`
	PartitionBound string `json:"partition_bound"`
}

// IndexAttachment represents an index attachment for partitions
type IndexAttachment struct {
	ParentSchema string `json:"parent_schema"`
	ParentIndex  string `json:"parent_index"`
	ChildSchema  string `json:"child_schema"`
	ChildIndex   string `json:"child_index"`
}