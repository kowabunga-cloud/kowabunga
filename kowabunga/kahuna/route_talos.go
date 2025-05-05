/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

// func NewTalosRouter() sdk.Router {
// 	return sdk.NewTalosAPIController(&TalosService{})
// }

// type TalosService struct{}

// func (s *TalosService) DeleteTalos(ctx context.Context, talosId string) (sdk.ImplResponse, error) {
// 	// ensure Talos exists
// 	k, err := FindTalosByID(talosId)
// 	if err != nil {
// 		return HttpNotFound(err)
// 	}

// 	if k.HasChildren() {
// 		return HttpConflict(fmt.Errorf("Talos has child resources that must be destroyed first."))
// 	}
// 	// remove Talos
// 	err = k.Delete()
// 	if err != nil {
// 		return HttpServerError(err)
// 	}

// 	return HttpOK(nil)
// }

// func (s *TalosService) ListTaloses(ctx context.Context) (sdk.ImplResponse, error) {
// 	taloses := FindTalos()
// 	var payload []string
// 	for _, t := range taloses {
// 		payload = append(payload, t.String())
// 	}

// 	return HttpOK(payload)
// }

// func (s *TalosService) ReadTalos(ctx context.Context, talosId string) (sdk.ImplResponse, error) {
// 	k, err := FindTalosByID(talosId)
// 	if err != nil {
// 		return HttpNotFound(err)
// 	}

// 	payload := k.Model()
// 	LogHttpResponse(payload)
// 	return HttpOK(payload)
// }

// func (s *TalosService) UpdateTalos(ctx context.Context, talosId string, kawaii sdk.Talos) (sdk.ImplResponse, error) {
// 	LogHttpRequest(RA("talosId", talosId), RA("kawaii", kawaii))

// 	// Get our kawaii
// 	gw, err := FindTalosByID(talosId)
// 	if err != nil {
// 		return HttpNotFound(err)
// 	}

// 	payload := gw.Model()
// 	LogHttpResponse(payload)
// 	return HttpOK(payload)
// }
 