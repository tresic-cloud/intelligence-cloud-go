package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	intelligencecloud "github.com/tresic-cloud/intelligence-cloud-go"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/confirm"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/output"
	"github.com/tresic-cloud/intelligence-cloud-go/cmd/icctl/tui"
	"github.com/tresic-cloud/intelligence-cloud-go/internal/generated"
)

// NewMeCmd creates the me subcommand tree with all sub-subcommands.
func NewMeCmd(profileLoader ProfileLoader, secretLoader SecretLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Manage your user profile",
		Long:  "View and manage your authenticated user profile, notification preferences, daily recap preferences, and avatar.",
		Example: `  # View your profile
  icctl me get

  # View notification preferences
  icctl me notification-prefs get

  # Update daily recap preferences
  icctl me daily-recap-prefs update --from-file prefs.json

  # Generate an avatar upload URL
  icctl me avatar upload-url --content-type image/jpeg`,
	}

	cmd.AddCommand(
		newMeGetCmd(),
		newMeNotificationPrefsCmd(),
		newMeDailyRecapPrefsCmd(),
		newMeAvatarCmd(),
	)

	return cmd
}

// ---------------------------------------------------------------------------
// me get
// ---------------------------------------------------------------------------

func newMeGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get your user profile",
		Long:  "Retrieve the authenticated user's profile information.",
		Example: `  icctl me get
  icctl me get --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeGet(cmd)
		},
	}
}

func runMeGet(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	profile, err := client.Me.Get(ctx)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(profile)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"ID", profile.Id},
			{"Display Name", profile.DisplayName},
			{"Email", profile.Email},
			{"First Name", profile.FirstName},
			{"Last Name", profile.LastName},
			{"Role", profile.Role},
			{"User Type", string(profile.UserType)},
			{"Status", string(profile.Status)},
		}
		return f.WriteTable(headers, rows)
	}
}

// ---------------------------------------------------------------------------
// me notification-prefs
// ---------------------------------------------------------------------------

func newMeNotificationPrefsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notification-prefs",
		Short: "Manage notification preferences",
		Long:  "View and update your notification preference settings.",
		Example: `  icctl me notification-prefs get
  icctl me notification-prefs update --from-file prefs.json`,
	}

	cmd.AddCommand(
		newMeNotificationPrefsGetCmd(),
		newMeNotificationPrefsUpdateCmd(),
	)

	return cmd
}

func newMeNotificationPrefsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get notification preferences",
		Long:  "Retrieve your current notification preference settings.",
		Example: `  icctl me notification-prefs get
  icctl me notification-prefs get --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeNotificationPrefsGet(cmd)
		},
	}
}

func runMeNotificationPrefsGet(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	prefs, err := client.Me.GetNotificationPreferences(ctx)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(prefs)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"Email Notifications", output.FormatBool(ptrBoolDef(prefs.EmailNotifications), true)},
			{"Push Notifications", output.FormatBool(ptrBoolDef(prefs.PushNotifications), true)},
			{"Daily Recap Email", output.FormatBool(ptrBoolDef(prefs.DailyRecapEmail), true)},
			{"Missed Call Alerts", output.FormatBool(ptrBoolDef(prefs.MissedCallAlerts), true)},
			{"Task Reminders", output.FormatBool(ptrBoolDef(prefs.TaskReminders), true)},
		}
		return f.WriteTable(headers, rows)
	}
}

func newMeNotificationPrefsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update notification preferences",
		Long:  "Partially update notification preferences. Only provided fields are modified.",
		Example: `  icctl me notification-prefs update --from-file prefs.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeNotificationPrefsUpdate(cmd)
		},
	}

	cmd.Flags().String("from-file", "", "Path to JSON file with preference updates")

	return cmd
}

func runMeNotificationPrefsUpdate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required for notification-prefs update")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.NotificationPreferencesPatch
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
	}

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	prefs, err := client.Me.PatchNotificationPreferences(ctx, req)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(prefs)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"Email Notifications", output.FormatBool(ptrBoolDef(prefs.EmailNotifications), true)},
			{"Push Notifications", output.FormatBool(ptrBoolDef(prefs.PushNotifications), true)},
			{"Daily Recap Email", output.FormatBool(ptrBoolDef(prefs.DailyRecapEmail), true)},
			{"Missed Call Alerts", output.FormatBool(ptrBoolDef(prefs.MissedCallAlerts), true)},
			{"Task Reminders", output.FormatBool(ptrBoolDef(prefs.TaskReminders), true)},
		}
		return f.WriteTable(headers, rows)
	}
}

// ---------------------------------------------------------------------------
// me daily-recap-prefs
// ---------------------------------------------------------------------------

func newMeDailyRecapPrefsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daily-recap-prefs",
		Short: "Manage daily recap preferences",
		Long:  "View and update your daily recap preference settings.",
		Example: `  icctl me daily-recap-prefs get
  icctl me daily-recap-prefs update --from-file prefs.json`,
	}

	cmd.AddCommand(
		newMeDailyRecapPrefsGetCmd(),
		newMeDailyRecapPrefsUpdateCmd(),
	)

	return cmd
}

func newMeDailyRecapPrefsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get daily recap preferences",
		Long:  "Retrieve your current daily recap preference settings.",
		Example: `  icctl me daily-recap-prefs get
  icctl me daily-recap-prefs get --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeDailyRecapPrefsGet(cmd)
		},
	}
}

