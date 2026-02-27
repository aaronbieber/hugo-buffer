package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const bufferEndpoint = "https://api.buffer.com/graphql"

const createPostMutation = `
mutation CreatePost($text: String!, $channelId: String!) {
  createPost(input: {
    text: $text,
    channelId: $channelId,
    schedulingType: automatic,
    mode: shareNow
  }) {
    ... on PostActionSuccess {
      post { id }
    }
    ... on MutationError {
      message
    }
  }
}`

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type createPostResponse struct {
	CreatePost struct {
		Post *struct {
			ID string `json:"id"`
		} `json:"post"`
		Message *string `json:"message"`
	} `json:"createPost"`
}

// --- GetOrganizations ---

const getOrganizationsQuery = `
query GetOrganizations {
  account {
    organizations {
      id
      name
      ownerEmail
    }
  }
}`

type Organization struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	OwnerEmail string `json:"ownerEmail"`
}

type getOrganizationsResponse struct {
	Account struct {
		Organizations []Organization `json:"organizations"`
	} `json:"account"`
}

func (c *BufferClient) GetOrganizations(ctx context.Context) ([]Organization, error) {
	var data getOrganizationsResponse
	if err := c.query(ctx, getOrganizationsQuery, nil, &data); err != nil {
		return nil, err
	}
	return data.Account.Organizations, nil
}

// --- GetChannels ---

type Channel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Service       string `json:"service"`
	Avatar        string `json:"avatar"`
	IsQueuePaused bool   `json:"isQueuePaused"`
}

type getChannelsResponse struct {
	Channels []Channel `json:"channels"`
}

func (c *BufferClient) GetChannels(ctx context.Context, organizationID string) ([]Channel, error) {
	// OrganizationId is a custom scalar — inlined to avoid variable type mismatch.
	q := fmt.Sprintf(`
query GetChannels {
  channels(input: {
    organizationId: "%s"
  }) {
    id
    name
    displayName
    service
    avatar
    isQueuePaused
  }
}`, organizationID)

	var data getChannelsResponse
	if err := c.query(ctx, q, nil, &data); err != nil {
		return nil, err
	}
	return data.Channels, nil
}

type BufferClient struct {
	token      string
	httpClient *http.Client
}

func newBufferClient(token string) *BufferClient {
	return &BufferClient{
		token:      token,
		httpClient: &http.Client{},
	}
}

// query executes a GraphQL request and unmarshals data into dest.
func (c *BufferClient) query(ctx context.Context, gql string, vars map[string]any, dest any) error {
	reqBody := gqlRequest{Query: gql, Variables: vars}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bufferEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var gqlResp gqlResponse
	if err := json.Unmarshal(respBytes, &gqlResp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
		return fmt.Errorf("parsing response data: %w", err)
	}

	return nil
}

func (c *BufferClient) CreatePost(ctx context.Context, channelID, text string) (string, error) {
	vars := map[string]any{
		"text":      text,
		"channelId": channelID,
	}

	var data createPostResponse
	if err := c.query(ctx, createPostMutation, vars, &data); err != nil {
		return "", err
	}

	if data.CreatePost.Message != nil {
		return "", fmt.Errorf("mutation error: %s", *data.CreatePost.Message)
	}
	if data.CreatePost.Post == nil {
		return "", fmt.Errorf("unexpected empty post in response")
	}

	return data.CreatePost.Post.ID, nil
}
