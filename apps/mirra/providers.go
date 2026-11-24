package main

import (
	"errors"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
	ailogo "github.com/yldm-tech/ai-logo/nsgo"
)

// protocol is the shape of a provider's image API; Generate and CheckConnection speak each one.
type protocol int

const (
	// protoOpenAI is OpenAI's multipart /images/edits, which OpenAI-compatible servers copy.
	protoOpenAI protocol = iota
	// protoAzure is protoOpenAI on Azure OpenAI's v1 API, which takes the key in an api-key header.
	protoAzure
	protoGemini
	protoSeedream
	protoXAI
	protoOpenRouter
	protoDashScope
)

// provider is a preset AI service: where its API is, the model and size it starts with, and where to get a key.
type provider struct {
	ID string
	// Name is the brand's name, the same in every language unless the provider has a local one.
	Name  string
	Local func(t locales.Messages) string
	// Logo is the provider's mark in github.com/yldm-tech/ai-logo; "" draws a server.
	Logo     string
	Protocol protocol
	// BaseURL is the API's address; "" asks the user for theirs, as Azure's names their resource.
	BaseURL string
	// URLHint shows in an empty address field when BaseURL is "".
	URLHint string
	Model   string
	Size    string
	// KeyURL is where the user makes a key; "" for a server they run.
	KeyURL string
	// KeyHint is how the provider's keys start, shown in the empty key field; "" where they have no fixed form.
	KeyHint string
	// KeyOptional marks a service that may take no key, such as a server the user runs.
	KeyOptional bool
	// OwnServer marks software the user runs on their own machine, which costs nothing a request.
	OwnServer bool
	// Gateway marks software the user runs that passes the request on to a service behind it, whose plan pays for it.
	Gateway bool
	// Deployment marks a service whose model is a deployment the user named, as Azure's is: it shows with the key, as the deployment's name, rather than among the advanced settings.
	Deployment bool
}

// name is what the app calls the provider in the user's language.
func (p provider) name(t locales.Messages) string {
	if p.Local != nil {
		return p.Local(t)
	}
	return p.Name
}

// icon is the provider's mark, in color where the brand has one.
func (p provider) icon() *core.SVG {
	if p.Logo == "" {
		return lucide.Server()
	}
	if svg, err := ailogo.Parse(p.Logo, ailogo.Color); err == nil {
		return svg
	}
	return ailogo.MustParse(p.Logo, ailogo.Mono)
}

// keyAccount is the secret account the provider's key is kept under.
func (p provider) keyAccount() string { return "api-key." + p.ID }

// storageKey is where one of the provider's settings is kept.
func (p provider) storageKey(field string) string { return "provider." + p.ID + "." + field }

// Provider IDs the app refers to by name.
const (
	providerOpenAI = "openai"
	providerCustom = "custom"
)

// providers are the presets in the order the settings list them: cloud services, then servers the user runs.
var providers = []provider{
	{ID: providerOpenAI, Name: "OpenAI", Logo: "openai", Protocol: protoOpenAI, BaseURL: defaultBaseURL, Model: defaultModel, Size: defaultSize,
		KeyURL: "https://platform.openai.com/api-keys", KeyHint: "sk-…"},
	{ID: "gemini", Name: "Google Gemini", Logo: "gemini", Protocol: protoGemini, BaseURL: "https://generativelanguage.googleapis.com/v1beta", Model: "gemini-nano-banana-2.1", Size: defaultSize,
		KeyURL: "https://aistudio.google.com/apikey", KeyHint: "AIza…"},
	{ID: "volcengine", Name: "Volcengine Ark", Local: func(t locales.Messages) string { return t.ProviderVolcengine() }, Logo: "volcengine", Protocol: protoSeedream,
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3", Model: "doubao-seedream-5-5-261028", Size: defaultSize,
		KeyURL: "https://ark.volcengine.com/region:cn-beijing/apiKey"},
	{ID: "xai", Name: "xAI", Logo: "xai", Protocol: protoXAI, BaseURL: "https://api.x.ai/v1", Model: "grok-imagine-image-2.0", Size: defaultSize,
		KeyURL: "https://console.x.ai/team/default/api-keys", KeyHint: "xai-…"},
	{ID: "openrouter", Name: "OpenRouter", Logo: "openrouter", Protocol: protoOpenRouter, BaseURL: "https://openrouter.ai/api/v1", Model: "bytedance-seed/seedream-4.5", Size: defaultSize,
		KeyURL: "https://openrouter.ai/keys", KeyHint: "sk-or-…"},
	{ID: "everyapi", Name: "EveryAPI", Logo: "everyapi", Protocol: protoOpenAI, BaseURL: "https://api.everyapi.ai/v1", Model: defaultModel, Size: defaultSize,
		KeyURL: "https://app.everyapi.ai", KeyHint: "sk-everyapi-…"},
	{ID: "dashscope", Name: "Alibaba Cloud Model Studio", Local: func(t locales.Messages) string { return t.ProviderDashScope() }, Logo: "bailian", Protocol: protoDashScope,
		BaseURL: "https://dashscope.aliyuncs.com/api/v1", Model: "qwen-image-2.0-pro", Size: defaultSize,
		KeyURL: "https://bailian.console.aliyun.com/cn-beijing/model/settings/api-key", KeyHint: "sk-…"},
	{ID: "azure", Name: "Azure OpenAI", Logo: "azure", Protocol: protoAzure, URLHint: "https://<resource>.openai.azure.com/openai/v1", Model: defaultModel, Size: defaultSize, Deployment: true,
		KeyURL: "https://portal.azure.com/#view/Microsoft_Azure_ProjectOxford/CognitiveServicesHub/~/OpenAI"},
	{ID: "sdcpp", Name: "stable-diffusion.cpp", Protocol: protoOpenAI, BaseURL: "http://127.0.0.1:1234/v1", Model: "sd-cpp-local", Size: defaultSize, KeyOptional: true, OwnServer: true},
	// Magpie serves the image models of the accounts the user is signed into, a ChatGPT plan's GPT Image among them, from one local address, taking any key. It passes the request on to the service behind it, so it is a gateway, not a server of its own.
	{ID: "magpie", Name: "Magpie", Protocol: protoOpenAI, BaseURL: "http://127.0.0.1:3425/v1", Model: defaultModel, Size: defaultSize, KeyOptional: true, Gateway: true},
	{ID: providerCustom, Name: "OpenAI-compatible", Local: func(t locales.Messages) string { return t.ProviderCustom() }, Protocol: protoOpenAI, URLHint: defaultLocalURL,
		Model: defaultModel, Size: defaultSize, KeyOptional: true},
}