func runMeDailyRecapPrefsGet(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	prefs, err := client.Me.GetDailyRecapPreferences(ctx)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(prefs)
	default:
		return formatDailyRecapPrefsTable(cmd, prefs)
	}
}

func newMeDailyRecapPrefsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update daily recap preferences",
		Long:  "Partially update daily recap preferences. Only provided fields are modified.",
		Example: `  icctl me daily-recap-prefs update --from-file prefs.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeDailyRecapPrefsUpdate(cmd)
		},
	}

	cmd.Flags().String("from-file", "", "Path to JSON file with preference updates")

	return cmd
}

func runMeDailyRecapPrefsUpdate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	fromFile, _ := cmd.Flags().GetString("from-file")
	if fromFile == "" {
		return fmt.Errorf("--from-file is required for daily-recap-prefs update")
	}

	data, err := os.ReadFile(fromFile)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", fromFile, err)
	}

	var req intelligencecloud.DailyRecapPreferencesPatch
	if err := json.Unmarshal(data, &req); err != nil {
		return fmt.Errorf("failed to parse JSON from %q: %w", fromFile, err)
	}

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	prefs, err := client.Me.PatchDailyRecapPreferences(ctx, req)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(prefs)
	default:
		return formatDailyRecapPrefsTable(cmd, prefs)
	}
}

// ---------------------------------------------------------------------------
// me avatar
// ---------------------------------------------------------------------------

func newMeAvatarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "avatar",
		Short: "Manage your avatar",
		Long:  "Upload, confirm, or delete your profile avatar.",
		Example: `  icctl me avatar upload-url --content-type image/jpeg
  icctl me avatar confirm --asset-url https://cdn.example.com/avatar.jpg
  icctl me avatar delete`,
	}

	cmd.AddCommand(
		newMeAvatarUploadURLCmd(),
		newMeAvatarConfirmCmd(),
		newMeAvatarDeleteCmd(),
	)

	return cmd
}

func newMeAvatarUploadURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload-url",
		Short: "Generate an avatar upload URL",
		Long:  "Generate a pre-signed upload URL for a profile photo. After uploading the image, use 'avatar confirm' to store the asset.",
		Example: `  icctl me avatar upload-url --content-type image/jpeg
  icctl me avatar upload-url --content-type image/png --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeAvatarUploadURL(cmd)
		},
	}

	cmd.Flags().String("content-type", "", "MIME type of the image (image/jpeg or image/png)")

	return cmd
}

func runMeAvatarUploadURL(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	contentType, _ := cmd.Flags().GetString("content-type")
	if contentType == "" {
		return fmt.Errorf("--content-type is required (image/jpeg or image/png)")
	}

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	req := intelligencecloud.AvatarUploadURLRequest{
		ContentType: generated.AvatarUploadURLRequestContentType(contentType),
	}

	result, err := client.Me.CreateAvatarUploadURL(ctx, req)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(result)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"Upload URL", result.UploadUrl},
			{"Asset URL", result.AssetUrl},
			{"Expires At", result.ExpiresAt.Format("2006-01-02 15:04:05")},
		}
		return f.WriteTable(headers, rows)
	}
}

func newMeAvatarConfirmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm an avatar upload",
		Long:  "Confirm a previously generated avatar upload by storing the asset URL on the user profile.",
		Example: `  icctl me avatar confirm --asset-url https://cdn.example.com/avatar.jpg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeAvatarConfirm(cmd)
		},
	}

	cmd.Flags().String("asset-url", "", "The asset_url returned from the upload-url endpoint")

	return cmd
}

func runMeAvatarConfirm(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	assetURL, _ := cmd.Flags().GetString("asset-url")
	if assetURL == "" {
		return fmt.Errorf("--asset-url is required")
	}

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	req := intelligencecloud.AvatarConfirmRequest{
		AssetUrl: assetURL,
	}

	result, err := client.Me.ConfirmAvatar(ctx, req)
	if err != nil {
		return err
	}

	switch state.EffectiveOutput {
	case "json":
		f := output.NewJSONFormatter(cmd.OutOrStdout())
		return f.WriteEntity(result)
	default:
		f := output.NewTableFormatter(cmd.OutOrStdout(), true)
		headers := []string{"FIELD", "VALUE"}
		rows := [][]string{
			{"Avatar URL", result.AvatarUrl},
		}
		return f.WriteTable(headers, rows)
	}
}

func newMeAvatarDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete your avatar",
		Long:  "Remove your profile photo. This is a destructive operation that requires confirmation.",
		Example: `  icctl me avatar delete
  icctl me avatar delete --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMeAvatarDelete(cmd)
		},
	}

	cmd.Flags().Bool("force-tty", false, "Force TTY mode for testing")
	_ = cmd.Flags().MarkHidden("force-tty")

	return cmd
}

func runMeAvatarDelete(cmd *cobra.Command) error {
	ctx := cmd.Context()
	state := StateFrom(ctx)

	preview := confirm.DestructivePreview{
		Environment: confirm.EnvironmentInfo{
			Name:    state.EffectiveEnvironment,
			BaseURL: state.EffectiveBaseURL,
		},
		Operation:    "Delete avatar",
		ResourceType: "avatar",
		ResourceID:   "me",
		Irreversible: true,
	}

	// Emit preview based on output format.
	switch state.EffectiveOutput {
	case "json":
		if err := confirm.RenderJSONPreview(cmd.OutOrStdout(), preview); err != nil {
			return err
		}
	default:
		if err := confirm.RenderTablePreview(cmd.ErrOrStderr(), preview); err != nil {
			return err
		}
	}

	forceTTY, _ := cmd.Flags().GetBool("force-tty")
	isTTY := forceTTY

	if err := confirm.Prompt(confirm.PromptOptions{
		AssumeYes: state.AssumeYes,
		IsTTY:     isTTY,
		Stdin:     cmd.InOrStdin(),
		Stderr:    cmd.ErrOrStderr(),
	}); err != nil {
		return err
	}

	client, err := buildMeClient(state)
	if err != nil {
		return err
	}

	if err := client.Me.DeleteAvatar(ctx); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Avatar deleted.")
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildMeClient constructs an SDK client from the CLI state.
func buildMeClient(state *State) (*intelligencecloud.Client, error) {
	if state == nil {
		return nil, fmt.Errorf("CLI state not available")
	}
	return tui.BuildClient(state.EffectiveBaseURL, state.EffectiveToken, state.LogLevel)
}

// ptrBoolDef safely dereferences a *bool, returning false if nil.
func ptrBoolDef(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// formatDailyRecapPrefsTable formats daily recap preferences as a table.
func formatDailyRecapPrefsTable(cmd *cobra.Command, prefs *intelligencecloud.DailyRecapPreferences) error {
	f := output.NewTableFormatter(cmd.OutOrStdout(), true)
	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Timezone", ptrStrDef(prefs.Timezone)},
		{"Send Time", ptrStrDef(prefs.SendTime)},
		{"Date Format", stringFromDateFormat(prefs.DateFormat)},
		{"Time Format", stringFromTimeFormat(prefs.TimeFormat)},
	}
	return f.WriteTable(headers, rows)
}

// ptrStrDef safely dereferences a *string, returning "" if nil.
func ptrStrDef(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringFromDateFormat converts a *DailyRecapPreferencesDateFormat to a string.
func stringFromDateFormat(f *generated.DailyRecapPreferencesDateFormat) string {
	if f == nil {
		return ""
	}
	return string(*f)
}

// stringFromTimeFormat converts a *DailyRecapPreferencesTimeFormat to a string.
func stringFromTimeFormat(f *generated.DailyRecapPreferencesTimeFormat) string {
	if f == nil {
		return ""
	}
	return string(*f)
}
