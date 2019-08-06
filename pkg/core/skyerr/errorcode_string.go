// Code generated by "stringer -type=ErrorCode"; DO NOT EDIT.

package skyerr

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NotAuthenticated-101]
	_ = x[PermissionDenied-102]
	_ = x[AccessKeyNotAccepted-103]
	_ = x[AccessTokenNotAccepted-104]
	_ = x[InvalidCredentials-105]
	_ = x[InvalidSignature-106]
	_ = x[BadRequest-107]
	_ = x[InvalidArgument-108]
	_ = x[Duplicated-109]
	_ = x[ResourceNotFound-110]
	_ = x[NotSupported-111]
	_ = x[NotImplemented-112]
	_ = x[ConstraintViolated-113]
	_ = x[IncompatibleSchema-114]
	_ = x[AtomicOperationFailure-115]
	_ = x[PartialOperationFailure-116]
	_ = x[UndefinedOperation-117]
	_ = x[PluginUnavailable-118]
	_ = x[PluginTimeout-119]
	_ = x[RecordQueryInvalid-120]
	_ = x[PluginInitializing-121]
	_ = x[ResponseTimeout-122]
	_ = x[DeniedArgument-123]
	_ = x[RecordQueryDenied-124]
	_ = x[NotConfigured-125]
	_ = x[PasswordPolicyViolated-126]
	_ = x[UserDisabled-127]
	_ = x[VerificationRequired-128]
	_ = x[AssetSizeTooLarge-129]
	_ = x[WebHookTimeOut-130]
	_ = x[WebHookFailed-131]
	_ = x[CurrentIdentityBeingDeleted-132]
	_ = x[UnexpectedError-10000]
	_ = x[UnexpectedAuthInfoNotFound-10001]
	_ = x[UnexpectedUnableToOpenDatabase-10002]
	_ = x[UnexpectedPushNotificationNotConfigured-10003]
	_ = x[InternalQueryInvalid-10004]
	_ = x[UnexpectedUserNotFound-10005]
}

const (
	_ErrorCode_name_0 = "NotAuthenticatedPermissionDeniedAccessKeyNotAcceptedAccessTokenNotAcceptedInvalidCredentialsInvalidSignatureBadRequestInvalidArgumentDuplicatedResourceNotFoundNotSupportedNotImplementedConstraintViolatedIncompatibleSchemaAtomicOperationFailurePartialOperationFailureUndefinedOperationPluginUnavailablePluginTimeoutRecordQueryInvalidPluginInitializingResponseTimeoutDeniedArgumentRecordQueryDeniedNotConfiguredPasswordPolicyViolatedUserDisabledVerificationRequiredAssetSizeTooLargeWebHookTimeOutWebHookFailedCurrentIdentityBeingDeleted"
	_ErrorCode_name_1 = "UnexpectedErrorUnexpectedAuthInfoNotFoundUnexpectedUnableToOpenDatabaseUnexpectedPushNotificationNotConfiguredInternalQueryInvalidUnexpectedUserNotFound"
)

var (
	_ErrorCode_index_0 = [...]uint16{0, 16, 32, 52, 74, 92, 108, 118, 133, 143, 159, 171, 185, 203, 221, 243, 266, 284, 301, 314, 332, 350, 365, 379, 396, 409, 431, 443, 463, 480, 494, 507, 534}
	_ErrorCode_index_1 = [...]uint8{0, 15, 41, 71, 110, 130, 152}
)

func (i ErrorCode) String() string {
	switch {
	case 101 <= i && i <= 132:
		i -= 101
		return _ErrorCode_name_0[_ErrorCode_index_0[i]:_ErrorCode_index_0[i+1]]
	case 10000 <= i && i <= 10005:
		i -= 10000
		return _ErrorCode_name_1[_ErrorCode_index_1[i]:_ErrorCode_index_1[i+1]]
	default:
		return "ErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
