package handler

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/kern/internal/application"
	"github.com/jeheskielSunloy77/kern/internal/domain"
	httpdto "github.com/jeheskielSunloy77/kern/internal/interface/http/dto"
	"github.com/jeheskielSunloy77/kern/internal/interface/http/response"
	httputils "github.com/jeheskielSunloy77/kern/internal/interface/http/utils"
)

type CommunityHandler struct {
	Handler
	service application.CommunityService
}

func NewCommunityHandler(h Handler, service application.CommunityService) *CommunityHandler {
	return &CommunityHandler{Handler: h, service: service}
}

func (h *CommunityHandler) ListBooks() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (response.PaginatedResponse[domain.CommunityBook], error) {
		limit := httputils.ParseQueryInt(c.Query("limit"), 100, 20)
		offset := httputils.ParseQueryInt(c.Query("offset"), 10000, 0)
		books, total, err := h.service.ListBooks(
			c.UserContext(),
			strings.TrimSpace(c.Query("q")),
			strings.TrimSpace(c.Query("ownerUsername")),
			limit,
			offset,
		)
		if err != nil {
			return response.PaginatedResponse[domain.CommunityBook]{}, err
		}
		resp := response.NewPaginatedResponse("Community books fetched successfully.", books, total, limit, offset)
		return resp, nil
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *CommunityHandler) GetBook() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*domain.CommunityBook, error) {
		libraryBookID, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}
		return h.service.GetBook(c.UserContext(), libraryBookID)
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *CommunityHandler) SaveBook() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*domain.UserLibraryBook, error) {
		userID, err := parseUserIDFromContext(c)
		if err != nil {
			return nil, err
		}
		libraryBookID, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}
		return h.service.SaveBook(c.UserContext(), userID, libraryBookID)
	}, http.StatusOK, &httpdto.Empty{})
}
