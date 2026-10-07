package response

import "github.com/gofiber/fiber/v2"

// Meta berisi informasi paginasi (sesuai definisi soal UTS).
type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}

// Body adalah amplop baku untuk semua respons JSON.
type Body struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func OK(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(Body{Success: true, Message: message, Data: data})
}

func OKList(c *fiber.Ctx, message string, data any, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(Body{Success: true, Message: message, Data: data, Meta: meta})
}

// Created -> 201 disertai header Location ke sumber daya baru.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set(fiber.HeaderLocation, location)
	return c.Status(fiber.StatusCreated).JSON(Body{Success: true, Message: message, Data: data})
}

// NoContent -> 204: berhasil, tanpa body.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(Body{Success: false, Message: message})
}

// FailWithErrors dipakai untuk error yang membawa rincian per field (mis. 422 SKS/kuota).
func FailWithErrors(c *fiber.Ctx, status int, message string, errs map[string][]string) error {
	body := Body{Success: false, Message: message}
	if len(errs) > 0 {
		body.Errors = errs
	}
	return c.Status(status).JSON(body)
}

// FailValidation -> 422 dengan rincian per field.
func FailValidation(c *fiber.Ctx, errs map[string][]string) error {
	return FailWithErrors(c, fiber.StatusUnprocessableEntity, "Validasi gagal", errs)
}
