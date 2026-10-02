package service

import (
	"fmt"
	"strings"

	"kopiq/internal/domain"
	"kopiq/internal/store"
)

// OrganizeService exposes collections and snippets (KQ-017).
type OrganizeService struct {
	Store *store.Store
	Emit  func(domain.HistoryChangedEvent)
}

func (o *OrganizeService) emit(id int64, action string) {
	if o.Emit == nil {
		return
	}
	o.Emit(domain.HistoryChangedEvent{ItemID: id, Action: action})
}

// ListCollections returns all collections.
func (o *OrganizeService) ListCollections() ([]domain.Collection, error) {
	return o.Store.ListCollections()
}

// CreateCollection creates a named collection.
func (o *OrganizeService) CreateCollection(name string) (domain.Collection, error) {
	c, err := o.Store.CreateCollection(name)
	if err != nil {
		return c, err
	}
	o.emit(c.ID, "collection-create")
	return c, nil
}

// RenameCollection renames a collection.
func (o *OrganizeService) RenameCollection(id int64, name string) error {
	if err := o.Store.RenameCollection(id, name); err != nil {
		return err
	}
	o.emit(id, "collection-rename")
	return nil
}

// DeleteCollection removes a collection (items stay in history).
func (o *OrganizeService) DeleteCollection(id int64) error {
	if err := o.Store.DeleteCollection(id); err != nil {
		return err
	}
	o.emit(id, "collection-delete")
	return nil
}

// AddToCollection adds a history item to a collection.
func (o *OrganizeService) AddToCollection(collectionID, itemID int64) error {
	if err := o.Store.AddToCollection(collectionID, itemID); err != nil {
		return err
	}
	o.emit(itemID, "collection-add")
	return nil
}

// RemoveFromCollection removes an item from a collection.
func (o *OrganizeService) RemoveFromCollection(collectionID, itemID int64) error {
	if err := o.Store.RemoveFromCollection(collectionID, itemID); err != nil {
		return err
	}
	o.emit(itemID, "collection-remove")
	return nil
}

// ListCollectionItems returns items in a collection.
func (o *OrganizeService) ListCollectionItems(collectionID int64) ([]domain.ClipboardItem, error) {
	return o.Store.ListCollectionItems(collectionID, 300)
}

// ListSnippets returns saved snippets.
func (o *OrganizeService) ListSnippets() ([]domain.Snippet, error) {
	return o.Store.ListSnippets(300)
}

// SaveSnippetFromItem creates a snippet from a clipboard item.
// alias may include a leading colon; it is normalized.
func (o *OrganizeService) SaveSnippetFromItem(itemID int64, title, alias string) (domain.Snippet, error) {
	sn, err := o.Store.CreateSnippetFromItem(itemID, title, alias)
	if err != nil {
		return sn, err
	}
	o.emit(sn.ID, "snippet-create")
	return sn, nil
}

// DeleteSnippet removes a snippet.
func (o *OrganizeService) DeleteSnippet(id int64) error {
	if err := o.Store.DeleteSnippet(id); err != nil {
		return err
	}
	o.emit(id, "snippet-delete")
	return nil
}

// SetSnippetAlias sets the primary alias for a snippet.
func (o *OrganizeService) SetSnippetAlias(id int64, alias string) error {
	if err := o.Store.SetSnippetAlias(id, alias); err != nil {
		return err
	}
	o.emit(id, "snippet-alias")
	return nil
}

// FindSnippetByAlias resolves :alias or alias.
func (o *OrganizeService) FindSnippetByAlias(alias string) (*domain.Snippet, error) {
	return o.Store.FindSnippetByAlias(alias)
}

// EnsureDefaultCollection creates a collection if the name is new, else returns existing.
func (o *OrganizeService) EnsureCollection(name string) (domain.Collection, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Collection{}, fmt.Errorf("name required")
	}
	list, err := o.Store.ListCollections()
	if err != nil {
		return domain.Collection{}, err
	}
	for _, c := range list {
		if strings.EqualFold(c.Name, name) {
			return c, nil
		}
	}
	return o.CreateCollection(name)
}
