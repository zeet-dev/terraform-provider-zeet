package customtypes

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/pkg/errors"
)

type UUIDType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = &UUIDType{}
var _ xattr.TypeWithValidate = UUIDType{}

func (t UUIDType) String() string {
	return "UUIDType"
}

func (t UUIDType) ValueType(context.Context) attr.Value {
	return UUIDValue{}
}

func (t UUIDType) Equal(o attr.Type) bool {
	other, ok := o.(UUIDType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t UUIDType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return UUIDValue{
		StringValue: in,
	}, nil
}

func (t UUIDType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	uuidValue, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to UUIDValue: %v", diags)
	}

	return uuidValue, nil
}

// Validate checks that the value is a valid UUID, if it is known and not null.
func (t UUIDType) Validate(ctx context.Context, value tftypes.Value, valuePath path.Path) diag.Diagnostics {
	if value.IsNull() || !value.IsKnown() {
		return nil
	}

	var diags diag.Diagnostics
	var valueString string
	if err := value.As(&valueString); err != nil {
		diags.AddAttributeError(
			valuePath,
			"expected a string",
			err.Error(),
		)
		return diags
	}

	if _, err := uuid.Parse(valueString); err != nil {
		diags.AddAttributeError(
			valuePath,
			"expected a valid UUID",
			err.Error(),
		)
		return diags
	}

	return diags
}

func NewUUIDValue(value uuid.UUID) UUIDValue {
	return UUIDValue{
		StringValue: basetypes.NewStringValue(value.String()),
	}
}

// UUIDValue is a custom value used to validate that a string is a UUID.
type UUIDValue struct {
	basetypes.StringValue
}

var _ basetypes.StringValuableWithSemanticEquals = UUIDValue{}

func (v UUIDValue) Type(context.Context) attr.Type {
	return UUIDType{}
}

func (v UUIDValue) ValueUUID() uuid.UUID {
	if v.IsNull() || v.IsUnknown() {
		return uuid.Nil
	}

	u, _ := uuid.Parse(v.ValueString())
	return u
}

func (v UUIDValue) Equal(o attr.Value) bool {
	other, ok := o.(UUIDValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v UUIDValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	newValue, ok := newValuable.(UUIDValue)
	if !ok {
		diags.AddError("expected new value to be a UUIDValue", "")
		return false, diags
	}

	oldUUID, err := uuid.Parse(v.ValueString())
	if err != nil {
		diags.AddError("expected old value to be a valid UUID", err.Error())
	}
	newUUID, err := uuid.Parse(newValue.ValueString())
	if err != nil {
		diags.AddError("expected new value to be a valid UUID", err.Error())
	}

	if diags.HasError() {
		return false, diags
	}

	return reflect.DeepEqual(oldUUID, newUUID), diags
}
