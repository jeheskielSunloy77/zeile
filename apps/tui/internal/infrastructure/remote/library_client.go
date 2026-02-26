package remote

import (
	"context"
	"fmt"
	"strings"
)

type BookCatalog struct {
	ID string `json:"id"`
}

type UserLibraryBook struct {
	ID string `json:"id"`
}

type envelope[T any] struct {
	Data *T `json:"data"`
}

func (c *Client) CreateCatalogBook(ctx context.Context, accessToken, title, authors string) (BookCatalog, error) {
	reqBody := struct {
		Title   string `json:"title"`
		Authors string `json:"authors"`
	}{
		Title:   strings.TrimSpace(title),
		Authors: strings.TrimSpace(authors),
	}

	var resp envelope[BookCatalog]
	if err := c.doJSON(ctx, "POST", "/api/v1/library/catalog/books", reqBody, strings.TrimSpace(accessToken), &resp); err != nil {
		return BookCatalog{}, err
	}
	if resp.Data == nil || strings.TrimSpace(resp.Data.ID) == "" {
		return BookCatalog{}, fmt.Errorf("invalid catalog response")
	}
	return *resp.Data, nil
}

func (c *Client) UpsertLibraryBook(ctx context.Context, accessToken, catalogBookID string) (UserLibraryBook, error) {
	reqBody := struct {
		CatalogBookID string `json:"catalogBookId"`
	}{
		CatalogBookID: strings.TrimSpace(catalogBookID),
	}

	var resp UserLibraryBook
	if err := c.doJSON(ctx, "POST", "/api/v1/library/books", reqBody, strings.TrimSpace(accessToken), &resp); err != nil {
		return UserLibraryBook{}, err
	}
	if strings.TrimSpace(resp.ID) == "" {
		return UserLibraryBook{}, fmt.Errorf("invalid library book response")
	}
	return resp, nil
}

func (c *Client) UpsertReadingState(ctx context.Context, accessToken, libraryBookID, mode string, locator map[string]any, progressPercent float64) error {
	reqBody := struct {
		LocatorJSON     map[string]any `json:"locatorJson"`
		ProgressPercent float64        `json:"progressPercent"`
	}{
		LocatorJSON:     locator,
		ProgressPercent: progressPercent,
	}

	path := fmt.Sprintf("/api/v1/library/books/%s/reading-states/%s", strings.TrimSpace(libraryBookID), strings.TrimSpace(mode))
	return c.doJSON(ctx, "PUT", path, reqBody, strings.TrimSpace(accessToken), nil)
}

