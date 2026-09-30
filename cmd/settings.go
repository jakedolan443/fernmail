package main

import (
	"encoding/json"

	"strings"

	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/httputil"
	"github.com/jakedolan443/fernmail/internal/setting/models"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetGeneralSettings fetches general settings, this endpoint is not behind auth as it has no sensitive data and is required for the app to function.
func handleGetGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	out, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Unmarshal to set the app.update to the settings, so the frontend can show that an update is available.
	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		app.lo.Error("error unmarshalling settings", "err", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Set the app.update to the settings, adding `app` prefix to the key to match the settings structure in db.
	settings["app.update"] = app.update
	// Set app version.
	settings["app.version"] = versionString
	// Set restart required flag.
	settings["app.restart_required"] = app.restartRequired
	return r.SendEnvelope(settings)
}

// handleUpdateGeneralSettings updates general settings.
func handleUpdateGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.General{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Trim whitespace from string fields.
	req.SiteName = strings.TrimSpace(req.SiteName)
	req.FaviconURL = strings.TrimSpace(req.FaviconURL)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.Timezone = strings.TrimSpace(req.Timezone)
	if req.Timezone != "" && !stringutil.IsValidTimezone(req.Timezone) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid timezone.", nil, envelope.InputError)
	}
	// Trim whitespace and trailing slash from root URL.
	req.RootURL = strings.TrimRight(strings.TrimSpace(req.RootURL), "/")
	if !httputil.IsValidHTTPURL(req.RootURL) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("admin.general.rootURL.valid"), nil, envelope.InputError)
	}

	// Get current language before update.
	app.Lock()
	oldLang := ko.String("app.lang")
	app.Unlock()

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reload the settings and templates.
	if err := reloadSettings(app); err != nil {
		app.lo.Error("error reloading settings", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	// Check if language changed and reload i18n if needed.
	app.Lock()
	newLang := ko.String("app.lang")
	if oldLang != newLang {
		app.lo.Info("language changed, reloading i18n", "old_lang", oldLang, "new_lang", newLang)
		app.i18n = initI18n(app.fs)
		app.lo.Info("reloaded i18n", "old_lang", oldLang, "new_lang", newLang)
	}
	app.Unlock()

	if err := reloadTemplates(app); err != nil {
		app.lo.Error("error reloading templates", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(true)
}
