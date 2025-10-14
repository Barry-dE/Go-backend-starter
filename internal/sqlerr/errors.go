package sqlerr

type Code string

const (
	Other Code = "other"

	DeadlockDetected Code = "deadlock_detected"

	NotNullViolation Code = "not_null_violation"

	TransactionFailed Code = "transaction_failed"

	CheckViolation Code = "check_violation"

	UniqueViolation Code = "unique_violation"

	ExcludeViolation Code = "exclude_violation"

	ForeignKeyViolation Code = "foreign_key_violation "

	TooManyConnections Code = "too_many_connections"
)

func MapDatabaseErrorCode(code string) Code {
	switch code {
	case "40P01":
		return DeadlockDetected
	case "23502":
		return NotNullViolation
	case "52P02":
		return TransactionFailed
	case "23514":
		return CheckViolation
	case "23505":
		return UniqueViolation
	case "23P01":
		return ExcludeViolation
	case "23503":
		return ForeignKeyViolation
	case "53300":
		return TooManyConnections
	default:
		return Other

	}

}

// Define the severity of the database error
type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityLog     Severity = "LOG"
	SeverityFatal   Severity = "FATAL"
	SeverityPanic   Severity = "PANIC"
	SeverityDebug   Severity = "DEBUG"
	SeverityInfo    Severity = "INFO"
	SeverityWarning Severity = "WARNING"
	SeverityNotice  Severity = "NOTICE"
)

func MapDatabaseSeverity(severity string) Severity {
	switch severity {
	case "ERROR":
		return SeverityError
	case "LOG":
		return SeverityLog
	case "FATAL":
		return SeverityFatal
	case "PANIC":
		return SeverityPanic
	case "DEBUG":
		return SeverityDebug
	case "INFO":
		return SeverityInfo
	case "WARNING":
		return SeverityWarning
	case "NOTICE":
		return SeverityWarning
	default:
		return SeverityError

	}
}

// Error represents a structured database error
type Error struct {
	Code           Code
	Severity       Severity
	DatabaseCode   string
	Message        string
	SchemaName     string
	TableName      string
	ColumnName     string
	DataTypeName   string
	ConstraintName string
	driverErr      error
}

func (pe *Error) Error() string {
	return string(pe.Severity) + ": " + pe.Message + " (Code " + string(pe.Code) + ": SQLSTATE " + pe.DatabaseCode + ")"
}

func (pe *Error) Unwrap() error {
	return pe.driverErr
}
