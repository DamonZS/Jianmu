package agent

import (
	"bytes"
	"cmp"
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DamonZS/Jianmu/internal/edit"
	"github.com/DamonZS/Jianmu/internal/gateway"
	"github.com/DamonZS/Jianmu/internal/provider"
)

// Pi's codemode calls classifier and image models (models.classify,
// models.generateImages; Pi's docs/models.md "Use classifier models" and
// "Use image models"), wangbinyq on Discord. Pi's models.json defines chat
// models only: its model definitions have no type, and a custom provider's
// classifier or image models come from an extension alone
// (docs/custom-provider.md, "Register mixed-operation models"). So jianmu
// writes one, extensions/jianmu.ts in Pi's agent folder, which Pi loads by
// itself. It lists jianmu's decision models as provider jianmu-systemone,
// on Pi's own System One client against jianmu's /v1/systemone, and its
// image models as jianmu-images, drawn by /v1/images/generations. It is
// there while models.json has providers.jianmu, and gone with it.

//go:embed pi_extension.ts
var piExtensionSource string

// piExtensionMark opens the file jianmu writes. A file at that path
// without it is the user's own, and is never written over or removed.
const piExtensionMark = "// jianmu: written by jianmu"

// piExtensionPath is jianmu's extension in Pi's agent folder dir.
func piExtensionPath(dir string) string {
	return filepath.Join(dir, "extensions", "jianmu.ts")
}

// piExtensionModel is a model as the extension registers it.
type piExtensionModel struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Input []string `json:"input,omitempty"`
}

// piExtensionFor is the extension's source for a gateway at baseURL (its
// /v1) and key; "" when jianmu has no decision or image model to list.
func piExtensionFor(baseURL, key string) string {
	ds := provider.OfferedDeciders(nil)
	for i := range ds {
		ds[i].Name = cmp.Or(ds[i].Name, ds[i].Model)
	}
	labels := provider.LabelsFor("pi", ds)
	classifiers := make([]piExtensionModel, 0, len(ds))
	for i, e := range ds {
		classifiers = append(classifiers, piExtensionModel{ID: e.ID, Name: labels[i]})
	}
	images := []piExtensionModel{}
	for _, m := range gateway.ImageModels() {
		in := []string{"text"}
		if m.Images {
			in = append(in, "image")
		}
		images = append(images, piExtensionModel{ID: m.ID, Name: m.Name, Input: in})
	}
	if len(classifiers) == 0 && len(images) == 0 {
		return ""
	}
	data, _ := json.Marshal(map[string]any{
		"baseUrl": baseURL, "apiKey": key,
		"classifiers": classifiers, "images": images,
	})
	return strings.Replace(piExtensionSource, "__JIANMU__", string(data), 1)
}

// piExtensionSynced brings jianmu's extension in Pi's agent folder dir
// to what models.json says: written while it has providers.jianmu and
// jianmu has models to list, else gone. A file there jianmu didn't write
// is left as it is.
func piExtensionSynced(dir, modelsPath, baseURL, key string) error {
	path := piExtensionPath(dir)
	cur, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		cur = nil
	case err != nil:
		return err // unknown: neither written over nor removed
	case !bytes.HasPrefix(edit.TrimBOM(cur), []byte(piExtensionMark)):
		return nil // the user's own
	}
	want := ""
	if _, on := edit.GetJSON(modelsPath, "providers."+jianmuID); on {
		want = piExtensionFor(baseURL, key)
	}
	if want == "" {
		if cur == nil {
			return nil
		}
		return os.Remove(path)
	}
	if string(cur) == want {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return edit.WriteAtomic(path, []byte(want))
}
