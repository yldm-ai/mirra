package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
)

// Connection test states.
type connState int

const (
	connUntested connState = iota
	connTesting
	connOK
	connFailed
)

// CheckConnection checks the address and the key before generating, with a request the provider answers cheaply: its list of models where it has one, OpenRouter's key, whose models anyone may list, and otherwise an empty generation, which a valid key gets a 400 for and an invalid one a 401 or 403.
func CheckConnection(ctx context.Context, client *http.Client, cfg Config) error {
	var req *http.Request
	var err error
	emptyPost := false
	switch cfg.Protocol {
	case protoSeedream:
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint(cfg, "/images/generations"), strings.NewReader("{}"))
		emptyPost = true
	case protoDashScope:
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint(cfg, dashScopePath), strings.NewReader("{}"))
		emptyPost = true
	case protoOpenRouter:
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint(cfg, "/key"), nil)
	default:
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpoint(cfg, "/models"), nil)
	}
	if err != nil {
		return err
	}
	if emptyPost {
		req.Header.Set("Content-Type", "application/json")
	}
	authorize(req, cfg)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 == 2 || emptyPost && resp.StatusCode == http.StatusBadRequest {
		return nil
	}
	return apiError(resp)
}

// authorize puts cfg's key on req in the header its provider reads; none without a key.
func authorize(req *http.Request, cfg Config) {
	key := strings.TrimSpace(cfg.APIKey)
	switch {
	case key == "":
	case cfg.Protocol == protoGemini:
		req.Header.Set("x-goog-api-key", key)
	case cfg.Protocol == protoAzure:
		req.Header.Set("api-key", key)
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

// canListModels says whether the app can list the image models a provider offers with its key: not Model Studio, whose listing is on a host of the user's workspace and region, nor Azure, whose model field is a deployment's name that a key cannot list.
func canListModels(p provider) bool {
	return p.Protocol != protoDashScope && !p.Deployment
}

// imageModelNames are the words of image models' IDs, for the lists that don't say which models make images: OpenAI's gpt-image and dall-e, Gemini's image, nano-banana and imagen, and the open models an OpenAI-compatible server may serve.
var imageModelNames = []string{"image", "dall-e", "banana", "imagen", "imagine", "seedream", "flux", "stable-diffusion", "sdxl", "sd-", "sd3", "kolors", "hidream"}

func looksLikeImageModel(id string) bool {
	id = strings.ToLower(id)
	return slices.ContainsFunc(imageModelNames, func(w string) bool { return strings.Contains(id, w) })
}

// ListModels returns the IDs of the image models cfg's key can use, sorted: xAI's image generation models, OpenRouter's models with image output, Ark's image generation domain, and the models of Gemini and of OpenAI and compatible servers whose names are an image model's, every model when none is. A server the user runs, a local model server or a gateway, lists what it serves, whose names need not be an image model's, so every one of them is offered.
func ListModels(ctx context.Context, client *http.Client, cfg Config) ([]string, error) {
	var ids []string
	switch cfg.Protocol {
	case protoXAI:
		var out struct {
			Models []struct {
				ID string `json:"id"`
			} `json:"models"`
		}
		if err := getJSON(ctx, client, cfg, "/image-generation-models", &out); err != nil {
			return nil, err
		}
		for _, m := range out.Models {
			ids = append(ids, m.ID)
		}
	case protoOpenRouter:
		var out struct {
			Data []struct {
				ID           string `json:"id"`
				Architecture struct {
					OutputModalities []string `json:"output_modalities"`
				} `json:"architecture"`
			} `json:"data"`
		}
		if err := getJSON(ctx, client, cfg, "/models?output_modalities=image", &out); err != nil {
			return nil, err
		}
		for _, m := range out.Data {
			if slices.Contains(m.Architecture.OutputModalities, "image") {
				ids = append(ids, m.ID)
			}
		}
	case protoSeedream:
		var out struct {
			Data []struct {
				ID         string `json:"id"`
				Domain     string `json:"domain"`
				Modalities struct {
					Output []string `json:"output_modalities"`
				} `json:"modalities"`
			} `json:"data"`
		}
		if err := getJSON(ctx, client, cfg, "/models", &out); err != nil {
			return nil, err
		}
		for _, m := range out.Data {
			if m.Domain == "ImageGeneration" || slices.Contains(m.Modalities.Output, "image") {
				ids = append(ids, m.ID)
			}
		}
	case protoGemini:
		var out struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := getJSON(ctx, client, cfg, "/models?pageSize=1000", &out); err != nil {
			return nil, err
		}
		for _, m := range out.Models {
			if id := strings.TrimPrefix(m.Name, "models/"); looksLikeImageModel(id) {
				ids = append(ids, id)
			}
		}
	default:
		var out struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := getJSON(ctx, client, cfg, "/models", &out); err != nil {
			return nil, err
		}
		var all []string
		for _, m := range out.Data {
			all = append(all, m.ID)
			// A server the user runs serves what it lists, a local model or a gateway's route, whose name need not be an image model's; a cloud service's catalog mixes its models, so only the images' names are.
			if cfg.Local || looksLikeImageModel(m.ID) {
				ids = append(ids, m.ID)
			}
		}
		if len(ids) == 0 {
			ids = all
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids), nil
}

// getJSON gets path under cfg's address with its key, as CheckConnection sends it, and decodes the answer into v.
func getJSON(ctx context.Context, client *http.Client, cfg Config, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint(cfg, path), nil)
	if err != nil {
		return err
	}
	authorize(req, cfg)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return apiError(resp)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxIndexBytes)).Decode(v)
}
