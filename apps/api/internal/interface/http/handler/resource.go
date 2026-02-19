package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/jeheskielSunloy77/zeile/internal/application"
	applicationdto "github.com/jeheskielSunloy77/zeile/internal/application/dto"
	"github.com/jeheskielSunloy77/zeile/internal/application/port"
	"github.com/jeheskielSunloy77/zeile/internal/domain"
	httpdto "github.com/jeheskielSunloy77/zeile/internal/interface/http/dto"
	"github.com/jeheskielSunloy77/zeile/internal/interface/http/response"
	httputils "github.com/jeheskielSunloy77/zeile/internal/interface/http/utils"
)

type ResourceHandler[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T], TS httpdto.StoreDTO[S], TU httpdto.UpdateDTO[U]] struct {
	Handler
	resourceName string
	service      application.ResourceService[T, S, U]
}

func NewResourceHandler[T domain.BaseModel, S applicationdto.StoreDTO[T], U applicationdto.UpdateDTO[T], TS httpdto.StoreDTO[S], TU httpdto.UpdateDTO[U]](resourceName string, base Handler, service application.ResourceService[T, S, U]) *ResourceHandler[T, S, U, TS, TU] {
	return &ResourceHandler[T, S, U, TS, TU]{
		resourceName: resourceName,
		Handler:      base,
		service:      service,
	}
}

func (h *ResourceHandler[T, S, U, TS, TU]) Update() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, dto TU) (*T, error) {
		id, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}

		return h.service.Update(c.UserContext(), id, dto.ToUsecase())
	}, http.StatusOK, httpdto.NewDTO[TU]())
}

func (h *ResourceHandler[T, S, U, TS, TU]) GetByID() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*T, error) {
		id, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}

		preloads := port.ParsePreloads(c.Query("preloads"))
		return h.service.GetByID(c.UserContext(), id, preloads)
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *ResourceHandler[T, S, U, TS, TU]) GetMany() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (response.PaginatedResponse[T], error) {
		options := getManyOptionsFromRequest(c)
		entities, total, err := h.service.GetMany(c.UserContext(), options)
		if err != nil {
			return response.PaginatedResponse[T]{}, err
		}

		resp := response.NewPaginatedResponse("Successfully fetched "+h.resourceName+"s!", entities, total, options.Limit, options.Offset)
		return resp, nil
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *ResourceHandler[T, S, U, TS, TU]) Destroy() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*response.Response[T], error) {
		id, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}

		err = h.service.Destroy(c.UserContext(), id)
		if err != nil {
			return nil, err
		}

		resp := response.Response[T]{
			Status:  http.StatusNoContent,
			Success: true,
			Message: "Successfully deleted " + h.resourceName + "!",
		}

		return &resp, nil
	}, http.StatusNoContent, &httpdto.Empty{})
}

func (h *ResourceHandler[T, S, U, TS, TU]) Kill() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*response.Response[T], error) {
		id, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}

		err = h.service.Kill(c.UserContext(), id)
		if err != nil {
			return nil, err
		}

		resp := response.Response[T]{
			Status:  http.StatusNoContent,
			Success: true,
			Message: "Successfully permanently deleted " + h.resourceName + "!",
		}

		return &resp, nil
	}, http.StatusNoContent, &httpdto.Empty{})
}

func (h *ResourceHandler[T, S, U, TS, TU]) Restore() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, _ *httpdto.Empty) (*response.Response[T], error) {
		id, err := httputils.ParseUUIDParam(c.Params("id"))
		if err != nil {
			return nil, err
		}

		preloads := port.ParsePreloads(c.Query("preloads"))
		entity, err := h.service.Restore(c.UserContext(), id, preloads)
		if err != nil {
			return nil, err
		}

		resp := response.Response[T]{
			Status:  http.StatusOK,
			Success: true,
			Message: "Successfully restored " + h.resourceName + "!",
			Data:    entity,
		}

		return &resp, nil
	}, http.StatusOK, &httpdto.Empty{})
}

func (h *ResourceHandler[T, S, U, TS, TU]) Store() fiber.Handler {
	return Handle(h.Handler, func(c *fiber.Ctx, dto TS) (*response.Response[T], error) {
		entity, err := h.service.Store(c.UserContext(), dto.ToUsecase())
		if err != nil {
			return nil, err
		}

		resp := response.Response[T]{
			Status:  http.StatusCreated,
			Success: true,
			Message: "Successfully created " + h.resourceName + "!",
			Data:    entity,
		}

		return &resp, nil
	}, http.StatusCreated, httpdto.NewDTO[TS]())
}

func getManyOptionsFromRequest(c *fiber.Ctx) port.GetManyOptions {
	opts := port.GetManyOptions{
		Limit:          httputils.ParseQueryInt(c.Query("limit")),
		Offset:         httputils.ParseQueryInt(c.Query("offset")),
		Preloads:       port.ParsePreloads(c.Query("preloads")),
		OrderBy:        c.Query("orderBy"),
		OrderDirection: c.Query("orderDirection"),
	}
	opts.Normalize()
	return opts
}
