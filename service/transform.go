package service

import (
	"fmt"

	"kopiq/internal/capture"
	"kopiq/internal/store"
	"kopiq/internal/transform"
)

// TransformService exposes KQ-016 transforms to the UI.
type TransformService struct {
	Store    *store.Store
	Pipeline *capture.Pipeline
	reg      *transform.Registry
}

func (t *TransformService) registry() *transform.Registry {
	if t.reg == nil {
		t.reg = transform.Default()
	}
	return t.reg
}

// ListActions returns transforms that apply to the item.
func (t *TransformService) ListActions(itemID int64) ([]transform.Action, error) {
	item, err := t.Store.GetRaw(itemID)
	if err != nil {
		return nil, err
	}
	return t.registry().ActionsFor(item), nil
}

// Apply runs a transform and returns the result (does not write history).
func (t *TransformService) Apply(itemID int64, transformID string) (transform.Result, error) {
	item, err := t.Store.GetRaw(itemID)
	if err != nil {
		return transform.Result{}, err
	}
	return t.registry().Apply(item, transformID), nil
}

// ApplyAndCopy runs a transform and writes the payload/view text to the clipboard (self-owned).
func (t *TransformService) ApplyAndCopy(itemID int64, transformID string) (transform.Result, error) {
	item, err := t.Store.GetRaw(itemID)
	if err != nil {
		return transform.Result{}, err
	}
	if item.Sensitivity == "suspected" || item.Masked {
		return transform.Result{}, fmt.Errorf("suspected items cannot be transformed onto the clipboard")
	}
	res := t.registry().Apply(item, transformID)
	if (res.Kind != "payload" && res.Kind != "view") || res.Text == "" {
		if res.Error != "" {
			return res, fmt.Errorf("%s", res.Error)
		}
		return res, fmt.Errorf("transform produced no text")
	}
	if t.Pipeline == nil {
		return res, fmt.Errorf("capture pipeline not ready")
	}
	if err := t.Pipeline.WriteText(res.Text); err != nil {
		return res, err
	}
	return res, nil
}
