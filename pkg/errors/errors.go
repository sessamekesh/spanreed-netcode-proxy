package errors

import "fmt"

type Underflow struct {
	MessageName string
	MsgSize     int
	MinimumSize int
}

func (e *Underflow) Error() string {
	return fmt.Sprintf("Message parsing underflowed (type=%s), provided %d bytes, needed at least %d", e.MessageName, e.MsgSize, e.MinimumSize)
}

type InvalidEnumValue struct {
	EnumName string
	IntValue uint8
}

func (e *InvalidEnumValue) Error() string {
	return fmt.Sprintf("Invalid enum value=%d (enum: %s)", e.IntValue, e.EnumName)
}

type InvalidHeaderVersion struct {
	ExpectedMagicNumber uint32
	ActualMagicNumber   uint32
	ExpectedVersion     uint8
	ActualVersion       uint8
}

func (e *InvalidHeaderVersion) Error() string {
	return fmt.Sprintf("Invalid header: expected MagicNumber=%d, got MagicNumber=%d. Expected version %d, got %d", e.ExpectedMagicNumber, e.ActualMagicNumber, e.ExpectedVersion, e.ActualVersion)
}

type MissingFieldError struct {
	MessageName string
	FieldName   string
}

func (e *MissingFieldError) Error() string {
	return fmt.Sprintf("Missing field %s in message type %s", e.FieldName, e.MessageName)
}

type NameCollision struct {
	CollisionContext string
	Name             string
}

func (e *NameCollision) Error() string {
	return fmt.Sprintf("Name collision for name '%s' in context '%s'", e.Name, e.CollisionContext)
}
