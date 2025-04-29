/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"context"

	"github.com/kowabunga-cloud/kowabunga/kowabunga/sdk"
)

func NewTemplateRouter() sdk.Router {
	return sdk.NewTemplateAPIController(&TemplateService{})
}

type TemplateService struct{}

func (s *TemplateService) DeleteTemplate(ctx context.Context, templateId string) (sdk.ImplResponse, error) {
	// ensure template exists
	t, err := FindTemplateByID(templateId)
	if err != nil {
		return HttpNotFound(err)
	}

	// remove template
	err = t.Delete()
	if err != nil {
		return HttpServerError(err)
	}

	return HttpOK(nil)
}

func (s *TemplateService) ListTemplates(ctx context.Context) (sdk.ImplResponse, error) {
	templates := FindTemplates()
	var payload []string
	for _, t := range templates {
		payload = append(payload, t.String())
	}

	return HttpOK(payload)
}

func (s *TemplateService) ReadTemplate(ctx context.Context, templateId string) (sdk.ImplResponse, error) {
	t, err := FindTemplateByID(templateId)
	if err != nil {
		return HttpNotFound(err)
	}

	payload := t.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, templateId string, template sdk.Template) (sdk.ImplResponse, error) {
	LogHttpRequest(RA("templateId", templateId), RA("template", template))

	// check for params
	if template.Name == "" && template.Description == "" {
		return HttpBadParams(nil)
	}

	// ensure template exists
	t, err := FindTemplateByID(templateId)
	if err != nil {
		return HttpNotFound(err)
	}

	// update template
	t.Update(template.Name, template.Description)

	payload := t.Model()
	LogHttpResponse(payload)
	return HttpOK(payload)
}
