package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/retailcrm/api-client-go/errs"
	"github.com/retailcrm/api-client-go/v5"
)

func buildIntegrationModule(code, name string) v5.IntegrationModule {
	return v5.IntegrationModule{
		Code:            code,
		IntegrationCode: code,
		Active:          true,
		Name:            name,
		ClientID:        code,
		BaseURL:         "https://example.com",
		Integrations: &v5.Integrations{
			MgBot: &v5.MgBot{},
		},
	}
}

func updateIntegrationModule(apiURL, apiKey, code, name string) (string, string, error) {
	client := v5.New(apiURL, apiKey)
	resp, _, err := client.IntegrationModuleEdit(buildIntegrationModule(code, name))
	if err != nil {
		if nErr := normalizeAPIError(err); nErr != nil {
			return "", "", nErr
		}
	}
	return strings.TrimRight(resp.Info.MgBotInfo.EndpointUrl, "/"), resp.Info.MgBotInfo.Token, nil
}

func normalizeAPIError(err *errs.Failure) error {
	if err == nil {
		return nil
	}

	if err.Error() != "" {
		return errors.New(err.Error())
	}

	if err.ApiError() != "" {
		return errors.New(err.ApiError())
	}

	if len(err.ApiErrors()) > 0 {
		var sb strings.Builder
		sb.Grow(128)

		for field, value := range err.ApiErrors() {
			sb.WriteString(fmt.Sprintf("[%s: %s]", field, value))
		}

		return errors.New(sb.String())
	}

	return nil
}
