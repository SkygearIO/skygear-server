package template

import (
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/template"
)

const (
	TemplateItemTypeAuthUIAuthorizeHTML config.TemplateItemType = "auth_ui_authorize.html"
	// nolint
	TemplateItemTypeAuthUIEnterPasswordHTML config.TemplateItemType = "auth_ui_enter_password.html"
)

// TODO(authui): Introduce a build pipeline to upload asset.

const defineHead = `
{{ define "HEAD" }}
<head>
<title>{{ .appname }}</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="{{ .static_asset_url_prefix }}/css/authui.css">
{{ if .css }}
<style>
{{ .css }}
</style>
{{ end }}
</head>
{{ end }}
`

const defineHidden = `
{{ define "HIDDEN" }}
<input type="hidden" name="scope" value="{{ .scope }}">
<input type="hidden" name="response_type" value="{{ .response_type }}">
<input type="hidden" name="client_id" value="{{ .client_id }}">
<input type="hidden" name="redirect_uri" value="{{ .redirect_uri }}">
<input type="hidden" name="code_challenge_method" value="{{ .code_challenge_method }}">
<input type="hidden" name="code_challenge" value="{{ .code_challenge }}">
<input type="hidden" name="x_login_id_input_type" value="{{ .x_login_id_input_type }}">
{{ end }}
`

const defineLogo = `
{{ define "LOGO" }}
{{ if .logo_url }}
<div class="logo" style="background-image: url('{{ .logo_url }}')"></div>
{{ else }}
<div class="logo"></div>
{{ end }}
{{ end }}
`

const defineError = `
{{ define "ERROR" }}
{{ if .error }}{{ if eq .error.reason "ValidationFailed" }}
<ul class="errors">
{{ range .error.info.causes }}
<li class="error-txt">{{ .message }}</li>
{{ end }}
</ul>
{{ else }}
<ul>
<li class="error-txt">{{ .error.message }}</li>
</ul>
{{ end }}{{ end }}
{{ end }}
`

const defineSkygearLogo = `
{{ define "SKYGEAR_LOGO" }}
<div class="skygear-logo" style="background-image: url('{{ .skygear_logo_url }}')"></div>
{{ end }}
`

var defines = []string{
	defineHead,
	defineHidden,
	defineLogo,
	defineError,
	defineSkygearLogo,
}

var TemplateAuthUIAuthorizeHTML = template.Spec{
	Type:    TemplateItemTypeAuthUIAuthorizeHTML,
	IsHTML:  true,
	Defines: defines,
	Default: `<!DOCTYPE html>
<html>
{{ template "HEAD" . }}
<body class="page">
	<div class="content">
		{{ template "LOGO" . }}
		<form class="authorize-loginid-form" method="post">
			{{ template "HIDDEN" . }}

			<a class="btn sso-btn apple">Sign in with Apple</a>
			<a class="btn sso-btn google">Sign in with Google</a>
			<a class="btn sso-btn facebook">Sign in with Facebook</a>
			<a class="btn sso-btn linkedin">Sign in with Linkedin</a>
			<a class="btn sso-btn azuread">Sign in with Azure AD</a>

			<div class="primary-txt sso-loginid-separator">or</div>

			{{ template "ERROR" . }}

			{{ if .x_login_id_input_type }}{{ if and (eq .x_login_id_input_type "phone") .x_login_id_input_type_has_phone }}
			<div class="phone-input">
				<select class="input select" name="x_calling_code">
					<option value="">Code</option>
					{{ range .x_calling_codes }}
					<option
						value="{{ . }}"
						{{ if $.x_calling_code }}{{ if eq $.x_calling_code . }}
						selected
						{{ end }}{{ end }}
						>
						+{{ . }}
					</option>
					{{ end }}
				</select>
				<input class="input text-input" type="tel" name="x_national_number" placeholder="Phone number" value="{{ .x_national_number }}">
			</div>
			{{ end }}{{ end }}

			{{ if .x_login_id_input_type }}{{ if and (not (eq .x_login_id_input_type "phone")) .x_login_id_input_type_has_text }}
			<input class="input text-input" type="text" name="x_login_id" placeholder="Email or Username" value="{{ .x_login_id }}">
			{{ end }}{{ end }}

			{{ if .x_login_id_input_type }}{{ if and (eq .x_login_id_input_type "phone") .x_login_id_input_type_has_text }}
			<a class="link anchor" href="{{ .x_use_text_url }}">Use an email or username instead</a>
			{{ end }}{{ end }}
			{{ if .x_login_id_input_type }}{{ if and (not (eq .x_login_id_input_type "phone")) .x_login_id_input_type_has_phone }}
			<a class="link anchor" href="{{ .x_use_phone_url }}">Use a phone number instead</a>
			{{ end }}{{ end }}

			<div class="link"><span class="primary-text">Don't have an account yet? </span><a class="anchor" href="#">Create one!</a></div>
			<a class="link anchor" href="#">Can't access your account?</a>

			{{ if or .x_login_id_input_type_has_phone .x_login_id_input_type_has_text }}
			<button class="btn primary-btn" type="submit" name="x_step" value="submit_login_id">Next</button>
			{{ end }}
		</form>
		{{ template "SKYGEAR_LOGO" . }}
	</div>
</body>
</html>
`,
}

var TemplateAuthUIEnterPasswordHTML = template.Spec{
	Type:    TemplateItemTypeAuthUIEnterPasswordHTML,
	IsHTML:  true,
	Defines: defines,
	Default: `<!DOCTYPE html>
<html>
{{ template "HEAD" . }}
<body class="page">
<div class="content">

{{ template "LOGO" . }}

<form class="enter-password-form" method="post">

{{ template "HIDDEN" . }}

<div class="nav-bar">
	<div class="back-button"></div>
	<div class="login-id primary-txt">
	{{ if .x_calling_code }}
		+{{ .x_calling_code}} {{ .x_national_number }}
	{{ else }}
		{{ .x_login_id }}
	{{ end }}
	</div>
</div>

<div class="title primary-txt">Enter password</div>

{{ template "ERROR" . }}

<input type="hidden" name="x_calling_code" value="{{ .x_calling_code }}">
<input type="hidden" name="x_national_number" value="{{ .x_national_number }}">
<input type="hidden" name="x_login_id" value="{{ .x_login_id }}">

<input id="password" class="input text-input" type="password" name="x_password" placeholder="Password" value="{{ .x_password }}">

<a class="anchor" href="">Forgot Password?</a>

<button class="btn primary-btn" type="submit" name="x_step" value="submit_password">Next</button>

</form>
{{ template "SKYGEAR_LOGO" . }}

</div>
</body>
</html>
`,
}
