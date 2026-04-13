package intelligencecloud

import (
	"context"

	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// Type aliases re-export generated types at the package root so callers
// never import internal/generated directly.

// UserProfile is the authenticated user's profile returned by MeService.Get.
type UserProfile = generated.UserProfile

// AvatarUploadURLRequest is the request body for MeService.CreateAvatarUploadURL.
type AvatarUploadURLRequest = generated.AvatarUploadURLRequest

// AvatarUploadURLResponseData is the response from MeService.CreateAvatarUploadURL.
type AvatarUploadURLResponseData = generated.AvatarUploadURLResponseData

// AvatarConfirmRequest is the request body for MeService.ConfirmAvatar.
type AvatarConfirmRequest = generated.AvatarConfirmRequest

// AvatarResponseData is the response from MeService.ConfirmAvatar.
type AvatarResponseData = generated.AvatarResponseData

// NotificationPreferences holds the user's notification preference settings.
type NotificationPreferences = generated.NotificationPreferences

// DailyRecapPreferences holds the user's daily recap preference settings.
type DailyRecapPreferences = generated.DailyRecapPreferences

// NotificationPreferencesPatch is the request body for
// MeService.PatchNotificationPreferences (all fields optional).
type NotificationPreferencesPatch = generated.PatchNotificationPreferencesRequest

// DailyRecapPreferencesPatch is the request body for
// MeService.PatchDailyRecapPreferences (all fields optional).
type DailyRecapPreferencesPatch = generated.PatchDailyRecapPreferencesRequest

// Get retrieves the authenticated user's profile.
func (s *MeService) Get(ctx context.Context, opts ...CallOption) (*UserProfile, error) {
	const op = "GetMe"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetMe(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.UserProfileResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// CreateAvatarUploadURL generates a pre-signed upload URL for a profile
// photo. After uploading the image to the returned URL, call ConfirmAvatar
// to store the asset on the profile.
func (s *MeService) CreateAvatarUploadURL(ctx context.Context, req AvatarUploadURLRequest, opts ...CallOption) (*AvatarUploadURLResponseData, error) {
	const op = "CreateAvatarUploadURL"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.CreateAvatarUploadURL(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.AvatarUploadURLResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// ConfirmAvatar confirms a previously generated avatar upload by storing
// the asset URL on the user profile.
func (s *MeService) ConfirmAvatar(ctx context.Context, req AvatarConfirmRequest, opts ...CallOption) (*AvatarResponseData, error) {
	const op = "ConfirmAvatar"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.ConfirmAvatar(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.AvatarResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// DeleteAvatar removes the authenticated user's profile photo.
func (s *MeService) DeleteAvatar(ctx context.Context, opts ...CallOption) error {
	const op = "DeleteAvatar"

	gc, err := s.client.genClient()
	if err != nil {
		return err
	}

	resp, err := gc.DeleteAvatar(ctx)
	if err != nil {
		return err
	}

	return handleResponse(resp, op, nil)
}

// GetNotificationPreferences retrieves the authenticated user's
// notification preference settings.
func (s *MeService) GetNotificationPreferences(ctx context.Context, opts ...CallOption) (*NotificationPreferences, error) {
	const op = "GetNotificationPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetNotificationPreferences(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.NotificationPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// PatchNotificationPreferences partially updates the authenticated user's
// notification preferences. Only the fields present in req are modified.
func (s *MeService) PatchNotificationPreferences(ctx context.Context, req NotificationPreferencesPatch, opts ...CallOption) (*NotificationPreferences, error) {
	const op = "PatchNotificationPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.PatchNotificationPreferences(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.NotificationPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// GetDailyRecapPreferences retrieves the authenticated user's daily recap
// preference settings.
func (s *MeService) GetDailyRecapPreferences(ctx context.Context, opts ...CallOption) (*DailyRecapPreferences, error) {
	const op = "GetDailyRecapPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.GetDailyRecapPreferences(ctx)
	if err != nil {
		return nil, err
	}

	var envelope generated.DailyRecapPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}

// PatchDailyRecapPreferences partially updates the authenticated user's
// daily recap preferences. Only the fields present in req are modified.
func (s *MeService) PatchDailyRecapPreferences(ctx context.Context, req DailyRecapPreferencesPatch, opts ...CallOption) (*DailyRecapPreferences, error) {
	const op = "PatchDailyRecapPreferences"

	gc, err := s.client.genClient()
	if err != nil {
		return nil, err
	}

	resp, err := gc.PatchDailyRecapPreferences(ctx, req)
	if err != nil {
		return nil, err
	}

	var envelope generated.DailyRecapPreferencesResponse
	if err := handleResponse(resp, op, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Data, nil
}