// providerByID returns the preset with id, OpenAI's when there is none.
func providerByID(id string) provider {
	for _, p := range providers {
		if p.ID == id {
			return p
		}
	}
	return providers[0]
}

// privacyLine is what the settings say about where the requests go: a browser keeps no key, a gateway passes the request on to a service behind it, a server the user runs answers it, and a cloud service is asked directly.
func privacyLine(p provider, keyKept bool, t locales.Messages) string {
	switch {
	case !keyKept:
		return t.PrivacyWeb()
	case p.Gateway:
		return t.PrivacyGateway()
	case p.OwnServer:
		return t.PrivacyLocal()
	}
	return t.PrivacyCloud()
}

// providerSettings are what the user set for one provider: "" keeps the preset's address, model or size.
type providerSettings struct {
	url, key, model, size *core.Signal[string]
	// models are the image models the key can use, nil until listed; listing is on while they are fetched, and modelsErr says why they couldn't be.
	models    *core.Signal[[]string]
	listing   *core.Signal[bool]
	modelsErr *core.Signal[string]
}

// Storage keys of the settings before providers, read once to carry them over.
const (
	legacyService    = "service"
	legacyCloudURL   = "baseURL"
	legacyLocalURL   = "localURL"
	legacyModel      = "model"
	legacySize       = "size"
	legacyKeyAccount = "api-key"
	legacyLocalKey   = "local-api-key"
	keyProvider      = "provider"
)

// The sizes the settings offer: portrait, which suits a full-length look, square and landscape.
const (
	sizePortrait  = "1024x1536"
	sizeSquare    = "1024x1024"
	sizeLandscape = "1536x1024"
)

// defaultLocalURL is the address an OpenAI-compatible server on the user's machine starts with.
const defaultLocalURL = "http://127.0.0.1:8080/v1"

// migrateSettings carries the cloud or local service set up before providers over to the provider it was: OpenAI at its own address, any other address as an OpenAI-compatible server. It returns the provider and the secret account its key was kept under, "" when there is nothing to carry over.
func migrateSettings(store core.Storage) (id, keyAccount string) {
	if _, ok := store.GetItem(keyProvider); ok {
		return "", ""
	}
	local := false
	if v, _ := store.GetItem(legacyService); v == "local" {
		local = true
	}
	urlKey, account := legacyCloudURL, legacyKeyAccount
	if local {
		urlKey, account = legacyLocalURL, legacyLocalKey
	}
	url, hadURL := store.GetItem(urlKey)
	p := providerByID(providerOpenAI)
	if local || (hadURL && strings.TrimRight(strings.TrimSpace(url), "/") != defaultBaseURL) {
		p = providerByID(providerCustom)
		if !hadURL {
			url = defaultLocalURL
		}
		_ = store.SetItem(p.storageKey("url"), url)
	}
	if v, ok := store.GetItem(legacyModel); ok && v != p.Model {
		_ = store.SetItem(p.storageKey("model"), v)
	}
	if v, ok := store.GetItem(legacySize); ok && v != p.Size {
		_ = store.SetItem(p.storageKey("size"), v)
	}
	_ = store.SetItem(keyProvider, p.ID)
	return p.ID, account
}

// readKeys reads every provider's key, by provider ID. A key kept before providers moves to the provider migrateSettings carried it to, under that provider's account; the old account stays, so an older version still finds it. The error is core.ErrSecretsUnavailable where there is no secure store.
func readKeys(secrets core.Secrets, migrated, legacyAccount string) (map[string]string, error) {
	keys := map[string]string{}
	for _, p := range providers {
		key, ok, err := secrets.Get(p.keyAccount())
		if errors.Is(err, core.ErrSecretsUnavailable) {
			return nil, err
		}
		if err == nil && ok {
			keys[p.ID] = key
		}
	}
	if migrated != "" && keys[migrated] == "" {
		if key, ok, err := secrets.Get(legacyAccount); err == nil && ok && key != "" {
			keys[migrated] = key
			_ = secrets.Set(providerByID(migrated).keyAccount(), key)
		}
	}
	return keys, nil
}
